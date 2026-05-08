//go:build !windows

package updater

// SpawnInstaller is a no-op stub on non-Windows platforms; the installer
// pathway is Windows-only in v1. Returning ErrPlatformUnsupported lets the
// service emit a structured error rather than panicking.
func SpawnInstaller(installerPath string, args []string) error {
	_ = installerPath
	_ = args
	return ErrPlatformUnsupported
}
