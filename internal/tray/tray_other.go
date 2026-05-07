//go:build !windows

package tray

import (
	"context"

	"redshell/internal/preferences"
)

type noopManager struct{}

func newManager() Manager { return &noopManager{} }

func (n *noopManager) Start(ctx context.Context, prefs *preferences.Service) error {
	_ = ctx
	_ = prefs
	return nil
}

func (n *noopManager) Stop()                     {}
func (n *noopManager) Available() bool           { return false }
func (n *noopManager) RequestExit()              {}
func (n *noopManager) ShowWindow()               {}
func (n *noopManager) HideWindow()               {}
func (n *noopManager) SetRequestExit(func())     {}
func (n *noopManager) SetCheckForUpdates(func()) {}
