package hooks

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedInstalledPlugins writes a v2-shaped installed_plugins.json under
// home/.claude/plugins/ with the supplied entries. Each entry's
// installPath is resolved against home before being written so test code
// can use relative subpaths.
func seedInstalledPlugins(t *testing.T, home string, plugins map[string][]installedPluginEntry) {
	t.Helper()
	dir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	doc := installedPluginsFile{Version: 2, Plugins: plugins}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "installed_plugins.json"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func writeHookFile(t *testing.T, installPath, body string) {
	t.Helper()
	dir := filepath.Join(installPath, "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write hooks.json: %v", err)
	}
}

func TestScanPluginCandidates_HappyPath(t *testing.T) {
	home := t.TempDir()
	withHooks := filepath.Join(home, ".claude", "plugins", "cache", "official", "withhooks", "1.0.0")
	noHooks := filepath.Join(home, ".claude", "plugins", "cache", "official", "nohooks", "1.0.0")

	writeHookFile(t, withHooks, `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"echo"}]}]}}`)
	if err := os.MkdirAll(noHooks, 0o755); err != nil {
		t.Fatalf("mkdir nohooks: %v", err)
	}

	seedInstalledPlugins(t, home, map[string][]installedPluginEntry{
		"withhooks@official": {
			{Scope: "user", InstallPath: withHooks, Version: "1.0.0"},
		},
		"nohooks@official": {
			{Scope: "user", InstallPath: noHooks, Version: "1.0.0"},
		},
	})

	candidates, err := scanPluginCandidates(home)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2 (both pass scan; presence check is in service)", len(candidates))
	}

	var withHooksCand pluginCandidate
	for _, c := range candidates {
		if c.PluginID == "withhooks" {
			withHooksCand = c
		}
	}
	if withHooksCand.Marketplace != "official" {
		t.Errorf("marketplace = %q, want official", withHooksCand.Marketplace)
	}
	if withHooksCand.Label != "Plugin: withhooks@official" {
		t.Errorf("label = %q, want Plugin: withhooks@official", withHooksCand.Label)
	}
	if !strings.HasSuffix(withHooksCand.HookFile, filepath.FromSlash("hooks/hooks.json")) {
		t.Errorf("hookFile = %q, expected hooks/hooks.json suffix", withHooksCand.HookFile)
	}
}

func TestScanPluginCandidates_MultiEntryGetsScopeSuffix(t *testing.T) {
	home := t.TempDir()
	userPath := filepath.Join(home, "scopes", "user")
	projectPath := filepath.Join(home, "scopes", "project")
	if err := os.MkdirAll(userPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	seedInstalledPlugins(t, home, map[string][]installedPluginEntry{
		"shared@market": {
			{Scope: "user", InstallPath: userPath},
			{Scope: "project", InstallPath: projectPath},
		},
	})

	candidates, err := scanPluginCandidates(home)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("got %d, want 2", len(candidates))
	}
	labels := []string{candidates[0].Label, candidates[1].Label}
	wantUser := "Plugin: shared@market (user)"
	wantProject := "Plugin: shared@market (project)"
	hasUser, hasProject := false, false
	for _, l := range labels {
		if l == wantUser {
			hasUser = true
		}
		if l == wantProject {
			hasProject = true
		}
	}
	if !hasUser || !hasProject {
		t.Errorf("labels = %+v, want both %q and %q", labels, wantUser, wantProject)
	}
}

func TestScanPluginCandidates_MarketplaceTreeIsNotScanned(t *testing.T) {
	home := t.TempDir()
	// Create a marketplaces/ tree with a hooks file but DO NOT register
	// it in installed_plugins.json. The scanner should not surface it.
	bogusPath := filepath.Join(home, ".claude", "plugins", "marketplaces", "official", "plugins", "ghost")
	writeHookFile(t, bogusPath, `{"hooks":{"PreToolUse":[]}}`)

	seedInstalledPlugins(t, home, map[string][]installedPluginEntry{})

	candidates, err := scanPluginCandidates(home)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("scanner surfaced %d candidates from marketplaces/ tree, want 0: %+v", len(candidates), candidates)
	}
}

func TestScanPluginCandidates_GitHooksSegmentRejected(t *testing.T) {
	home := t.TempDir()
	// installPath that crosses .git/hooks/ should be rejected by
	// ResolvePluginHookPath, dropping the candidate silently.
	bad := filepath.Join(home, "weird", ".git", "hooks", "post-checkout")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	seedInstalledPlugins(t, home, map[string][]installedPluginEntry{
		"sneaky@x": {{Scope: "user", InstallPath: bad}},
	})

	candidates, err := scanPluginCandidates(home)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("scanner accepted .git/hooks installPath: %+v", candidates)
	}
}

func TestScanPluginCandidates_MalformedFile(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "installed_plugins.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := scanPluginCandidates(home); err == nil {
		t.Errorf("malformed installed_plugins.json should error")
	}
}

func TestScanPluginCandidates_MissingFileReturnsNoError(t *testing.T) {
	home := t.TempDir()
	candidates, err := scanPluginCandidates(home)
	if err != nil {
		t.Errorf("missing file should not error, got %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected zero candidates, got %d", len(candidates))
	}
}

func TestSplitPluginKey(t *testing.T) {
	cases := []struct {
		in            string
		wantID, wantM string
	}{
		{"foo@bar", "foo", "bar"},
		{"weird-id@market", "weird-id", "market"},
		{"with@in@id@last", "with@in@id", "last"},
		{"no-at-sign", "no-at-sign", ""},
	}
	for _, c := range cases {
		gotID, gotM := splitPluginKey(c.in)
		if gotID != c.wantID || gotM != c.wantM {
			t.Errorf("splitPluginKey(%q) = (%q,%q), want (%q,%q)", c.in, gotID, gotM, c.wantID, c.wantM)
		}
	}
}

func TestReadPluginHookFile_MissingReturnsNoError(t *testing.T) {
	home := t.TempDir()
	missing := filepath.Join(home, "absent", "hooks", "hooks.json")
	res, ok, err := readPluginHookFile(missing, "src")
	if err != nil {
		t.Errorf("missing file should not error, got %v", err)
	}
	if ok {
		t.Errorf("ok should be false for missing file")
	}
	if len(res.Hooks) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}

func TestReadPluginHookFile_ParseErrorPropagates(t *testing.T) {
	home := t.TempDir()
	bad := filepath.Join(home, "bad")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	hookFile := filepath.Join(bad, "hooks.json")
	if err := os.WriteFile(hookFile, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := readPluginHookFile(hookFile, "src"); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Errorf("parse error should propagate, got %v", err)
	}
}
