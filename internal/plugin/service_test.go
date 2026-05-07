package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"redshell/internal/agent"
	"redshell/internal/marketplace"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestManifestParser_OptionalFieldsIgnored_Claude(t *testing.T) {
	var m marketplaceManifest
	if err := json.Unmarshal(readFixture(t, "claude-marketplace.json"), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Name != "Example Claude Marketplace" {
		t.Errorf("unexpected name: %q", m.Name)
	}
	if len(m.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(m.Plugins))
	}
	if m.Plugins[0].Name != "alpha" || m.Plugins[0].Source != "./plugins/alpha" || m.Plugins[0].Description != "First plugin" {
		t.Errorf("alpha fields wrong: %+v", m.Plugins[0])
	}
}

func TestManifestParser_OptionalFieldsIgnored_Copilot(t *testing.T) {
	var m marketplaceManifest
	if err := json.Unmarshal(readFixture(t, "copilot-marketplace.json"), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(m.Plugins))
	}
	names := []string{m.Plugins[0].Name, m.Plugins[1].Name}
	sort.Strings(names)
	if names[0] != "spark" || names[1] != "workiq" {
		t.Errorf("unexpected plugin names: %v", names)
	}
}

func TestManifestParser_MalformedJSON(t *testing.T) {
	var m marketplaceManifest
	if err := json.Unmarshal(readFixture(t, "malformed.json"), &m); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

// newCacheBackedService builds a plugin.Service + marketplace.Service pair
// pointed at an isolated TempDir. Caller seeds cache directories directly.
func newCacheBackedService(t *testing.T, entries []marketplace.Marketplace) (*Service, *marketplace.Service, *agent.SettingsService, string) {
	t.Helper()
	root := t.TempDir()
	registryPath := filepath.Join(root, "marketplace.json")
	cacheRoot := filepath.Join(root, ".cache")
	mktSvc := marketplace.NewServiceWithCacheRoot(registryPath, cacheRoot)
	settingsSvc := agent.NewSettingsServiceWithPaths(filepath.Join(root, ".redshell", "settings.json"), registryPath)

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(registryPath, data, 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	if err := settingsSvc.SetEnabledAgents(agent.SupportedAgentIDs()); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	svc := NewService(mktSvc, agent.NewService(), settingsSvc)
	return svc, mktSvc, settingsSvc, cacheRoot
}

func seedCache(t *testing.T, mktSvc *marketplace.Service, id string, files map[string]string) {
	t.Helper()
	dir := mktSvc.CacheDir(id)
	for relPath, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", relPath, err)
		}
	}
}

func TestFetchForAgent_RequiredFieldsEnforced(t *testing.T) {
	entries := []marketplace.Marketplace{
		{ID: "gh::owner@repo", URL: "https://github.com/owner/repo"},
	}
	svc, mktSvc, _, _ := newCacheBackedService(t, entries)

	partial := `{
		"name": "Partial",
		"plugins": [
			{"name": "no-source"},
			{"source": "./plugins/no-name"},
			{"name": "good", "source": "./plugins/good", "description": "keeper"}
		]
	}`
	seedCache(t, mktSvc, "gh::owner@repo", map[string]string{
		".claude-plugin/marketplace.json": partial,
	})

	got, err := svc.fetchForAgent(entries[0], "claude")
	if err != nil {
		t.Fatalf("fetchForAgent: %v", err)
	}
	if len(got) != 1 || got[0].Name != "good" {
		t.Fatalf("expected 1 plugin named 'good', got %+v", got)
	}
	if got[0].InstallName != "good@Partial" {
		t.Errorf("unexpected install name: %q", got[0].InstallName)
	}
}

func TestIsMarketplaceRegistered_CopilotMatchesMarketplaces(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, readFixture(t, "copilot-config.json"), 0o644); err != nil {
		t.Fatalf("write copilot config: %v", err)
	}

	if !isMarketplaceRegistered(configPath, "https://github.com/seanmars/ai-tools.git", "copilot") {
		t.Error("expected registered marketplace to be detected (with .git suffix)")
	}
	if !isMarketplaceRegistered(configPath, "https://github.com/seanmars/ai-tools", "copilot") {
		t.Error("expected registered marketplace to be detected (without .git suffix)")
	}
	if isMarketplaceRegistered(configPath, "https://github.com/other/repo.git", "copilot") {
		t.Error("expected unrelated marketplace URL to be reported as not registered")
	}
}

func TestReadCopilotInstalled_ReadsInstalledPluginsKey_SnakeCase(t *testing.T) {
	home := t.TempDir()
	copilotDir := filepath.Join(home, ".copilot")
	if err := os.MkdirAll(copilotDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(copilotDir, "config.json"), readFixture(t, "copilot-config.json"), 0o644); err != nil {
		t.Fatalf("write copilot config: %v", err)
	}

	got, err := readCopilotInstalled(home)
	if err != nil {
		t.Fatalf("readCopilotInstalled: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 installed plugin, got %d (%+v)", len(got), got)
	}
	want := InstalledPlugin{
		DisplayName:     "code-review-csharp@chineseagamer-plugins",
		UninstallName:   "code-review-csharp@chineseagamer-plugins",
		Agent:           "copilot",
		MarketplaceName: "chineseagamer-plugins",
	}
	if got[0] != want {
		t.Errorf("unexpected installed plugin:\n got:  %+v\n want: %+v", got[0], want)
	}
}

