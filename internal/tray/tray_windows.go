//go:build windows

package tray

import (
	"context"
	_ "embed"
	"errors"
	"sync"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"redshell/internal/preferences"
)

//go:embed assets/tray.ico
var trayIcon []byte

type windowsManager struct {
	mu      sync.Mutex
	ctx     context.Context
	prefs   *preferences.Service
	started bool

	mShow            *systray.MenuItem
	mMinimize        *systray.MenuItem
	mCheckForUpdates *systray.MenuItem
	mQuit            *systray.MenuItem
	requestExit      func()
	checkForUpdates  func()
}

func newManager() Manager {
	return &windowsManager{}
}

func (m *windowsManager) Start(ctx context.Context, prefs *preferences.Service) error {
	if ctx == nil {
		return errors.New("tray: nil context")
	}
	if prefs == nil {
		return errors.New("tray: nil preferences service")
	}

	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.started = true
	m.ctx = ctx
	m.prefs = prefs
	m.mu.Unlock()

	go systray.Run(m.onReady, m.onExit)
	return nil
}

func (m *windowsManager) Stop() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.started = false
	m.mu.Unlock()
	systray.Quit()
}

func (m *windowsManager) Available() bool { return true }

func (m *windowsManager) RequestExit() {
	m.mu.Lock()
	cb := m.requestExit
	ctx := m.ctx
	m.mu.Unlock()
	if cb != nil {
		cb()
		return
	}
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

func (m *windowsManager) ShowWindow() {
	m.mu.Lock()
	ctx := m.ctx
	m.mu.Unlock()
	if ctx == nil {
		return
	}
	runtime.WindowShow(ctx)
}

func (m *windowsManager) HideWindow() {
	m.mu.Lock()
	ctx := m.ctx
	m.mu.Unlock()
	if ctx == nil {
		return
	}
	runtime.WindowHide(ctx)
}

// SetRequestExit lets the app layer hand the tray a function that performs an
// explicit-quit (sets the bypass flag, then calls runtime.Quit). Without it
// the tray falls back to runtime.Quit directly, which works but skips the
// flag and would cause OnBeforeClose to consult the close-behavior preference.
func (m *windowsManager) SetRequestExit(fn func()) {
	m.mu.Lock()
	m.requestExit = fn
	m.mu.Unlock()
}

// SetCheckForUpdates registers the click handler for the optional "Check
// for Updates" tray menu item. Must be called before Start so the menu is
// built with the item present. Passing nil leaves the item out entirely.
func (m *windowsManager) SetCheckForUpdates(fn func()) {
	m.mu.Lock()
	m.checkForUpdates = fn
	m.mu.Unlock()
}

func (m *windowsManager) onReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("RedShell")
	systray.SetTooltip("RedShell")

	m.mShow = systray.AddMenuItem("Show RedShell", "Bring the RedShell window to the foreground")
	m.mMinimize = systray.AddMenuItemCheckbox("Close button minimizes to tray", "Toggle whether the close button hides the window or exits the app", false)
	m.mu.Lock()
	hasUpdater := m.checkForUpdates != nil
	m.mu.Unlock()
	if hasUpdater {
		m.mCheckForUpdates = systray.AddMenuItem("Check for Updates", "Check for a newer release of RedShell")
	}
	systray.AddSeparator()
	m.mQuit = systray.AddMenuItem("Quit RedShell", "Exit RedShell")

	m.syncMinimizeFromPrefs()
	m.prefs.OnChange(func(p preferences.Preferences) {
		m.applyMinimizeChecked(p.CloseBehavior == preferences.CloseBehaviorMinimizeToTray)
	})

	go m.handleMenuClicks()
}

func (m *windowsManager) onExit() {
	// systray.Run returns once Quit() is called; nothing to do here.
}

func (m *windowsManager) handleMenuClicks() {
	// systray.MenuItem zero-value has a nil ClickedCh; reading from a nil
	// channel blocks forever, so it's safe to include in the select even
	// when the optional item wasn't created.
	var updatesCh <-chan struct{}
	if m.mCheckForUpdates != nil {
		updatesCh = m.mCheckForUpdates.ClickedCh
	}
	for {
		select {
		case <-m.mShow.ClickedCh:
			m.ShowWindow()
		case <-m.mMinimize.ClickedCh:
			m.toggleCloseBehavior()
		case <-updatesCh:
			m.invokeCheckForUpdates()
		case <-m.mQuit.ClickedCh:
			m.RequestExit()
			return
		}
	}
}

func (m *windowsManager) invokeCheckForUpdates() {
	m.mu.Lock()
	cb := m.checkForUpdates
	m.mu.Unlock()
	m.ShowWindow()
	if cb != nil {
		cb()
	}
}

func (m *windowsManager) toggleCloseBehavior() {
	current, err := m.prefs.GetCloseBehavior()
	if err != nil {
		return
	}
	next := preferences.CloseBehaviorMinimizeToTray
	if current == preferences.CloseBehaviorMinimizeToTray {
		next = preferences.CloseBehaviorExit
	}
	if err := m.prefs.SetCloseBehavior(next); err != nil {
		return
	}
	m.applyMinimizeChecked(next == preferences.CloseBehaviorMinimizeToTray)
}

func (m *windowsManager) syncMinimizeFromPrefs() {
	value, err := m.prefs.GetCloseBehavior()
	if err != nil {
		return
	}
	m.applyMinimizeChecked(value == preferences.CloseBehaviorMinimizeToTray)
}

func (m *windowsManager) applyMinimizeChecked(checked bool) {
	if m.mMinimize == nil {
		return
	}
	if checked {
		m.mMinimize.Check()
	} else {
		m.mMinimize.Uncheck()
	}
}
