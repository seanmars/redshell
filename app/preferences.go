package app

import (
	"context"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"redshell/internal/preferences"
	"redshell/internal/tray"
)

// CloseBehaviorPromptEvent is the runtime event the backend emits when the
// user attempts to close the main window for the first time and no
// close-behavior preference has been recorded yet. The frontend listens for
// this event and opens the close-behavior prompt modal.
const CloseBehaviorPromptEvent = "tray:close-behavior-prompt"

// AppPreferencesApp wires the preferences service and tray manager into the
// Wails binding layer. It owns the "explicit quit" flag the OnBeforeClose
// hook uses to bypass the close-behavior preference when the exit was
// triggered programmatically (e.g. by the tray's Quit menu item or by the
// user's "Exit RedShell" choice in the prompt modal).
type AppPreferencesApp struct {
	ctx          context.Context
	prefs        *preferences.Service
	tray         tray.Manager
	explicitQuit atomic.Bool
}

func NewAppPreferencesApp(prefs *preferences.Service, trayMgr tray.Manager) *AppPreferencesApp {
	return &AppPreferencesApp{prefs: prefs, tray: trayMgr}
}

func (a *AppPreferencesApp) Startup(ctx context.Context) {
	a.ctx = ctx
	if a.tray != nil {
		a.tray.SetRequestExit(a.requestExit)
	}
}

func (a *AppPreferencesApp) GetCloseBehavior() (string, error) {
	return a.prefs.GetCloseBehavior()
}

func (a *AppPreferencesApp) SetCloseBehavior(value string) error {
	return a.prefs.SetCloseBehavior(value)
}

// GetAutoUpdate exposes the persisted auto-update preferences block to the
// frontend Settings -> Updates tab.
func (a *AppPreferencesApp) GetAutoUpdate() (preferences.AutoUpdate, error) {
	return a.prefs.GetAutoUpdate()
}

func (a *AppPreferencesApp) SetAutoUpdateEnabled(value bool) error {
	return a.prefs.SetAutoUpdateEnabled(value)
}

func (a *AppPreferencesApp) SetAutoUpdateInterval(hours int) error {
	return a.prefs.SetAutoUpdateInterval(hours)
}

func (a *AppPreferencesApp) SetAutoUpdateSource(source string) error {
	return a.prefs.SetAutoUpdateSource(source)
}

func (a *AppPreferencesApp) SetAutoUpdateGithubRepo(repo string) error {
	return a.prefs.SetAutoUpdateGithubRepo(repo)
}

func (a *AppPreferencesApp) SetAutoUpdateGitlabHost(host string) error {
	return a.prefs.SetAutoUpdateGitlabHost(host)
}

func (a *AppPreferencesApp) SetAutoUpdateGitlabProject(project string) error {
	return a.prefs.SetAutoUpdateGitlabProject(project)
}

func (a *AppPreferencesApp) SetAutoUpdateSkipVersion(version string) error {
	return a.prefs.SetAutoUpdateSkipVersion(version)
}

// RequestExit is bound to the frontend so the close-behavior prompt modal
// can ask the backend to terminate after the user picks "Exit RedShell".
func (a *AppPreferencesApp) RequestExit() {
	a.requestExit()
}

// HideToTray is bound to the frontend so the close-behavior prompt modal
// can hide the main window immediately after the user picks
// "Minimize to tray". Without this the window would remain visible after
// the choice was recorded.
func (a *AppPreferencesApp) HideToTray() {
	if a.ctx == nil {
		return
	}
	runtime.WindowHide(a.ctx)
}

// HandleBeforeClose implements the Wails OnBeforeClose hook policy. Returning
// true cancels the close; returning false allows it.
//
// - Explicit quit (set by RequestExit) → return false, let the process exit.
// - closeBehavior == "exit"            → return false, let the process exit.
// - closeBehavior == "minimize-to-tray"→ hide the window and return true.
// - closeBehavior == "unset"           → emit the prompt event and return true.
func (a *AppPreferencesApp) HandleBeforeClose(ctx context.Context) bool {
	if a.explicitQuit.Load() {
		return false
	}
	value, err := a.prefs.GetCloseBehavior()
	if err != nil || value == preferences.CloseBehaviorUnset {
		runtime.EventsEmit(ctx, CloseBehaviorPromptEvent)
		return true
	}
	switch value {
	case preferences.CloseBehaviorExit:
		return false
	case preferences.CloseBehaviorMinimizeToTray:
		runtime.WindowHide(ctx)
		return true
	default:
		runtime.EventsEmit(ctx, CloseBehaviorPromptEvent)
		return true
	}
}

func (a *AppPreferencesApp) requestExit() {
	a.explicitQuit.Store(true)
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