func TestFetchAll_CacheMiss(t *testing.T) {
	claude := string(readFixture(t, "claude-marketplace.json"))
	copilot := string(readFixture(t, "copilot-marketplace.json"))

	entries := []marketplace.Marketplace{
		{ID: "gh::full@repo", URL: "https://github.com/full/repo"},
		{ID: "gh::claude-only@repo", URL: "https://github.com/claude-only/repo"},
	}
	svc, mktSvc, _, _ := newCacheBackedService(t, entries)

	seedCache(t, mktSvc, "gh::full@repo", map[string]string{
		".claude-plugin/marketplace.json": claude,
		".github/plugin/marketplace.json": copilot,
	})
	seedCache(t, mktSvc, "gh::claude-only@repo", map[string]string{
		".claude-plugin/marketplace.json": claude,
	})

	result := svc.FetchAll()

	var fullClaude, fullCopilot, claudeOnlyClaude, claudeOnlyCopilot int
	for _, p := range result.Plugins {
		switch p.Marketplace + "/" + p.Agent {
		case "gh::full@repo/claude":
			fullClaude++
		case "gh::full@repo/copilot":
			fullCopilot++
		case "gh::claude-only@repo/claude":
			claudeOnlyClaude++
		case "gh::claude-only@repo/copilot":
			claudeOnlyCopilot++
		}
	}
	if fullClaude != 2 || fullCopilot != 2 {
		t.Errorf("gh::full@repo: expected 2+2 plugins, got claude=%d copilot=%d", fullClaude, fullCopilot)
	}
	if claudeOnlyClaude != 2 || claudeOnlyCopilot != 0 {
		t.Errorf("gh::claude-only@repo: expected 2 claude + 0 copilot, got claude=%d copilot=%d", claudeOnlyClaude, claudeOnlyCopilot)
	}

	var sawCacheMissing bool
	for _, e := range result.Errors {
		if strings.HasPrefix(e, "[gh::claude-only@repo/copilot]") && strings.Contains(e, "cache missing") {
			sawCacheMissing = true
		}
	}
	if !sawCacheMissing {
		t.Errorf("expected [gh::claude-only@repo/copilot] cache-missing error, got: %v", result.Errors)
	}
}

