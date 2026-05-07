package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type SetupState struct {
	EnabledAgents       []string `json:"enabledAgents"`
	AgentSetupCompleted bool     `json:"agentSetupCompleted"`
}

type SettingsService struct {
	filePath     string
	registryPath string
}

func NewSettingsService() *SettingsService {
	home, _ := os.UserHomeDir()
	return &SettingsService{
		filePath:     filepath.Join(home, ".redshell", "settings.json"),
		registryPath: filepath.Join(home, ".redshell", "marketplace.json"),
	}
}

func NewSettingsServiceWithPaths(filePath, registryPath string) *SettingsService {
	return &SettingsService{
		filePath:     filePath,
		registryPath: registryPath,
	}
}

func SupportedAgentIDs() []string {
	specs := agentSpecs()
	ids := make([]string, 0, len(specs))
	for _, spec := range specs {
		ids = append(ids, spec.id)
	}
	return ids
}

func (s *SettingsService) GetSetupState() (SetupState, error) {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return s.defaultState(), nil
	}
	if err != nil {
		return SetupState{}, err
	}

	var state SetupState
	if err := json.Unmarshal(data, &state); err != nil {
		return SetupState{}, fmt.Errorf("parse agent settings: %w", err)
	}

	normalized, err := normalizeEnabledAgents(state.EnabledAgents)
	if err != nil {
		return SetupState{}, err
	}
	state.EnabledAgents = normalized
	if state.AgentSetupCompleted && len(state.EnabledAgents) == 0 {
		return SetupState{}, errors.New("agent settings must enable at least one agent")
	}
	return state, nil
}

func (s *SettingsService) GetEnabledAgents() ([]string, error) {
	state, err := s.GetSetupState()
	if err != nil {
		return nil, err
	}
	return state.EnabledAgents, nil
}

func (s *SettingsService) SetEnabledAgents(agentIDs []string) error {
	normalized, err := normalizeEnabledAgents(agentIDs)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return errors.New("at least one agent must be enabled")
	}
	return s.writeState(SetupState{
		EnabledAgents:       normalized,
		AgentSetupCompleted: true,
	})
}

func (s *SettingsService) IsAgentEnabled(agentID string) (bool, error) {
	if !isSupportedAgentID(agentID) {
		return false, fmt.Errorf("unknown agent: %s", agentID)
	}
	enabled, err := s.GetEnabledAgents()
	if err != nil {
		return false, err
	}
	for _, id := range enabled {
		if id == agentID {
			return true, nil
		}
	}
	return false, nil
}

func (s *SettingsService) defaultState() SetupState {
	return SetupState{
		EnabledAgents:       []string{},
		AgentSetupCompleted: false,
	}
}

func (s *SettingsService) writeState(state SetupState) error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0o644)
}

func normalizeEnabledAgents(agentIDs []string) ([]string, error) {
	seen := make(map[string]bool, len(agentIDs))
	for _, id := range agentIDs {
		if !isSupportedAgentID(id) {
			return nil, fmt.Errorf("unknown agent: %s", id)
		}
		seen[id] = true
	}

	supported := SupportedAgentIDs()
	normalized := make([]string, 0, len(seen))
	for _, id := range supported {
		if seen[id] {
			normalized = append(normalized, id)
		}
	}
	return normalized, nil
}

func isSupportedAgentID(agentID string) bool {
	for _, id := range SupportedAgentIDs() {
		if id == agentID {
			return true
		}
	}
	return false
}
