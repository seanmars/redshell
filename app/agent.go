package app

import (
	"context"

	"redshell/internal/agent"
)

type AgentApp struct {
	svc         *agent.Service
	settingsSvc *agent.SettingsService
	ctx         context.Context
}

func NewAgentApp(svc *agent.Service, settingsSvc *agent.SettingsService) *AgentApp {
	return &AgentApp{svc: svc, settingsSvc: settingsSvc}
}

func (a *AgentApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *AgentApp) ListAgents() []agent.Agent {
	return a.svc.ListAgents()
}

func (a *AgentApp) GetAgentSetupState() (agent.SetupState, error) {
	return a.settingsSvc.GetSetupState()
}

func (a *AgentApp) GetEnabledAgents() ([]string, error) {
	return a.settingsSvc.GetEnabledAgents()
}

func (a *AgentApp) SetEnabledAgents(agentIDs []string) error {
	return a.settingsSvc.SetEnabledAgents(agentIDs)
}

func (a *AgentApp) IsAgentEnabled(agentID string) (bool, error) {
	return a.settingsSvc.IsAgentEnabled(agentID)
}
