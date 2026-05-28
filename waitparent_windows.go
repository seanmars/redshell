//go:build windows

package main

import (
	"time"

	"golang.org/x/sys/windows"
)

// waitForParentExit blocks until the process identified by pid exits or the
// timeout elapses, whichever comes first. It backs the updater relaunch
// handshake: the new binary waits for the outgoing process to release the
// single-instance lock before the Wails runtime tries to acquire it. Returns
// immediately if the process is already gone or cannot be opened.
func waitForParentExit(pid int, timeout time.Duration) {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)

	ms := timeout.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	_, _ = windows.WaitForSingleObject(h, uint32(ms))
}
