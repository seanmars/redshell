package updater

import (
	"errors"
	"io/fs"
	"os"
)

// CleanupStale removes any update-flow sibling left next to exePath from a
// previous swap or interrupted download:
//
//	<exe>.old           - prior version waiting to be reaped
//	<exe>.new           - verified-but-not-swapped binary (e.g. crash post-verify)
//	<exe>.new.partial   - in-flight download interrupted before rename
//	<exe>.partial       - legacy / direct-write partial (defensive)
//
// Missing files are silent; other errors are returned so the caller can
// decide whether to surface them.
func CleanupStale(exePath string) error {
	for _, suffix := range []string{".old", ".new", ".new.partial", ".partial"} {
		path := exePath + suffix
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}
