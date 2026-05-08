//go:build windows

package updater

import (
	"errors"
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// errorCancelled is the Win32 error code returned by ShellExecute when the
// user clicks No on the UAC elevation prompt. We map it to ErrUACDeclined so
// callers can distinguish "user said no" from "spawn failed for some other
// reason" without parsing string messages.
const errorCancelled syscall.Errno = 1223

// SpawnInstaller launches installerPath under the Windows shell with the
// `runas` verb, which triggers the UAC elevation prompt. It blocks until the
// user accepts or declines the prompt; on accept the elevated process is
// created (the OS guarantees this before the call returns) and SpawnInstaller
// returns nil. On decline, returns ErrUACDeclined. On any other failure it
// returns the wrapped Win32 error.
//
// The installer process runs detached — we do not wait for it. The caller is
// expected to call quitApp() shortly after this returns success so the
// running RedShell releases its file lock and the elevated installer can
// overwrite the binary (NSIS install Section sleeps briefly to absorb the
// race; see project.nsi).
func SpawnInstaller(installerPath string, args []string) error {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return fmt.Errorf("installer spawn: encode verb: %w", err)
	}
	file, err := windows.UTF16PtrFromString(installerPath)
	if err != nil {
		return fmt.Errorf("installer spawn: encode path: %w", err)
	}
	var paramsPtr *uint16
	if len(args) > 0 {
		params, err := windows.UTF16PtrFromString(strings.Join(args, " "))
		if err != nil {
			return fmt.Errorf("installer spawn: encode args: %w", err)
		}
		paramsPtr = params
	}

	// SW_SHOW = 5; harmless for a silent installer (no window will appear
	// because of /S) and matches the default for ShellExecute.
	const swShow = 5
	if err := windows.ShellExecute(0, verb, file, paramsPtr, nil, swShow); err != nil {
		var errno syscall.Errno
		if errors.As(err, &errno) && errno == errorCancelled {
			return ErrUACDeclined
		}
		return fmt.Errorf("installer spawn: %w", err)
	}
	return nil
}
