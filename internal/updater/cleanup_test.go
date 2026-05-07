package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupStaleRemovesAllUpdateArtifacts(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "redshell.exe")
	if err := os.WriteFile(exe, []byte("running"), 0o644); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	for _, suffix := range []string{".old", ".new", ".new.partial", ".partial"} {
		if err := os.WriteFile(exe+suffix, []byte("stale"+suffix), 0o644); err != nil {
			t.Fatalf("write %s: %v", suffix, err)
		}
	}

	if err := CleanupStale(exe); err != nil {
		t.Fatalf("CleanupStale: %v", err)
	}

	for _, suffix := range []string{".old", ".new", ".new.partial", ".partial"} {
		if _, err := os.Stat(exe + suffix); !os.IsNotExist(err) {
			t.Fatalf("%s should be removed", suffix)
		}
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("exe itself must remain: %v", err)
	}
}

func TestCleanupStaleSilentOnMissingFiles(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "redshell.exe")
	if err := CleanupStale(exe); err != nil {
		t.Fatalf("CleanupStale should be silent on missing files, got: %v", err)
	}
}
