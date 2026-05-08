package updater

import "testing"

// withBuildKind temporarily overrides the package-level BuildKind for the
// duration of a test, restoring the previous value via t.Cleanup. Callers
// using this helper MUST NOT call t.Parallel() because BuildKind is shared
// state across the package.
func withBuildKind(t *testing.T, kind string) {
	t.Helper()
	prev := BuildKind
	BuildKind = kind
	t.Cleanup(func() { BuildKind = prev })
}

func TestBuildKind_DefaultIsPortable(t *testing.T) {
	if BuildKind != "portable" {
		t.Fatalf("default BuildKind: got %q want portable", BuildKind)
	}
	if !IsPortable() {
		t.Fatal("IsPortable should be true for default value")
	}
	if IsInstaller() {
		t.Fatal("IsInstaller should be false for default value")
	}
}

func TestBuildKind_InstallerHelpers(t *testing.T) {
	withBuildKind(t, "installer")
	if !IsInstaller() {
		t.Fatal("IsInstaller should be true when BuildKind=installer")
	}
	if IsPortable() {
		t.Fatal("IsPortable should be false when BuildKind=installer")
	}
}

func TestBuildKind_UnknownValueTreatedAsPortable(t *testing.T) {
	withBuildKind(t, "something-bogus")
	if IsInstaller() {
		t.Fatal("IsInstaller should be false for unknown value")
	}
	if !IsPortable() {
		t.Fatal("IsPortable should be true for unknown value (safe fallback)")
	}
}
