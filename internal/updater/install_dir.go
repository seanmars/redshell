package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// IsWritable returns true when the current process can create and remove a
// file inside dir. Used to detect installer-installed builds (Program Files)
// where auto-update cannot proceed without elevation.
func IsWritable(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	probe := filepath.Join(dir, fmt.Sprintf("redshell-update-probe-%d", time.Now().UnixNano()))
	f, err := os.Create(probe)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return true
}
