package sessionhistory

// EventKind is the normalized event role used by the frontend.
type EventKind string

const (
	KindUser       EventKind = "user"
	KindAssistant  EventKind = "assistant"
	KindToolUse    EventKind = "tool_use"
	KindToolResult EventKind = "tool_result"
	KindSystem     EventKind = "system"
	KindAttachment EventKind = "attachment"
	KindMeta       EventKind = "meta"
)

// SessionMeta carries the cheap-listing metadata for a single session.
// DisplayName is populated by the rich-title resolver; while only listing,
// readers may leave it empty and let the façade resolve it on open.
type SessionMeta struct {
	AgentID     string `json:"agentID"`
	SessionID   string `json:"sessionID"`
	DisplayName string `json:"displayName,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Summary     string `json:"summary,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	ModifiedAt  string `json:"modifiedAt,omitempty"`
	ByteSize    int64  `json:"byteSize,omitempty"`
	HasEvents   bool   `json:"hasEvents"`
}

// SessionGroup is the Claude-style cwd-grouped listing entry.
type SessionGroup struct {
	EncodedDir string        `json:"encodedDir"`
	Cwd        string        `json:"cwd,omitempty"`
	Sessions   []SessionMeta `json:"sessions"`
}

// Listing is a discriminated union: Kind == "groups" populates Groups,
// Kind == "flat" populates Flat. Frontend dispatches on Kind.
type Listing struct {
	AgentID string         `json:"agentID"`
	Kind    string         `json:"kind"`
	Groups  []SessionGroup `json:"groups,omitempty"`
	Flat    []SessionMeta  `json:"flat,omitempty"`
}

// Event is one parsed JSONL line, normalized for the frontend.
// Raw is the full event JSON as a generic map with encrypted fields replaced
// by a `{ "_redacted": "<field>", "_size": <bytes> }` sentinel.
type Event struct {
	Index    int                    `json:"index"`
	Kind     EventKind              `json:"kind"`
	Subtype  string                 `json:"subtype,omitempty"`
	Summary  string                 `json:"summary"`
	Raw      map[string]interface{} `json:"raw"`
	Children []Event                `json:"children,omitempty"`
}

// EventPage is a paginated slice of events for the frontend's virtual list.
// Total is the count of events (including skipped lines) the parser saw up
// to the end of the file. SkippedLines counts json.Unmarshal failures
// encountered anywhere in the file (not only the returned page).
type EventPage struct {
	AgentID      string  `json:"agentID"`
	SessionID    string  `json:"sessionID"`
	Offset       int     `json:"offset"`
	Limit        int     `json:"limit"`
	Total        int     `json:"total"`
	HasMore      bool    `json:"hasMore"`
	SkippedLines int     `json:"skippedLines"`
	Events       []Event `json:"events"`
}

// Reader is the per-agent contract dispatched by Service.
type Reader interface {
	ListSessions() (Listing, error)
	SessionMeta(sessionID string) (SessionMeta, error)
	ReadEvents(sessionID string, offset, limit int) (EventPage, error)
}
