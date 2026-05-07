package hooks

// SourceKind enumerates where a hook configuration came from on disk.
type SourceKind string

const (
	SourceUser   SourceKind = "user"
	SourceLocal  SourceKind = "local"
	SourcePlugin SourceKind = "plugin"
)

// Source describes a single hook-configuration file the viewer loaded.
// ID is the stable identifier referenced by Hook.SourceID and SourceError.SourceID
// — the frontend looks up the owning Source via this field rather than
// inferring from position. Path is the absolute filesystem path the user
// can open in their OS file manager. Label is human-facing ("User",
// "Local", "Plugin: foo@bar"). PluginKey/Scope are populated only when
// Kind == SourcePlugin.
type Source struct {
	ID        string     `json:"id"`
	Kind      SourceKind `json:"kind"`
	Path      string     `json:"path"`
	Label     string     `json:"label"`
	PluginKey string     `json:"pluginKey,omitempty"`
	Scope     string     `json:"scope,omitempty"`
}

// Hook is one flattened hook entry, normalized across Claude's nested
// matcher-group shape and Copilot's flat per-event shape.
//
// SourceID indexes back into Listing.Sources. Event is the lifecycle hook
// name verbatim ("PreToolUse", "sessionStart", ...). Matcher is empty for
// Copilot. DupCount is the number of distinct sources (for the same agent)
// in which the dedup key resolves to the same canonical value; 1 means no
// duplicate.
type Hook struct {
	ID       string                 `json:"id"`
	SourceID string                 `json:"sourceID"`
	Event    string                 `json:"event"`
	Matcher  string                 `json:"matcher,omitempty"`
	Type     string                 `json:"type"`
	Summary  string                 `json:"summary"`
	DupCount int                    `json:"dupCount"`
	Raw      map[string]interface{} `json:"raw"`
}

// SourceError attaches a per-source non-fatal error (parse failure, etc.)
// to a Source the user can still see in the tree.
type SourceError struct {
	SourceID string `json:"sourceID"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

// DisableFlag records that a loaded source had top-level
// "disableAllHooks": true. The frontend renders one banner per flag.
type DisableFlag struct {
	SourceID string `json:"sourceID"`
	Path     string `json:"path"`
}

// EmptyReason is a closed enum the frontend uses to render a tab-level
// empty state instead of a tree.
type EmptyReason string

const (
	EmptyReasonNone                 EmptyReason = ""
	EmptyReasonCopilotProjectScoped EmptyReason = "copilot-project-scoped"
)

// Listing is the per-agent payload returned by ListHooks.
type Listing struct {
	AgentID     string        `json:"agentID"`
	Sources     []Source      `json:"sources"`
	Hooks       []Hook        `json:"hooks"`
	Errors      []SourceError `json:"errors"`
	DisableAll  []DisableFlag `json:"disableAll"`
	EmptyReason EmptyReason   `json:"emptyReason"`
}

// ListOpts reserves the workspace argument for the future per-cwd Copilot
// scope (B-route in the design). v1 ignores any non-empty Workspace value.
type ListOpts struct {
	Workspace string `json:"workspace"`
}
