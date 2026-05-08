package sessionhistory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type capturedLaunch struct {
	cli, sessionID, cwd string
}

func stubLauncher(t *testing.T) *capturedLaunch {
	t.Helper()
	c := &capturedLaunch{}
	prev := launchResumeTerminal
	launchResumeTerminal = func(cli, sessionID, cwd string) error {
		c.cli = cli
		c.sessionID = sessionID
		c.cwd = cwd
		return nil
	}
	t.Cleanup(func() { launchResumeTerminal = prev })
	return c
}

func TestResumeSession_StripsClaudePathPrefix(t *testing.T) {
	got := stubLauncher(t)
	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	projectDir := t.TempDir()
	err := svc.ResumeSession(
		"claude",
		"D--workspace-seanmars-my-agent-plugins/a21e4cc8-bbcc-4e4a-bb98-f79404e202ec",
		projectDir,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.cli != "claude" {
		t.Errorf("cli = %q, want %q", got.cli, "claude")
	}
	if got.sessionID != "a21e4cc8-bbcc-4e4a-bb98-f79404e202ec" {
		t.Errorf("sessionID = %q, want UUID-only basename", got.sessionID)
	}
	if got.cwd != filepath.Clean(projectDir) {
		t.Errorf("cwd = %q, want %q", got.cwd, filepath.Clean(projectDir))
	}
}

func TestResumeSession_PassesBareCopilotID(t *testing.T) {
	got := stubLauncher(t)
	svc := NewServiceWithRoots(map[string]string{"copilot": t.TempDir()})
	err := svc.ResumeSession("copilot", "cb1f5a48-3c8e-4b1f-9a01-a13f1b1d77a5", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.sessionID != "cb1f5a48-3c8e-4b1f-9a01-a13f1b1d77a5" {
		t.Errorf("sessionID = %q, want bare UUID", got.sessionID)
	}
	if got.cwd != "" {
		t.Errorf("cwd = %q, want empty string when caller passes empty", got.cwd)
	}
}

func TestResumeSession_RejectsUnknownAgent(t *testing.T) {
	prev := launchResumeTerminal
	launchResumeTerminal = func(string, string, string) error {
		t.Fatal("launcher must not run for unknown agent")
		return nil
	}
	defer func() { launchResumeTerminal = prev }()

	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	err := svc.ResumeSession("nonexistent", "abc", "")
	if !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("err = %v, want ErrUnknownAgent", err)
	}
}

func TestResumeSession_RejectsEmptySessionID(t *testing.T) {
	prev := launchResumeTerminal
	launchResumeTerminal = func(string, string, string) error {
		t.Fatal("launcher must not run for empty session id")
		return nil
	}
	defer func() { launchResumeTerminal = prev }()

	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	err := svc.ResumeSession("claude", "", "")
	if !errors.Is(err, ErrInvalidSessionID) {
		t.Errorf("err = %v, want ErrInvalidSessionID", err)
	}
}

func TestResumeSession_RejectsInjectionAttempts(t *testing.T) {
	cases := []string{
		"abc; rm -rf /",
		"abc`whoami`",
		"abc$env:PATH",
		"abc|notepad",
		"abc&calc",
		"abc'\"$(echo)",
		"path/with spaces",
		"path/sub;cmd",
	}
	prev := launchResumeTerminal
	launchResumeTerminal = func(string, string, string) error {
		t.Fatal("launcher must not run for malicious session id")
		return nil
	}
	defer func() { launchResumeTerminal = prev }()

	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	for _, id := range cases {
		err := svc.ResumeSession("claude", id, "")
		if !errors.Is(err, ErrInvalidSessionID) {
			t.Errorf("session id %q: err = %v, want ErrInvalidSessionID", id, err)
		}
	}
}

func TestResumeSession_ForwardsExistingDirectoryAsCwd(t *testing.T) {
	got := stubLauncher(t)
	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	projectDir := t.TempDir()
	err := svc.ResumeSession("claude", "abc12345", projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.cwd != filepath.Clean(projectDir) {
		t.Errorf("cwd = %q, want existing dir %q", got.cwd, filepath.Clean(projectDir))
	}
}

func TestResumeSession_ErrorsWhenCwdMissing(t *testing.T) {
	prev := launchResumeTerminal
	launchResumeTerminal = func(string, string, string) error {
		t.Fatal("launcher must not run when cwd is missing")
		return nil
	}
	defer func() { launchResumeTerminal = prev }()

	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	err := svc.ResumeSession("claude", "abc12345", missing)
	if !errors.Is(err, ErrProjectCwdMissing) {
		t.Fatalf("err = %v, want ErrProjectCwdMissing", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error message %q must include the offending path %q", err.Error(), missing)
	}
}

func TestResumeSession_ErrorsWhenCwdIsRelative(t *testing.T) {
	prev := launchResumeTerminal
	launchResumeTerminal = func(string, string, string) error {
		t.Fatal("launcher must not run for relative cwd")
		return nil
	}
	defer func() { launchResumeTerminal = prev }()

	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	err := svc.ResumeSession("claude", "abc12345", "relative/path")
	if !errors.Is(err, ErrProjectCwdMissing) {
		t.Errorf("err = %v, want ErrProjectCwdMissing", err)
	}
}

func TestResumeSession_ErrorsWhenCwdIsFile(t *testing.T) {
	prev := launchResumeTerminal
	launchResumeTerminal = func(string, string, string) error {
		t.Fatal("launcher must not run when cwd points at a file")
		return nil
	}
	defer func() { launchResumeTerminal = prev }()

	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})

	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("hi"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	err := svc.ResumeSession("claude", "abc12345", filePath)
	if !errors.Is(err, ErrProjectCwdMissing) {
		t.Errorf("err = %v, want ErrProjectCwdMissing", err)
	}
}

func TestResumeSession_LaunchesWhenCwdIsEmpty(t *testing.T) {
	got := stubLauncher(t)
	svc := NewServiceWithRoots(map[string]string{"claude": t.TempDir()})
	err := svc.ResumeSession("claude", "abc12345", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.cwd != "" {
		t.Errorf("cwd = %q, want empty (no working-dir hint) for empty input", got.cwd)
	}
	if got.sessionID != "abc12345" {
		t.Errorf("sessionID = %q, want abc12345", got.sessionID)
	}
}
