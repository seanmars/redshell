//go:build !windows

package sessionhistory

func defaultLaunchResumeTerminal(_, _, _ string) error {
	return ErrTerminalUnsupported
}
