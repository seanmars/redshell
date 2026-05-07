package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newSettingsService(t *testing.T) (*SettingsService, string, string) {
	t.Helper()
	root := t.TempDir()
	settingsPath := filepath.Join(root, ".redshell", "settings.json")
	registryPath := filepath.Join(root, ".redshell", "marketplace.json")
	return NewSettingsServiceWithPaths(settingsPath, registryPath), settingsPath, registryPath
}

func TestSettingsService_DefaultStateForFreshUser(t *testing.T) {
	svc, _, _ := newSettingsService(t)

	state, err := svc.GetSetupState()
	if err != nil {
		t.Fatalf("GetSetupState: %v", err)
	}
	if state.AgentSetupCompleted {
		t.Fatal("expected fresh user setup to be incomplete")
	}
	if len(state.EnabledAgents) != 0 {
		t.Fatalf("expected no agents preselected by default, got %v", state.EnabledAgents)
	}
}

func TestSettingsService_DefaultStateStillRequiresSetupWithoutSettingsFile(t *testing.T) {
	svc, _, registryPath := newSettingsService(t)
	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(registryPath, []byte("[]"), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	state, err := svc.GetSetupState()
	if err != nil {
		t.Fatalf("GetSetupState: %v", err)
	}
	if state.AgentSetupCompleted {
		t.Fatal("expected missing settings file to require setup even when a registry exists")
	}
	if len(state.EnabledAgents) != 0 {
		t.Fatalf("expected no agents preselected without settings file, got %v", state.EnabledAgents)
	}
}

func TestSettingsService_SetEnabledAgentsWritesNormalizedState(t *testing.T) {
	svc, settingsPath, _ := newSettingsService(t)

	if err := svc.SetEnabledAgents([]string{"copilot", "claude", "copilot"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	state, err := svc.GetSetupState()
	if err != nil {
		t.Fatalf("GetSetupState: %v", err)
	}
	if !state.AgentSetupCompleted {
		t.Fatal("expected setup to be marked complete after saving")
	}
	want := []string{"claude", "copilot"}
	if len(state.EnabledAgents) != len(want) {
		t.Fatalf("expected %v, got %v", want, state.EnabledAgents)
	}
	for i, id := range want {
		if state.EnabledAgents[i] != id {
			t.Fatalf("expected %v, got %v", want, state.EnabledAgents)
		}
	}

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var stored SetupState
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !stored.AgentSetupCompleted {
		t.Fatal("expected stored state to be complete")
	}
}

func TestSettingsService_SetEnabledAgentsRejectsEmpty(t *testing.T) {
	svc, _, _ := newSettingsService(t)
	if err := svc.SetEnabledAgents(nil); err == nil {
		t.Fatal("expected empty enabled agent list to fail")
	}
}

func TestSettingsService_IsAgentEnabled(t *testing.T) {
	svc, _, _ := newSettingsService(t)
	if err := svc.SetEnabledAgents([]string{"claude"}); err != nil {
		t.Fatalf("SetEnabledAgents: %v", err)
	}

	enabled, err := svc.IsAgentEnabled("claude")
	if err != nil {
		t.Fatalf("IsAgentEnabled: %v", err)
	}
	if !enabled {
		t.Fatal("expected claude to be enabled")
	}

	enabled, err = svc.IsAgentEnabled("copilot")
	if err != nil {
		t.Fatalf("IsAgentEnabled: %v", err)
	}
	if enabled {
		t.Fatal("expected copilot to be disabled")
	}
}

func TestSettingsService_RejectsUnknownAgent(t *testing.T) {
	svc, _, _ := newSettingsService(t)
	if err := svc.SetEnabledAgents([]string{"unknown"}); err == nil {
		t.Fatal("expected unknown agent to fail")
	}
}
