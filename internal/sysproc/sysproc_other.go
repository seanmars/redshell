//go:build !windows

package sysproc

import "syscall"

// Hidden is a no-op on non-Windows platforms; returning nil keeps the call
// site uniform without forcing the caller to branch on GOOS.
func Hidden() *syscall.SysProcAttr {
	return nil
}
