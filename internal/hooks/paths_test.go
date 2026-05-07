package hooks

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudePaths_JoinedUnderHome(t *testing.T) {
	home := t.TempDir()

	user := ClaudeUserSettingsPath(home)
	if !strings.HasPrefix(user, home) {
		t.Errorf("user path %q does not start with home %q", user, home)
	}
	if !strings.HasSuffix(user, filepath.FromSlash(".claude/settings.json")) {
		t.Errorf("user path %q does not end with .claude/settings.json", user)
	}

	local := ClaudeLocalSettingsPath(home)
	if !strings.HasSuffix(local, filepath.FromSlash(".claude/settings.local.json")) {
		t.Errorf("local path %q does not end with .claude/settings.local.json", local)
	}

	installed := ClaudeInstalledPluginsPath(home)
	if !strings.HasSuffix(installed, filepath.FromSlash(".claude/plugins/installed_plugins.json")) {
		t.Errorf("installed path %q does not end with installed_plugins.json", installed)
	}
}

func TestResolvePluginHookPath_Happy(t *testing.T) {
	install := filepath.Join(t.TempDir(), "claude-plugins-official", "superpowers", "5.1.0")

	got, err := ResolvePluginHookPath(install)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := filepath.Join(install, "hooks", "hooks.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolvePluginHookPath_RejectsGitHooksSegment(t *testing.T) {
	cases := []string{
		filepath.Join("/tmp", ".git", "hooks"),
		filepath.Join("/tmp", "plugins", ".git", "hooks", "post-checkout"),
		filepath.Join("C:", "Users", "x", ".git", "hooks", "deeper"),
	}
	for _, c := range cases {
		if _, err := ResolvePluginHookPath(c); !errors.Is(err, ErrPluginPathRejected) {
			t.Errorf("ResolvePluginHookPath(%q): want ErrPluginPathRejected, got %v", c, err)
		}
	}
}

func TestResolvePluginHookPath_EmptyRejected(t *testing.T) {
	if _, err := ResolvePluginHookPath(""); !errors.Is(err, ErrPluginPathRejected) {
		t.Errorf("empty installPath: want ErrPluginPathRejected, got %v", err)
	}
}

func TestCopilotPolicyPath_EmptyWorkspaceReturnsEmpty(t *testing.T) {
	if got := CopilotPolicyPath(""); got != "" {
		t.Errorf("empty workspace: got %q, want empty", got)
	}
}

func TestCopilotPolicyPath_JoinsBeneathWorkspace(t *testing.T) {
	ws := t.TempDir()
	got := CopilotPolicyPath(ws)
	if !strings.HasPrefix(got, ws) {
		t.Errorf("path %q does not start with workspace %q", got, ws)
	}
	if !strings.HasSuffix(got, filepath.FromSlash(".github/hooks/copilot-cli-policy.json")) {
		t.Errorf("path %q does not end with .github/hooks/copilot-cli-policy.json", got)
	}
}
