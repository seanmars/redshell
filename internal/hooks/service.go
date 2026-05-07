package hooks

import (
	"errors"
	"fmt"
	"os"
)

// ErrUnknownAgent is returned when an unsupported agentID is supplied.
var ErrUnknownAgent = errors.New("unknown agent")

// Service is the façade bound to the Wails layer. It owns the resolved
// home directory and dispatches per-agent loading.
type Service struct {
	home string
}

// NewService resolves os.UserHomeDir and returns a production Service.
func NewService() (*Service, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return NewServiceWithRoot(home), nil
}

// NewServiceWithRoot builds a Service rooted at the supplied home
// directory. Tests pass a t.TempDir() result.
func NewServiceWithRoot(home string) *Service {
	return &Service{home: home}
}

// ListHooks loads every loadable source for agentID and returns the
// flattened Listing. Errors loading individual sources are surfaced via
// Listing.Errors and do not abort the call.
//
// opts.Workspace is reserved for the future B-route; v1 ignores any
// non-empty value without erroring.
func (s *Service) ListHooks(agentID string, opts ListOpts) (Listing, error) {
	switch agentID {
	case "claude":
		return s.listClaude(), nil
	case "copilot":
		return s.listCopilot(opts), nil
	default:
		return Listing{}, fmt.Errorf("%w: %s", ErrUnknownAgent, agentID)
	}
}

func (s *Service) listClaude() Listing {
	// Initialize slices so the JSON contract is always arrays (never null),
	// which protects the frontend's `.map(...)` from a TypeError when the
	// user has zero hook files of any kind.
	listing := Listing{
		AgentID:    "claude",
		Sources:    []Source{},
		Hooks:      []Hook{},
		Errors:     []SourceError{},
		DisableAll: []DisableFlag{},
	}

	// User scope.
	userPath := ClaudeUserSettingsPath(s.home)
	loadClaudeFileInto(&listing, "user", SourceUser, "User", userPath, "")

	// Local scope.
	localPath := ClaudeLocalSettingsPath(s.home)
	loadClaudeFileInto(&listing, "local", SourceLocal, "Local", localPath, "")

	// Plugin scope (multiple sources, pre-sorted alphabetic by Label).
	candidates, err := scanPluginCandidates(s.home)
	if err != nil {
		// Register a synthetic source so the error has a parent group in
		// the source tree; otherwise the message would be dropped on the
		// floor by the frontend's per-source error lookup.
		const pluginsSourceID = "plugins"
		path := ClaudeInstalledPluginsPath(s.home)
		listing.Sources = append(listing.Sources, Source{
			ID:    pluginsSourceID,
			Kind:  SourcePlugin,
			Path:  path,
			Label: "Plugin index",
		})
		listing.Errors = append(listing.Errors, SourceError{
			SourceID: pluginsSourceID,
			Path:     path,
			Message:  err.Error(),
		})
	}
	for i, c := range candidates {
		sourceID := fmt.Sprintf("plugin-%d", i)
		loadClaudeFileInto(&listing, sourceID, SourcePlugin, c.Label, c.HookFile, c.Key)
	}

	annotateDuplicates(&listing)
	return listing
}

// loadClaudeFileInto loads one Claude-shape file into the Listing,
// appending Source/Hook/DisableFlag/SourceError entries as appropriate.
// Empty/missing sources are silently skipped (do not register a Source).
func loadClaudeFileInto(listing *Listing, sourceID string, kind SourceKind, label, path, pluginKey string) {
	source := Source{ID: sourceID, Kind: kind, Path: path, Label: label, PluginKey: pluginKey}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		listing.Errors = append(listing.Errors, SourceError{
			SourceID: sourceID,
			Path:     path,
			Message:  err.Error(),
		})
		// Still register the source so the user sees the error in the tree.
		listing.Sources = append(listing.Sources, source)
		return
	}

	res, parseErr := parseClaudeFile(data, sourceID)
	if parseErr != nil {
		listing.Errors = append(listing.Errors, SourceError{
			SourceID: sourceID,
			Path:     path,
			Message:  parseErr.Error(),
		})
		listing.Sources = append(listing.Sources, source)
		return
	}

	if len(res.Hooks) == 0 && !res.DisableAll {
		// Empty source: hide entirely per the spec ("Empty source is hidden").
		return
	}

	listing.Sources = append(listing.Sources, source)
	listing.Hooks = append(listing.Hooks, res.Hooks...)
	if res.DisableAll {
		listing.DisableAll = append(listing.DisableAll, DisableFlag{
			SourceID: sourceID,
			Path:     path,
		})
	}
}

// listCopilot returns the v1 Copilot empty-state shape. Workspace is
// ignored. The frontend dispatches on EmptyReason to render the
// "project-scoped hooks coming later" empty state.
func (s *Service) listCopilot(_ ListOpts) Listing {
	return Listing{
		AgentID:     "copilot",
		EmptyReason: EmptyReasonCopilotProjectScoped,
	}
}

// annotateDuplicates fills Hook.DupCount with the number of distinct
// sources that contain a hook with the same dedup key. The implementation
// counts unique source IDs per key so two duplicates inside the same
// source do not inflate the chip.
func annotateDuplicates(listing *Listing) {
	if len(listing.Hooks) < 2 {
		return
	}
	keyToSources := make(map[string]map[string]struct{})
	for _, h := range listing.Hooks {
		k := dedupKey(h)
		if k == "" {
			continue
		}
		set, ok := keyToSources[k]
		if !ok {
			set = make(map[string]struct{})
			keyToSources[k] = set
		}
		set[h.SourceID] = struct{}{}
	}
	for i := range listing.Hooks {
		k := dedupKey(listing.Hooks[i])
		if k == "" {
			continue
		}
		if n := len(keyToSources[k]); n > 0 {
			listing.Hooks[i].DupCount = n
		}
	}
}
