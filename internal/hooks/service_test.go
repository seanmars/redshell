package hooks

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestService_UnknownAgent(t *testing.T) {
	svc := NewServiceWithRoot(t.TempDir())
	if _, err := svc.ListHooks("vscode-copilot", ListOpts{}); !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("expected ErrUnknownAgent, got %v", err)
	}
}

func TestService_Claude_HappyPath(t *testing.T) {
	home := t.TempDir()
	userBody := `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"echo user"}]}]}}`
	writeFile(t, ClaudeUserSettingsPath(home), userBody)

	pluginPath := filepath.Join(home, ".claude", "plugins", "cache", "off", "wp", "1.0.0")
	writeHookFile(t, pluginPath, `{"hooks":{"SessionStart":[{"hooks":[{"type":"agent","prompt":"go"}]}]}}`)
	seedInstalledPlugins(t, home, map[string][]installedPluginEntry{
		"wp@off": {{Scope: "user", InstallPath: pluginPath}},
	})

	svc := NewServiceWithRoot(home)
	listing, err := svc.ListHooks("claude", ListOpts{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if listing.AgentID != "claude" {
		t.Errorf("AgentID = %q", listing.AgentID)
	}
	if len(listing.Sources) != 2 {
		t.Errorf("Sources = %d, want 2 (user + 1 plugin)", len(listing.Sources))
	}
	if len(listing.Hooks) != 2 {
		t.Errorf("Hooks = %d, want 2", len(listing.Hooks))
	}
	if listing.EmptyReason != EmptyReasonNone {
		t.Errorf("EmptyReason = %q, want empty", listing.EmptyReason)
	}
}

func TestService_Claude_SourcesOrdered_UserBeforeLocalBeforePlugin(t *testing.T) {
	home := t.TempDir()
	body := `{"hooks":{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"x"}]}]}}`
	writeFile(t, ClaudeUserSettingsPath(home), body)
	writeFile(t, ClaudeLocalSettingsPath(home), body)

	pluginAlpha := filepath.Join(home, "p", "alpha")
	pluginBeta := filepath.Join(home, "p", "beta")
	writeHookFile(t, pluginAlpha, body)
	writeHookFile(t, pluginBeta, body)
	seedInstalledPlugins(t, home, map[string][]installedPluginEntry{
		"beta@m":  {{Scope: "user", InstallPath: pluginBeta}},
		"alpha@m": {{Scope: "user", InstallPath: pluginAlpha}},
	})

	svc := NewServiceWithRoot(home)
	listing, _ := svc.ListHooks("claude", ListOpts{})
	if len(listing.Sources) != 4 {
		t.Fatalf("sources = %d, want 4", len(listing.Sources))
	}
	if listing.Sources[0].Kind != SourceUser {
		t.Errorf("[0].Kind = %v, want user", listing.Sources[0].Kind)
	}
	if listing.Sources[1].Kind != SourceLocal {
		t.Errorf("[1].Kind = %v, want local", listing.Sources[1].Kind)
	}
	if listing.Sources[2].Kind != SourcePlugin || listing.Sources[2].Label != "Plugin: alpha@m" {
		t.Errorf("[2] = %+v, want plugin alpha@m", listing.Sources[2])
	}
	if listing.Sources[3].Kind != SourcePlugin || listing.Sources[3].Label != "Plugin: beta@m" {
		t.Errorf("[3] = %+v, want plugin beta@m", listing.Sources[3])
	}
}

func TestService_Claude_EmptySourceIsHidden(t *testing.T) {
	home := t.TempDir()
	// User file present but contains zero hooks.
	writeFile(t, ClaudeUserSettingsPath(home), `{"hooks":{}}`)

	listing, _ := NewServiceWithRoot(home).ListHooks("claude", ListOpts{})
	for _, s := range listing.Sources {
		if s.Kind == SourceUser {
			t.Errorf("empty user source should be hidden, got %+v", s)
		}
	}
}

func TestService_Claude_ParseErrorRendersInline(t *testing.T) {
	home := t.TempDir()
	writeFile(t, ClaudeUserSettingsPath(home), `{not json`)

	listing, _ := NewServiceWithRoot(home).ListHooks("claude", ListOpts{})
	hasUser := false
	for _, s := range listing.Sources {
		if s.Kind == SourceUser {
			hasUser = true
		}
	}
	if !hasUser {
		t.Errorf("user source should remain present so the error has a parent")
	}
	if len(listing.Errors) != 1 {
		t.Errorf("expected 1 source error, got %d: %+v", len(listing.Errors), listing.Errors)
	}
}

func TestService_Claude_DisableAllReported(t *testing.T) {
	home := t.TempDir()
	writeFile(t, ClaudeUserSettingsPath(home), `{"disableAllHooks":true,"hooks":{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"x"}]}]}}`)

	listing, _ := NewServiceWithRoot(home).ListHooks("claude", ListOpts{})
	if len(listing.DisableAll) != 1 {
		t.Fatalf("DisableAll = %d, want 1", len(listing.DisableAll))
	}
	if listing.DisableAll[0].Path != ClaudeUserSettingsPath(home) {
		t.Errorf("DisableAll[0].Path = %q", listing.DisableAll[0].Path)
	}
}

func TestService_Claude_DuplicateAcrossSources(t *testing.T) {
	home := t.TempDir()
	body := `{"hooks":{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"shared.sh"}]}]}}`
	writeFile(t, ClaudeUserSettingsPath(home), body)
	pluginPath := filepath.Join(home, "p", "dup")
	writeHookFile(t, pluginPath, body)
	seedInstalledPlugins(t, home, map[string][]installedPluginEntry{
		"dup@m": {{Scope: "user", InstallPath: pluginPath}},
	})

	listing, _ := NewServiceWithRoot(home).ListHooks("claude", ListOpts{})
	if len(listing.Hooks) != 2 {
		t.Fatalf("hooks = %d, want 2", len(listing.Hooks))
	}
	for _, h := range listing.Hooks {
		if h.DupCount != 2 {
			t.Errorf("hook %+v: DupCount = %d, want 2", h, h.DupCount)
		}
	}
}

func TestService_Claude_NoDuplicateSelfCount(t *testing.T) {
	home := t.TempDir()
	// Two hooks in the SAME source with the same command. DupCount should
	// be 1 because dedup counts distinct sources, not distinct entries.
	body := `{"hooks":{"PreToolUse":[{"matcher":"A","hooks":[{"type":"command","command":"x"}]},{"matcher":"B","hooks":[{"type":"command","command":"x"}]}]}}`
	writeFile(t, ClaudeUserSettingsPath(home), body)

	listing, _ := NewServiceWithRoot(home).ListHooks("claude", ListOpts{})
	if len(listing.Hooks) != 2 {
		t.Fatalf("hooks = %d, want 2", len(listing.Hooks))
	}
	for _, h := range listing.Hooks {
		if h.DupCount != 1 {
			t.Errorf("hook %+v: DupCount = %d, want 1 (same source, no dup)", h, h.DupCount)
		}
	}
}

func TestService_Copilot_EmptyState(t *testing.T) {
	listing, err := NewServiceWithRoot(t.TempDir()).ListHooks("copilot", ListOpts{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if listing.AgentID != "copilot" {
		t.Errorf("AgentID = %q", listing.AgentID)
	}
	if listing.EmptyReason != EmptyReasonCopilotProjectScoped {
		t.Errorf("EmptyReason = %q, want %q", listing.EmptyReason, EmptyReasonCopilotProjectScoped)
	}
	if len(listing.Sources) != 0 || len(listing.Hooks) != 0 {
		t.Errorf("expected empty payload, got %+v", listing)
	}
}

func TestService_Claude_NoFilesProducesEmptyArrays(t *testing.T) {
	// Regression: with zero hook files the JSON contract must still emit
	// arrays (not nil), so the frontend's `.map(...)` does not crash.
	listing, err := NewServiceWithRoot(t.TempDir()).ListHooks("claude", ListOpts{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if listing.Sources == nil || listing.Hooks == nil ||
		listing.Errors == nil || listing.DisableAll == nil {
		t.Errorf("nil slices in empty Claude listing: %+v", listing)
	}
	if len(listing.Sources) != 0 {
		t.Errorf("Sources should be empty, got %+v", listing.Sources)
	}
}

func TestService_Claude_PluginScanErrorRegistersSyntheticSource(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "installed_plugins.json"), []byte("{ not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	listing, _ := NewServiceWithRoot(home).ListHooks("claude", ListOpts{})

	if len(listing.Errors) != 1 {
		t.Fatalf("Errors = %d, want 1", len(listing.Errors))
	}
	errSourceID := listing.Errors[0].SourceID
	matched := false
	for _, s := range listing.Sources {
		if s.ID == errSourceID {
			matched = true
			break
		}
	}
	if !matched {
		t.Errorf("plugin-scan error has no synthetic source parent: errors=%+v sources=%+v", listing.Errors, listing.Sources)
	}
}

func TestService_Copilot_NonEmptyWorkspaceIgnoredNotErrored(t *testing.T) {
	listing, err := NewServiceWithRoot(t.TempDir()).ListHooks("copilot", ListOpts{Workspace: "F:\\some\\ws"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if listing.EmptyReason != EmptyReasonCopilotProjectScoped {
		t.Errorf("non-empty workspace should still produce empty state in v1, got %q", listing.EmptyReason)
	}
}
