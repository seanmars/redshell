//go:build !windows

package updater

// Swap is not supported on non-Windows platforms in v1. The service layer
// detects this via ErrPlatformUnsupported and falls back to a manual-update
// message.
func Swap(currentPath, newPath string) error {
	return ErrPlatformUnsupported
}
