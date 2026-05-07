package sessionhistory

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"redshell/internal/sessionhistory/claude"
	"redshell/internal/sessionhistory/copilot"
)

// ErrUnknownAgent is returned when the frontend supplies an agentID that is
// not in the closed enum of supported agents.
var ErrUnknownAgent = errors.New("unknown agent")

// ErrInvalidSessionID is returned when a session id resolves to a path
// outside the configured agent root, or when it contains traversal segments.
var ErrInvalidSessionID = errors.New("invalid session id")

// Service is the façade bound to the Wails layer. It owns one Reader per
// agent and dispatches by agentID.
type Service struct {
	readers map[string]Reader
	roots   map[string]string
}

// NewService wires production session roots derived from os.UserHomeDir.
func NewService() (*Service, error) {
	roots, err := resolveProductionRoots()
	if err != nil {
		return nil, err
	}
	return NewServiceWithRoots(roots), nil
}

// NewServiceWithRoots builds a Service with the supplied per-agent roots.
// Tests use this to point at temporary directories.
func NewServiceWithRoots(roots map[string]string) *Service {
	readers := make(map[string]Reader, len(roots))
	for agentID, root := range roots {
		switch agentID {
		case "claude":
			readers[agentID] = claudeAdapter{r: claude.NewReader(root)}
		case "copilot":
			readers[agentID] = copilotAdapter{r: copilot.NewReader(root)}
		}
	}
	return &Service{readers: readers, roots: roots}
}

// ListSessions dispatches to the per-agent Reader.
func (s *Service) ListSessions(agentID string) (Listing, error) {
	r, err := s.readerFor(agentID)
	if err != nil {
		return Listing{}, err
	}
	listing, err := r.ListSessions()
	if err != nil {
		return Listing{}, err
	}
	listing.AgentID = agentID
	return listing, nil
}

// SessionMeta resolves the rich display name for a single session.
func (s *Service) SessionMeta(agentID, sessionID string) (SessionMeta, error) {
	r, err := s.readerFor(agentID)
	if err != nil {
		return SessionMeta{}, err
	}
	if err := s.validateSessionID(agentID, sessionID); err != nil {
		return SessionMeta{}, err
	}
	meta, err := r.SessionMeta(sessionID)
	if err != nil {
		return SessionMeta{}, err
	}
	meta.AgentID = agentID
	return meta, nil
}

// ListEvents returns one page of normalized events.
func (s *Service) ListEvents(agentID, sessionID string, offset, limit int) (EventPage, error) {
	r, err := s.readerFor(agentID)
	if err != nil {
		return EventPage{}, err
	}
	if err := s.validateSessionID(agentID, sessionID); err != nil {
		return EventPage{}, err
	}
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	page, err := r.ReadEvents(sessionID, offset, limit)
	if err != nil {
		return EventPage{}, err
	}
	page.AgentID = agentID
	page.SessionID = sessionID
	page.Offset = offset
	page.Limit = limit
	return page, nil
}

func (s *Service) readerFor(agentID string) (Reader, error) {
	r, ok := s.readers[agentID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownAgent, agentID)
	}
	return r, nil
}

// validateSessionID rejects session ids that resolve outside the agent root
// once joined with the root. Paths must not contain traversal that escapes
// the root after Clean, and must not be absolute.
func (s *Service) validateSessionID(agentID, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("%w: empty", ErrInvalidSessionID)
	}
	if strings.ContainsAny(sessionID, "\x00") {
		return fmt.Errorf("%w: contains null byte", ErrInvalidSessionID)
	}
	if filepath.IsAbs(sessionID) {
		return fmt.Errorf("%w: absolute path", ErrInvalidSessionID)
	}
	root, ok := s.roots[agentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownAgent, agentID)
	}
	joined := filepath.Join(root, sessionID)
	rel, err := filepath.Rel(root, joined)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSessionID, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%w: escapes root", ErrInvalidSessionID)
	}
	return nil
}

// claudeAdapter and copilotAdapter convert the per-package reader output
// to the shared DTOs in this package. The per-package readers intentionally
// do not import the parent package, so we translate field-by-field.
type claudeAdapter struct{ r *claude.Reader }

