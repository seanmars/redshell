//go:build windows

package sessionhistory

import (
	"fmt"
	"os/exec"
	"syscall"
)

// createNoWindow suppresses the transient cmd.exe console window so the user
// only sees the pwsh window that `start` opens. (Win32 process creation flag.)
const createNoWindow = 0x08000000

// defaultLaunchResumeTerminal opens a new pwsh window that runs
// `<cli> --resume <sessionID>` in cwd and SHALL stay open after the agent CLI
// exits, until the user closes it explicitly.
//
// The launch path is `cmd.exe /c start "" pwsh -NoExit -NoProfile -Command "<inner>"`:
//   - `cmd /c start` opens a fully-detached new console window for pwsh, so
//     the resumed terminal is independent of RedShell's process tree.
//   - The empty "" after `start` is the (no-op) window title — without it
//     `start` would parse the next quoted argument as the window title.
//   - `-NoExit` keeps pwsh interactive after the agent CLI returns; the
//     window persists until the user types `exit` or closes it manually.
//   - `-NoProfile` skips the user's pwsh profile so a hostile profile (e.g.
//     `$ErrorActionPreference = 'Stop'`) cannot subvert -NoExit.
//
// cwd is set as the cmd.exe process's working directory via cmd.Dir, which
// is inherited by the spawned pwsh; the cwd is never interpolated into the
// shell command line.
func defaultLaunchResumeTerminal(cli, sessionID, cwd string) error {
	inner := fmt.Sprintf("%s --resume %s", cli, sessionID)
	cmd := exec.Command(
		"cmd", "/c", "start", "",
		"pwsh", "-NoExit", "-NoProfile", "-Command", inner,
	)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd.Start()
}
