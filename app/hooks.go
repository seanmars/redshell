package app

import (
	"redshell/internal/hooks"
)

// HooksApp is the Wails-bound wrapper around the hooks service. It exposes
// only read methods; the viewer never writes to or modifies any settings
// file.
type HooksApp struct {
	svc *hooks.Service
}

// NewHooksApp returns a new wrapper. svc is constructed via
// hooks.NewService at startup.
func NewHooksApp(svc *hooks.Service) *HooksApp {
	return &HooksApp{svc: svc}
}

// ListHooks returns the per-agent Listing. opts.Workspace is reserved for
// the future per-workspace Copilot scope; v1 ignores any non-empty value.
func (a *HooksApp) ListHooks(agentID string, opts hooks.ListOpts) (hooks.Listing, error) {
	if a.svc == nil {
		return hooks.Listing{AgentID: agentID}, nil
	}
	return a.svc.ListHooks(agentID, opts)
}