func (a claudeAdapter) ListSessions() (Listing, error) {
	groups, err := a.r.ListSessions()
	if err != nil {
		return Listing{}, err
	}
	out := Listing{Kind: "groups", Groups: make([]SessionGroup, 0, len(groups))}
	for _, g := range groups {
		sessions := make([]SessionMeta, 0, len(g.Sessions))
		for _, s := range g.Sessions {
			sessions = append(sessions, SessionMeta{
				AgentID:    "claude",
				SessionID:  s.SessionID,
				ModifiedAt: s.ModifiedAt,
				ByteSize:   s.ByteSize,
				HasEvents:  true,
			})
		}
		out.Groups = append(out.Groups, SessionGroup{
			EncodedDir: g.EncodedDir,
			Cwd:        g.Cwd,
			Sessions:   sessions,
		})
	}
	return out, nil
}

func (a claudeAdapter) SessionMeta(sessionID string) (SessionMeta, error) {
	m, err := a.r.SessionMeta(sessionID)
	if err != nil {
		return SessionMeta{}, err
	}
	return SessionMeta{
		AgentID:     "claude",
		SessionID:   sessionID,
		DisplayName: m.DisplayName,
		Cwd:         m.Cwd,
		ModifiedAt:  m.ModifiedAt,
		ByteSize:    m.ByteSize,
		HasEvents:   true,
	}, nil
}

func (a claudeAdapter) ReadEvents(sessionID string, offset, limit int) (EventPage, error) {
	page, err := a.r.ReadEvents(sessionID, offset, limit)
	if err != nil {
		return EventPage{}, err
	}
	return EventPage{
		Total:        page.Total,
		HasMore:      page.HasMore,
		SkippedLines: page.SkippedLines,
		Events:       fromClaudeEvents(page.Events),
	}, nil
}

func fromClaudeEvents(in []claude.Event) []Event {
	out := make([]Event, len(in))
	for i, ev := range in {
		out[i] = Event{
			Index:    ev.Index,
			Kind:     EventKind(ev.Kind),
			Subtype:  ev.Subtype,
			Summary:  ev.Summary,
			Raw:      ev.Raw,
			Children: fromClaudeEvents(ev.Children),
		}
	}
	return out
}

type copilotAdapter struct{ r *copilot.Reader }

func (a copilotAdapter) ListSessions() (Listing, error) {
	groups, err := a.r.ListSessionGroups()
	if err != nil {
		return Listing{}, err
	}
	out := Listing{Kind: "groups", Groups: make([]SessionGroup, 0, len(groups))}
	for _, g := range groups {
		sessions := make([]SessionMeta, 0, len(g.Sessions))
		for _, s := range g.Sessions {
			sessions = append(sessions, SessionMeta{
				AgentID:    "copilot",
				SessionID:  s.SessionID,
				Cwd:        s.Cwd,
				Repository: s.Repository,
				Branch:     s.Branch,
				Summary:    s.Summary,
				CreatedAt:  s.CreatedAt,
				UpdatedAt:  s.UpdatedAt,
				HasEvents:  s.HasEvents,
			})
		}
		out.Groups = append(out.Groups, SessionGroup{
			EncodedDir: g.EncodedDir,
			Cwd:        g.Cwd,
			Sessions:   sessions,
		})
	}
	return out, nil
}

func (a copilotAdapter) SessionMeta(sessionID string) (SessionMeta, error) {
	m, err := a.r.SessionMeta(sessionID)
	if err != nil {
		return SessionMeta{}, err
	}
	return SessionMeta{
		AgentID:     "copilot",
		SessionID:   sessionID,
		DisplayName: m.DisplayName,
		Cwd:         m.Cwd,
		Repository:  m.Repository,
		Branch:      m.Branch,
		Summary:     m.Summary,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		HasEvents:   m.HasEvents,
	}, nil
}

func (a copilotAdapter) ReadEvents(sessionID string, offset, limit int) (EventPage, error) {
	page, err := a.r.ReadEvents(sessionID, offset, limit)
	if err != nil {
		return EventPage{}, err
	}
	return EventPage{
		Total:        page.Total,
		HasMore:      page.HasMore,
		SkippedLines: page.SkippedLines,
		Events:       fromCopilotEvents(page.Events),
	}, nil
}

func fromCopilotEvents(in []copilot.Event) []Event {
	out := make([]Event, len(in))
	for i, ev := range in {
		out[i] = Event{
			Index:    ev.Index,
			Kind:     EventKind(ev.Kind),
			Subtype:  ev.Subtype,
			Summary:  ev.Summary,
			Raw:      ev.Raw,
			Children: fromCopilotEvents(ev.Children),
		}
	}
	return out
}
