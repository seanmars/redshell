// Package claude reads Claude Code session jsonl files under
// ~/.claude/projects/<encoded-cwd>/<sessionID>.jsonl. Sessions are listed
// grouped by encoded-cwd folder; the displayed cwd is resolved from inside
// the jsonl, never decoded from the folder name (the encoding is lossy).
package claude

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Event mirrors the shared sessionhistory.Event shape but uses string for
// Kind so this package does not need to import the parent.
type Event struct {
	Index    int                    `json:"index"`
	Kind     string                 `json:"kind"`
	Subtype  string                 `json:"subtype,omitempty"`
	Summary  string                 `json:"summary"`
	Raw      map[string]interface{} `json:"raw"`
	Children []Event                `json:"children,omitempty"`
}

// EventPage mirrors sessionhistory.EventPage minus the agent/session/offset
// fields the façade fills in.
type EventPage struct {
	Total        int     `json:"total"`
	HasMore      bool    `json:"hasMore"`
	SkippedLines int     `json:"skippedLines"`
	Events       []Event `json:"events"`
}

// SessionMeta holds Claude-specific session metadata (subset of
// sessionhistory.SessionMeta filled by this reader).
type SessionMeta struct {
	SessionID   string `json:"sessionID"`
	DisplayName string `json:"displayName,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
	ModifiedAt  string `json:"modifiedAt,omitempty"`
	ByteSize    int64  `json:"byteSize,omitempty"`
}

// SessionGroup is one cwd-grouped listing entry.
type SessionGroup struct {
	EncodedDir string        `json:"encodedDir"`
	Cwd        string        `json:"cwd,omitempty"`
	Sessions   []SessionMeta `json:"sessions"`
}

// Reader walks ~/.claude/projects/<encoded-cwd>/*.jsonl.
type Reader struct {
	root string
}

// NewReader returns a Reader rooted at the absolute path supplied. The
// supplied root is typically ~/.claude/projects on production and a temp
// dir in tests.
func NewReader(root string) *Reader {
	return &Reader{root: root}
}

// ListSessions walks the root directory, groups jsonl files by their
// parent (encoded-cwd) folder, and resolves the display cwd for each
// group from the first event in any session that carries a non-empty cwd.
// Sessions whose jsonl file is empty (zero bytes) are skipped so the
// frontend never has to render an event-less session.
// Metadata is os.Stat-only; rich titles are deferred to SessionMeta.
func (r *Reader) ListSessions() ([]SessionGroup, error) {
	if r.root == "" {
		return nil, errors.New("claude reader: empty root")
	}
	entries, err := os.ReadDir(r.root)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionGroup{}, nil
		}
		return nil, err
	}

	groups := make([]SessionGroup, 0, len(entries))
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}
		encodedDir := dir.Name()
		dirPath := filepath.Join(r.root, encodedDir)
		jsonls, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		sessions := make([]SessionMeta, 0, len(jsonls))
		for _, f := range jsonls {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			if info.Size() == 0 {
				continue
			}
			sessionID := strings.TrimSuffix(name, ".jsonl")
			sessions = append(sessions, SessionMeta{
				// path.Join (not filepath.Join) so the id stays forward-slash separated on Windows; the frontend splits on "/" only.
				SessionID:  path.Join(encodedDir, sessionID),
				ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
				ByteSize:   info.Size(),
			})
		}
		if len(sessions) == 0 {
			continue
		}
		// Resolve a representative cwd by peeking the first session that
		// yields one. Cap the per-session peek to ~32 lines.
		cwd := ""
		for _, s := range sessions {
			path := filepath.Join(r.root, s.SessionID+".jsonl")
			if c := peekFirstCwd(path, 32); c != "" {
				cwd = c
				break
			}
		}
		// Sort sessions inside the group by mtime desc.
		sort.SliceStable(sessions, func(i, j int) bool {
			return sessions[i].ModifiedAt > sessions[j].ModifiedAt
		})
		groups = append(groups, SessionGroup{
			EncodedDir: encodedDir,
			Cwd:        cwd,
			Sessions:   sessions,
		})
	}

	// Sort groups by their newest session mtime desc.
	sort.SliceStable(groups, func(i, j int) bool {
		var li, lj string
		if len(groups[i].Sessions) > 0 {
			li = groups[i].Sessions[0].ModifiedAt
		}
		if len(groups[j].Sessions) > 0 {
			lj = groups[j].Sessions[0].ModifiedAt
		}
		return li > lj
	})
	return groups, nil
}

// SessionMeta resolves the rich display name for one session by walking
// the jsonl just far enough to find a custom-title / agent-name / first
// user message / slug. SessionID is the relative path returned in
// ListSessions (e.g. "<encoded-cwd>/<uuid>").
func (r *Reader) SessionMeta(sessionID string) (SessionMeta, error) {
	path := filepath.Join(r.root, sessionID+".jsonl")
	info, err := os.Stat(path)
	if err != nil {
		return SessionMeta{}, fmt.Errorf("claude session %s: %w", sessionID, err)
	}
	display, cwd := resolveRichTitle(path)
	if display == "" {
		// Final fallback: short session id (last 8 chars of the file name).
		short := filepath.Base(sessionID)
		if len(short) > 8 {
			short = short[:8]
		}
		display = short
	}
	return SessionMeta{
		SessionID:   sessionID,
		DisplayName: display,
		Cwd:         cwd,
		ModifiedAt:  info.ModTime().UTC().Format(time.RFC3339),
		ByteSize:    info.Size(),
	}, nil
}

// ReadEvents returns one page of normalized events from sessionID's jsonl.
func (r *Reader) ReadEvents(sessionID string, offset, limit int) (EventPage, error) {
	path := filepath.Join(r.root, sessionID+".jsonl")
	return parseEventPage(path, offset, limit)
}