func TestFetchAll_OnlyEnabledAgents(t *testing.T) {
	claude := string(readFixture(t, "claude-marketplace.json"))
	copilot := string(readFixture(t, "copilot-marketplace.json"))

	entries := []marketplace.Marketplace{
		{ID: "gh::full@repo", URL: "https://github.com/full/repo"},
	}
	svc, mktSvc, settingsSvc, _ := newCacheBackedService(t, entries)
	if err := settingsSvc.SetEnabledAgents([]string{"claude"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	seedCache(t, mktSvc, "gh::full@repo", map[string]string{
		".claude-plugin/marketplace.json": claude,
		".github/plugin/marketplace.json": copilot,
	})

	result := svc.FetchAll()
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
	for _, p := range result.Plugins {
		if p.Agent != "claude" {
			t.Fatalf("expected only claude plugins, got %+v", result.Plugins)
		}
	}
}

func TestListInstalled_RejectsDisabledAgent(t *testing.T) {
	svc, _, settingsSvc, _ := newCacheBackedService(t, nil)
	if err := settingsSvc.SetEnabledAgents([]string{"claude"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	if _, err := svc.ListInstalled("copilot"); err == nil || !strings.Contains(err.Error(), "agent is disabled: copilot") {
		t.Fatalf("expected disabled-agent error, got %v", err)
	}
}

func TestInstall_RejectsDisabledAgentBeforeShellingOut(t *testing.T) {
	svc, _, settingsSvc, _ := newCacheBackedService(t, nil)
	if err := settingsSvc.SetEnabledAgents([]string{"claude"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	err := svc.Install("copilot", nil, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "agent is disabled: copilot") {
		t.Fatalf("expected disabled-agent error, got %v", err)
	}
}

func TestUninstall_RejectsDisabledAgentBeforeShellingOut(t *testing.T) {
	svc, _, settingsSvc, _ := newCacheBackedService(t, nil)
	if err := settingsSvc.SetEnabledAgents([]string{"claude"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	err := svc.Uninstall("copilot", "plugin@market")
	if err == nil || !strings.Contains(err.Error(), "agent is disabled: copilot") {
		t.Fatalf("expected disabled-agent error, got %v", err)
	}
}

// stubStreamingRunner replaces runAgentCmdStreaming for the duration of a test.
// fn is invoked exactly once per call with the (agentID, args, stdoutFn) the
// service handed it; the returned error is propagated as the runner result.
func stubStreamingRunner(t *testing.T, fn func(agentID string, args []string, stdoutFn func(string)) error) {
	t.Helper()
	orig := runAgentCmdStreaming
	runAgentCmdStreaming = fn
	t.Cleanup(func() { runAgentCmdStreaming = orig })
}

func TestUpdateAgentMarketplaces_AllSucceed(t *testing.T) {
	svc, _, _, _ := newCacheBackedService(t, nil)

	var calls []string
	stubStreamingRunner(t, func(agentID string, args []string, stdoutFn func(string)) error {
		calls = append(calls, agentID+" "+strings.Join(args, " "))
		stdoutFn("Updated " + agentID)
		return nil
	})

	var logLines []string
	res := svc.UpdateAgentMarketplaces(func(s string) { logLines = append(logLines, s) })

	if len(res.Outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %d (%+v)", len(res.Outcomes), res.Outcomes)
	}
	for _, o := range res.Outcomes {
		if !o.OK {
			t.Errorf("expected OK for %s, got error %q", o.AgentID, o.Error)
		}
		if o.Error != "" {
			t.Errorf("expected empty error for %s, got %q", o.AgentID, o.Error)
		}
	}

	wantCalls := map[string]bool{
		"claude plugin marketplace update":  true,
		"copilot plugin marketplace update": true,
	}
	for _, c := range calls {
		if !wantCalls[c] {
			t.Errorf("unexpected runner call: %q", c)
		}
	}
	if len(calls) != len(wantCalls) {
		t.Errorf("expected %d runner calls, got %d (%v)", len(wantCalls), len(calls), calls)
	}

	var sawClaude, sawCopilot bool
	for _, line := range logLines {
		if line == "[claude] Updated claude" {
			sawClaude = true
		}
		if line == "[copilot] Updated copilot" {
			sawCopilot = true
		}
	}
	if !sawClaude || !sawCopilot {
		t.Errorf("expected prefixed log lines for both agents, got %v", logLines)
	}
}

func TestUpdateAgentMarketplaces_OneFailingDoesNotAbortOthers(t *testing.T) {
	svc, _, _, _ := newCacheBackedService(t, nil)

	stubStreamingRunner(t, func(agentID string, args []string, stdoutFn func(string)) error {
		if agentID == "claude" {
			return errors.New("claude plugin marketplace update: boom")
		}
		stdoutFn("ok")
		return nil
	})

	var logLines []string
	res := svc.UpdateAgentMarketplaces(func(s string) { logLines = append(logLines, s) })

	if len(res.Outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %+v", res.Outcomes)
	}
	byAgent := map[string]AgentUpdateOutcome{}
	for _, o := range res.Outcomes {
		byAgent[o.AgentID] = o
	}
	if byAgent["claude"].OK {
		t.Errorf("expected claude failure, got OK")
	}
	if !strings.HasPrefix(byAgent["claude"].Error, "[claude] ") {
		t.Errorf("claude error missing [claude] prefix: %q", byAgent["claude"].Error)
	}
	if !strings.Contains(byAgent["claude"].Error, "boom") {
		t.Errorf("claude error missing underlying message: %q", byAgent["claude"].Error)
	}
	if !byAgent["copilot"].OK {
		t.Errorf("expected copilot success, got error %q", byAgent["copilot"].Error)
	}

	var sawClaudeError bool
	for _, line := range logLines {
		if strings.HasPrefix(line, "[claude] ") && strings.Contains(line, "boom") {
			sawClaudeError = true
		}
	}
	if !sawClaudeError {
		t.Errorf("expected claude error to be forwarded to logFn, got %v", logLines)
	}
}

func TestUpdateAgentMarketplace_RejectsDisabledAgent(t *testing.T) {
	svc, _, settingsSvc, _ := newCacheBackedService(t, nil)
	if err := settingsSvc.SetEnabledAgents([]string{"claude"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	var calls int
	stubStreamingRunner(t, func(string, []string, func(string)) error {
		calls++
		return nil
	})

	outcome := svc.UpdateAgentMarketplace("copilot", nil)
	if outcome.OK {
		t.Fatalf("expected disabled-agent failure, got OK")
	}
	if !strings.Contains(outcome.Error, "agent is disabled: copilot") {
		t.Errorf("expected disabled-agent error message, got %q", outcome.Error)
	}
	if calls != 0 {
		t.Errorf("expected runner not to be called for disabled agent, got %d call(s)", calls)
	}
}

func TestUpdateAgentMarketplaces_SkipsDisabledAgents(t *testing.T) {
	svc, _, settingsSvc, _ := newCacheBackedService(t, nil)
	if err := settingsSvc.SetEnabledAgents([]string{"claude"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	var calledAgents []string
	stubStreamingRunner(t, func(agentID string, args []string, stdoutFn func(string)) error {
		calledAgents = append(calledAgents, agentID)
		return nil
	})

	res := svc.UpdateAgentMarketplaces(nil)

	if len(res.Outcomes) != 1 || res.Outcomes[0].AgentID != "claude" || !res.Outcomes[0].OK {
		t.Fatalf("expected 1 OK outcome for claude, got %+v", res.Outcomes)
	}
	if len(calledAgents) != 1 || calledAgents[0] != "claude" {
		t.Errorf("expected runner to be called only for claude, got %v", calledAgents)
	}
}
