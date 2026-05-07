package copilot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// UnknownCwd is the sentinel group key used when a session's workspace.yaml
// has no resolvable cwd / git_root / repository.
const UnknownCwd = "(unknown)"

// Event mirrors the shared sessionhistory.Event shape.
type Event struct {
	Index    int                    `json:"index"`
	Kind     string                 `json:"kind"`
	Subtype  string                 `json:"subtype,omitempty"`
	Summary  string                 `json:"summary"`
	Raw      map[string]interface{} `json:"raw"`
	Children []Event                `json:"children,omitempty"`
}

// EventPage carries the per-page subset and the file-wide stats.
type EventPage struct {
	Total        int     `json:"total"`
	HasMore      bool    `json:"hasMore"`
	SkippedLines int     `json:"skippedLines"`
	Events       []Event `json:"events"`
}

// SessionGroup is the cwd-grouped listing entry produced by ListSessionGroups.
// Cwd carries the full resolved working directory; the frontend shortens it
// for display via its own helper. EncodedDir is set to the same resolved cwd
// string so the Vue list key stays stable per group.
type SessionGroup struct {
	EncodedDir string        `json:"encodedDir"`
	Cwd        string        `json:"cwd,omitempty"`
	Sessions   []SessionMeta `json:"sessions"`
}

// SessionMeta carries Copilot-specific session metadata. GitRoot is local to
// this package — the shared sessionhistory.SessionMeta does not surface it
// because the frontend has no use for it; we only need it as a fallback group
// key when cwd is empty.
type SessionMeta struct {
	SessionID   string `json:"sessionID"`
	DisplayName string `json:"displayName,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
	GitRoot     string `json:"gitRoot,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Summary     string `json:"summary,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	HasEvents   bool   `json:"hasEvents"`
}

// Reader walks ~/.copilot/session-state/*/.
type Reader struct {
	root string
}

// NewReader returns a Reader rooted at the absolute path supplied.
func NewReader(root string) *Reader {
	return &Reader{root: root}
}

// ListSessions enumerates direct child directories under the root, parses
// each session's workspace.yaml, and returns the flat listing sorted by
// created_at descending. Sessions with no workspace.yaml are skipped.
// Sessions whose events.jsonl is missing or empty are also skipped so the
// frontend never has to render an event-less session.
func (r *Reader) ListSessions() ([]SessionMeta, error) {
	if r.root == "" {
		return nil, errors.New("copilot reader: empty root")
	}
	entries, err := os.ReadDir(r.root)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionMeta{}, nil
		}
		return nil, err
	}

	out := make([]SessionMeta, 0, len(entries))
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}
		sessionID := dir.Name()
		sessionDir := filepath.Join(r.root, sessionID)
		ws, err := readWorkspace(sessionDir)
		if err != nil || ws.ID == "" && ws.CreatedAt == "" {
			// Skip sessions whose manifest fails to parse; we treat
			// fully-missing manifests as "not a session".
			continue
		}
		info, err := os.Stat(filepath.Join(sessionDir, "events.jsonl"))
		if err != nil || info.IsDir() || info.Size() == 0 {
			continue
		}
		out = append(out, SessionMeta{
			SessionID:  sessionID,
			Cwd:        ws.Cwd,
			GitRoot:    ws.GitRoot,
			Repository: ws.Repository,
			Branch:     ws.Branch,
			Summary:    ws.Summary,
			CreatedAt:  ws.CreatedAt,
			UpdatedAt:  ws.UpdatedAt,
			HasEvents:  true,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt > out[j].CreatedAt
	})
	return out, nil
}

