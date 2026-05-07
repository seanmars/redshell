package tray

import (
	"context"

	"redshell/internal/preferences"
)

// Manager owns the system tray icon lifecycle and exposes the actions the
// rest of the app needs to drive it (show / hide / request exit).
//
// The Windows implementation is provided by tray_windows.go; non-Windows
// builds use the no-op stub in tray_other.go so the rest of the codebase can
// depend on the same interface unconditionally.
type Manager interface {
	Start(ctx context.Context, prefs *preferences.Service) error
	Stop()
	Available() bool
	RequestExit()
	ShowWindow()
	HideWindow()
	SetRequestExit(fn func())
	// SetCheckForUpdates registers the click handler for the "Check for
	// Updates" tray menu item. When fn is nil at the moment the menu is
	// built, the item is omitted entirely (e.g. on installer-installed
	// builds where auto-update is disabled).
	SetCheckForUpdates(fn func())
}

// NewManager returns the platform-appropriate Manager implementation. The
// concrete type is decided at build time via build tags on tray_windows.go
// and tray_other.go.
func NewManager() Manager { return newManager() }
