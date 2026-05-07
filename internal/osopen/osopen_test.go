package osopen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenPath_ExpandsTilde(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	target := filepath.Join(tmp, ".redshell-test")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var captured string
	swap := dispatch
	dispatch = func(absPath string) error {
		captured = absPath
		return nil
	}
	t.Cleanup(func() { dispatch = swap })

	if err := OpenPath("~/.redshell-test"); err != nil {
		t.Fatalf("OpenPath: %v", err)
	}

	wantSuffix := filepath.Join(".redshell-test")
	if !strings.HasSuffix(captured, wantSuffix) {
		t.Fatalf("expected dispatched path to end with %q, got %q", wantSuffix, captured)
	}
	if strings.Contains(captured, "~") {
		t.Fatalf("expected tilde to be expanded, got %q", captured)
	}
}

func TestOpenPath_MissingPathReturnsError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	called := false
	swap := dispatch
	dispatch = func(absPath string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { dispatch = swap })

	missing := filepath.Join(tmp, "does-not-exist")
	err := OpenPath(missing)
	if err == nil {
		t.Fatalf("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected 'does not exist' in error, got %q", err.Error())
	}
	if called {
		t.Fatalf("dispatcher should not be called for missing path")
	}
}

func TestOpenPath_ExpandsBareTilde(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var captured string
	swap := dispatch
	dispatch = func(absPath string) error {
		captured = absPath
		return nil
	}
	t.Cleanup(func() { dispatch = swap })

	if err := OpenPath("~"); err != nil {
		t.Fatalf("OpenPath: %v", err)
	}
	if captured == "" || strings.Contains(captured, "~") {
		t.Fatalf("expected tilde to expand to home, got %q", captured)
	}
}
