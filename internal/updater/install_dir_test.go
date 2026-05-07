package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWritableTempDir(t *testing.T) {
	if !IsWritable(t.TempDir()) {
		t.Fatal("expected freshly-created TempDir to be writable")
	}
}

func TestIsWritableNonExistent(t *testing.T) {
	if IsWritable(filepath.Join(t.TempDir(), "does-not-exist")) {
		t.Fatal("expected non-existent dir to be unwritable")
	}
}

func TestIsWritableRejectsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if IsWritable(path) {
		t.Fatal("expected file path to be unwritable as a directory")
	}
}
