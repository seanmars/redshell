package app

import (
	"redshell/internal/sessionhistory"
)

// SessionHistoryApp is the Wails-bound wrapper around the sessionhistory
// service. It exposes only read methods; the viewer never writes to or
// deletes any session file.
type SessionHistoryApp struct {
	svc *sessionhistory.Service
}

// NewSessionHistoryApp returns a new wrapper. The svc is constructed via
// sessionhistory.NewService at startup.
func NewSessionHistoryApp(svc *sessionhistory.Service) *SessionHistoryApp {
	return &SessionHistoryApp{svc: svc}
}

// ListSessions returns the per-agent listing (Groups for claude, Flat for
// copilot). The frontend dispatches on Listing.Kind.
func (a *SessionHistoryApp) ListSessions(agentID string) (sessionhistory.Listing, error) {
	return a.svc.ListSessions(agentID)
}

// SessionMeta resolves the rich display name and metadata for one session.
func (a *SessionHistoryApp) SessionMeta(agentID, sessionID string) (sessionhistory.SessionMeta, error) {
	return a.svc.SessionMeta(agentID, sessionID)
}

// ListEvents returns one paginated chunk of normalized events.
// offset 0 / limit <=0 falls back to defaults inside the service.
func (a *SessionHistoryApp) ListEvents(agentID, sessionID string, offset, limit int) (sessionhistory.EventPage, error) {
	return a.svc.ListEvents(agentID, sessionID, offset, limit)
}

// ResumeSession opens a new pwsh window and runs `<agent-cli> --resume <id>`
// for the given session, with its working directory set to `cwd` (the
// session's project directory). The sessionID may be the path-prefixed
// Claude shape or a bare Copilot UUID; the service extracts the basename and
// validates it strictly before invocation. The cwd is sanitized — if absent,
// not absolute, or non-existent, the spawned terminal falls back to the
// default working directory.
func (a *SessionHistoryApp) ResumeSession(agentID, sessionID, cwd string) error {
	return a.svc.ResumeSession(agentID, sessionID, cwd)
}