// ListSessionGroups returns the Copilot session listing bucketed by resolved
// working directory. Sessions are first produced by ListSessions (so the
// per-session filtering, parsing, and created_at-desc ordering already applied
// is preserved inside each bucket), then grouped by resolveCwd, then sorted
// across groups by the most recent sessionRecency in each group.
func (r *Reader) ListSessionGroups() ([]SessionGroup, error) {
	flat, err := r.ListSessions()
	if err != nil {
		return nil, err
	}
	if len(flat) == 0 {
		return []SessionGroup{}, nil
	}

	buckets := make(map[string][]SessionMeta)
	order := make([]string, 0)
	for _, s := range flat {
		key := resolveCwd(s.Cwd, s.GitRoot, s.Repository)
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
		}
		buckets[key] = append(buckets[key], s)
	}

	groups := make([]SessionGroup, 0, len(order))
	for _, key := range order {
		groups = append(groups, SessionGroup{
			EncodedDir: key,
			Cwd:        key,
			Sessions:   buckets[key],
		})
	}

	sort.SliceStable(groups, func(i, j int) bool {
		return groupRecency(groups[i]).After(groupRecency(groups[j]))
	})
	return groups, nil
}

// resolveCwd picks the best available group key from a session's workspace
// fields, falling back to UnknownCwd when nothing is set. The order matches
// the design: cwd → git_root → repository → "(unknown)".
func resolveCwd(cwd, gitRoot, repository string) string {
	if v := strings.TrimSpace(cwd); v != "" {
		return v
	}
	if v := strings.TrimSpace(gitRoot); v != "" {
		return v
	}
	if v := strings.TrimSpace(repository); v != "" {
		return v
	}
	return UnknownCwd
}

// sessionRecency returns max(created_at, updated_at) parsed as RFC3339. Zero
// time is returned when neither timestamp parses; callers compare these via
// time.Time.After which gives stable ordering for the unparseable case.
func sessionRecency(meta SessionMeta) time.Time {
	created, _ := time.Parse(time.RFC3339, meta.CreatedAt)
	updated, _ := time.Parse(time.RFC3339, meta.UpdatedAt)
	if updated.After(created) {
		return updated
	}
	return created
}

// groupRecency returns the latest sessionRecency across a group's sessions.
func groupRecency(g SessionGroup) time.Time {
	var latest time.Time
	for _, s := range g.Sessions {
		if r := sessionRecency(s); r.After(latest) {
			latest = r
		}
	}
	return latest
}

// SessionMeta resolves the rich display name for a single Copilot session.
// Resolution priority: workspace.summary → first user.message.data.content
// → repository → cwd → first 8 chars of session id.
func (r *Reader) SessionMeta(sessionID string) (SessionMeta, error) {
	sessionDir := filepath.Join(r.root, sessionID)
	ws, err := readWorkspace(sessionDir)
	if err != nil {
		return SessionMeta{}, fmt.Errorf("copilot session %s: %w", sessionID, err)
	}
	hasEvents := false
	if info, err := os.Stat(filepath.Join(sessionDir, "events.jsonl")); err == nil && !info.IsDir() && info.Size() > 0 {
		hasEvents = true
	}

	display := ws.Summary
	if display == "" && hasEvents {
		display = peekFirstUserContent(filepath.Join(sessionDir, "events.jsonl"))
	}
	if display == "" {
		display = ws.Repository
	}
	if display == "" {
		display = ws.Cwd
	}
	if display == "" {
		short := sessionID
		if len(short) > 8 {
			short = short[:8]
		}
		display = short
	}
	return SessionMeta{
		SessionID:   sessionID,
		DisplayName: strings.TrimSpace(display),
		Cwd:         ws.Cwd,
		GitRoot:     ws.GitRoot,
		Repository:  ws.Repository,
		Branch:      ws.Branch,
		Summary:     ws.Summary,
		CreatedAt:   ws.CreatedAt,
		UpdatedAt:   ws.UpdatedAt,
		HasEvents:   hasEvents,
	}, nil
}

// ReadEvents returns one page of normalized events from events.jsonl.
// If events.jsonl is missing, returns an empty page rather than an error.
func (r *Reader) ReadEvents(sessionID string, offset, limit int) (EventPage, error) {
	path := filepath.Join(r.root, sessionID, "events.jsonl")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return EventPage{Events: []Event{}}, nil
		}
		return EventPage{}, err
	}
	return parseEventPage(path, offset, limit)
}
