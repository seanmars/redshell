//go:build windows

package sysproc

import "syscall"

// Hidden returns a SysProcAttr that prevents Windows from spawning a console
// window for the child process. Required because Wails production builds are
// linked as -H=windowsgui (no console), so each child process would otherwise
// trigger Windows to allocate a fresh console window.
//
// CREATE_NO_WINDOW = 0x08000000.
func Hidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
