//go:build windows

package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSwapWindows_RenamesCurrentAndNew(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "redshell.exe")
	newp := filepath.Join(dir, "redshell.exe.new")
	if err := os.WriteFile(current, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write current: %v", err)
	}
	if err := os.WriteFile(newp, []byte("v2"), 0o644); err != nil {
		t.Fatalf("write new: %v", err)
	}

	if err := Swap(current, newp); err != nil {
		t.Fatalf("Swap: %v", err)
	}

	got, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current after swap: %v", err)
	}
	if string(got) != "v2" {
		t.Fatalf("current should be v2, got %q", got)
	}
	gotOld, err := os.ReadFile(current + ".old")
	if err != nil {
		t.Fatalf("read .old after swap: %v", err)
	}
	if string(gotOld) != "v1" {
		t.Fatalf(".old should be v1, got %q", gotOld)
	}
	if _, err := os.Stat(newp); !os.IsNotExist(err) {
		t.Fatal(".new file should be consumed by the swap")
	}
}

func TestSwapWindows_RemovesPriorOldBeforeRenaming(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "redshell.exe")
	old := current + ".old"
	newp := filepath.Join(dir, "redshell.exe.new")
	if err := os.WriteFile(current, []byte("v2"), 0o644); err != nil {
		t.Fatalf("write current: %v", err)
	}
	if err := os.WriteFile(old, []byte("stale-v0"), 0o644); err != nil {
		t.Fatalf("write stale old: %v", err)
	}
	if err := os.WriteFile(newp, []byte("v3"), 0o644); err != nil {
		t.Fatalf("write new: %v", err)
	}

	if err := Swap(current, newp); err != nil {
		t.Fatalf("Swap: %v", err)
	}

	gotOld, err := os.ReadFile(old)
	if err != nil {
		t.Fatalf("read .old: %v", err)
	}
	if string(gotOld) != "v2" {
		t.Fatalf(".old should be v2 (the previous current), got %q", gotOld)
	}
}

func TestSwapWindows_FailsCleanlyWhenNewMissing(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "redshell.exe")
	if err := os.WriteFile(current, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write current: %v", err)
	}
	if err := Swap(current, filepath.Join(dir, "missing.new")); err == nil {
		t.Fatal("expected Swap with missing new file to error")
	}
	got, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current after failed swap: %v", err)
	}
	if string(got) != "v1" {
		t.Fatalf("current should be rolled back to v1, got %q", got)
	}
}
