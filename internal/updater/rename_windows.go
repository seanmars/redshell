//go:build windows

package updater

import (
	"fmt"
	"os"
)

// Swap moves currentPath aside to currentPath+".old" and moves newPath into
// currentPath. On Windows this works while currentPath is the running
// executable, because rename is permitted on a running .exe even though
// delete and overwrite are not.
//
// On a partial failure (rename of newPath fails) Swap attempts to roll back
// the first rename so the caller is left with the original binary in place.
func Swap(currentPath, newPath string) error {
	oldPath := currentPath + ".old"
	// Best-effort cleanup of a stale .old left by a previous abandoned swap
	// (the file is no longer locked because the previous process is gone).
	_ = os.Remove(oldPath)

	if err := os.Rename(currentPath, oldPath); err != nil {
		return fmt.Errorf("swap: rename current -> old: %w", err)
	}
	if err := os.Rename(newPath, currentPath); err != nil {
		if rbErr := os.Rename(oldPath, currentPath); rbErr != nil {
			return fmt.Errorf("swap: rename new -> current failed (%w); rollback also failed (%v)", err, rbErr)
		}
		return fmt.Errorf("swap: rename new -> current: %w", err)
	}
	return nil
}
