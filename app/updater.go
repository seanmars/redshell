package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"redshell/internal/updater"
)

// UpdaterApp wires the updater service into the Wails binding layer. The
// service itself is platform/runtime-agnostic; this wrapper supplies the
// Wails-specific event emitter and quit hook.
type UpdaterApp struct {
	svc *updater.Service
	ctx context.Context
}

func NewUpdaterApp(svc *updater.Service) *UpdaterApp {
	return &UpdaterApp{svc: svc}
}

// Startup is called by the Wails OnStartup hook after preferences and tray
// are ready. It captures the Wails context and starts the updater's run loop.
func (a *UpdaterApp) Startup(ctx context.Context) {
	a.ctx = ctx
	emit := func(name string, data any) {
		runtime.EventsEmit(ctx, name, data)
	}
	quit := func() {
		runtime.Quit(ctx)
	}
	if err := a.svc.Start(ctx, emit, quit); err != nil {
		runtime.LogErrorf(ctx, "updater: start failed: %v", err)
	}
}

// CheckNow triggers an immediate check against the active source, ignoring
// the elapsed-interval debounce.
func (a *UpdaterApp) CheckNow() {
	a.svc.CheckNow()
}

// PeekBothSources queries both providers in parallel without affecting
// active-source state. Used by the Settings -> Updates tab to render
// side-by-side latest-version status.
func (a *UpdaterApp) PeekBothSources() updater.PeekResult {
	if a.ctx == nil {
		return updater.PeekResult{}
	}
	return a.svc.PeekBothSources(a.ctx)
}

// InstallAvailable downloads, verifies, and replaces the running binary
// with the most recently available release. Errors propagate to the caller
// and an `updater:error` event also fires.
func (a *UpdaterApp) InstallAvailable() error {
	if a.ctx == nil {
		return nil
	}
	return a.svc.InstallAvailable(a.ctx)
}

// SkipVersion persists `version` so subsequent checks resolving the same
// version do not re-emit the `updater:available` event.
func (a *UpdaterApp) SkipVersion(version string) error {
	return a.svc.SkipVersion(version)
}

// Unskip clears the persisted skip-version.
func (a *UpdaterApp) Unskip() error {
	return a.svc.Unskip()
}

// GetState returns the current updater snapshot for the Settings UI.
func (a *UpdaterApp) GetState() updater.State {
	return a.svc.GetState()
}

// InProgress is consumed by main.go's OnBeforeClose hook to short-circuit
// the close-behavior prompt during a rename swap.
func (a *UpdaterApp) InProgress() bool {
	return a.svc.InProgress()
}

// TrayOpenUpdatesEvent is emitted when the user clicks "Check for Updates"
// in the tray menu. The frontend listens for this event to switch the
// Settings view to the Updates tab.
const TrayOpenUpdatesEvent = "tray:open-updates"

// HandleTrayOpen is invoked by the tray's "Check for Updates" menu item.
// It emits the navigation event for the frontend and fires a manual check.
func (a *UpdaterApp) HandleTrayOpen() {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, TrayOpenUpdatesEvent)
	a.svc.CheckNow()
}

// ManualRequired exposes the snapshot bit so main.go can decide whether to
// register the tray "Check for Updates" item.
func (a *UpdaterApp) ManualRequired() bool {
	return a.svc.GetState().ManualRequired
}
