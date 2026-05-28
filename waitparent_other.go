//go:build !windows

package main

import "time"

// waitForParentExit is a no-op on non-Windows platforms; the single-instance
// lock and its updater relaunch handshake are only wired on Windows.
func waitForParentExit(int, time.Duration) {}
