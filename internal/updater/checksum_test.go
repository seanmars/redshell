package updater

import (
	"strings"
	"testing"
)

func TestParseChecksumsValid(t *testing.T) {
	body := `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  redshell-windows-amd64.exe
da39a3ee5e6b4b0d3255bfef95601890afd80709da39a3ee5e6b4b0d3255bfef  RedShell-amd64-installer.exe
`
	got, err := ParseChecksums(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseChecksums: %v", err)
	}
	if want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"; got["redshell-windows-amd64.exe"] != want {
		t.Fatalf("portable hash mismatch: got %q want %q", got["redshell-windows-amd64.exe"], want)
	}
	if _, ok := got["RedShell-amd64-installer.exe"]; !ok {
		t.Fatal("installer entry missing")
	}
}

func TestParseChecksumsAcceptsExtraWhitespace(t *testing.T) {
	body := "  abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\t  redshell-windows-amd64.exe  \n"
	got, err := ParseChecksums(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseChecksums: %v", err)
	}
	if got["redshell-windows-amd64.exe"] == "" {
		t.Fatal("expected entry parsed despite mixed whitespace")
	}
}

func TestParseChecksumsAcceptsBinaryStarPrefix(t *testing.T) {
	body := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  *redshell-windows-amd64.exe\n"
	got, err := ParseChecksums(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseChecksums: %v", err)
	}
	if _, ok := got["redshell-windows-amd64.exe"]; !ok {
		t.Fatal("expected '*'-prefixed filename to be normalized")
	}
}

func TestParseChecksumsSkipsCommentsAndBlankLines(t *testing.T) {
	body := `# A comment

abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  redshell-windows-amd64.exe

# Another comment
`
	got, err := ParseChecksums(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseChecksums: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
}

func TestParseChecksumsRejectsMalformedHash(t *testing.T) {
	body := "tooshort  file.exe\n"
	if _, err := ParseChecksums(strings.NewReader(body)); err == nil {
		t.Fatal("expected short hash to error")
	}
}

func TestParseChecksumsRejectsSingleField(t *testing.T) {
	body := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789\n"
	if _, err := ParseChecksums(strings.NewReader(body)); err == nil {
		t.Fatal("expected line without filename to error")
	}
}

func TestParseChecksumsRejectsEmptyFile(t *testing.T) {
	if _, err := ParseChecksums(strings.NewReader("")); err == nil {
		t.Fatal("expected empty checksums to error")
	}
	if _, err := ParseChecksums(strings.NewReader("\n# only comments\n")); err == nil {
		t.Fatal("expected whitespace/comments-only to error")
	}
}
