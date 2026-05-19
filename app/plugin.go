package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"redshell/internal/plugin"
)

type PluginApp struct {
	svc *plugin.Service
	ctx context.Context
}

func NewPluginApp(svc *plugin.Service) *PluginApp {
	return &PluginApp{svc: svc}
}

func (a *PluginApp) SetContext(ctx context.Context) {
	a.ctx = ctx
}

func (a *PluginApp) FetchAll() plugin.FetchAllResult {
	return a.svc.FetchAll()
}

func (a *PluginApp) Install(prov string, plugins []plugin.MarketplacePlugin) error {
	return a.svc.Install(prov, plugins, func(msg string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "plugin:install-log", msg)
		}
	})
}

func (a *PluginApp) ListInstalled(prov string) ([]plugin.InstalledPlugin, error) {
	return a.svc.ListInstalled(prov)
}

func (a *PluginApp) Uninstall(prov, pluginID string) error {
	return a.svc.Uninstall(prov, pluginID)
}

func (a *PluginApp) UpdatePlugin(prov, installName string) error {
	return a.svc.UpdatePlugin(prov, installName, func(msg string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "plugin:install-log", msg)
		}
	})
}

func (a *PluginApp) UpdateAgentMarketplaces() plugin.UpdateAgentMarketplacesResult {
	return a.svc.UpdateAgentMarketplaces(func(msg string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "plugin:install-log", msg)
		}
	})
}

func (a *PluginApp) UpdateAgentMarketplace(agentID string) plugin.AgentUpdateOutcome {
	return a.svc.UpdateAgentMarketplace(agentID, func(msg string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "plugin:install-log", msg)
		}
	})
}
