package updater

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// installerDownloadPath returns the path the installer pathway downloads
// the NSIS installer to. It lives in the user-writable system temp dir
// rather than next to the exe because for installer builds the install
// directory is admin-only and the download happens at medium integrity
// (before UAC elevation). The elevated installer child can still read
// from %TEMP% because it runs as the same user.
//
// The filename MUST end in .exe so Windows ShellExecute can find a verb
// handler. ShellExecute dispatches on file extension (asks the registry
// "what handles .<ext> with the <verb> verb?"); a non-.exe extension like
// .new returns SE_ERR_NOASSOC ("No application is associated with the
// specified file for this operation") even though the file is a valid PE
// binary. The portable rename-trick swap doesn't have this problem because
// it uses os.Rename + exec.Command, which call CreateProcess directly and
// inspect the PE header rather than the extension.
//
// Both service.installInstaller and CleanupStale must agree on this path
// — if you change it here, the next start's cleanup loses the leftover.
func installerDownloadPath() string {
	return filepath.Join(os.TempDir(), "redshell-installer.exe")
}

// CleanupStale removes any update-flow sibling left from a previous swap
// or interrupted download:
//
//	<exe>.old           - prior portable version waiting to be reaped
//	<exe>.new           - verified-but-not-swapped portable binary
//	<exe>.new.partial   - in-flight portable download interrupted before rename
//	<exe>.partial       - legacy / direct-write partial (defensive)
//	%TEMP%/redshell-installer.exe          - installer-pathway downloaded NSIS payload
//	%TEMP%/redshell-installer.exe.partial  - installer-pathway in-flight download
//	%TEMP%/redshell-installer.new          - legacy: pre-fix installer download (.new extension)
//	%TEMP%/redshell-installer.new.partial  - legacy: pre-fix in-flight download
//
// The two legacy entries are defensive: the first published version of the
// installer pathway used a .new extension, which broke ShellExecute. Users
// upgrading from that version may have leftovers we should reap.
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
	tempInstaller := installerDownloadPath()
	tempDir := filepath.Dir(tempInstaller)
	candidates := []string{
		tempInstaller,
		tempInstaller + ".partial",
		filepath.Join(tempDir, "redshell-installer.new"),
		filepath.Join(tempDir, "redshell-installer.new.partial"),
	}
	for _, path := range candidates {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}
