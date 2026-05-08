package main

import (
	"context"
	"embed"
	"encoding/json"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"redshell/app"
	"redshell/internal/agent"
	"redshell/internal/hooks"
	"redshell/internal/marketplace"
	"redshell/internal/plugin"
	"redshell/internal/preferences"
	"redshell/internal/sessionhistory"
	"redshell/internal/tray"
	"redshell/internal/updater"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsConfig []byte

type Config struct {
	Info struct {
		ProductVersion string `json:"productVersion"`
	} `json:"info"`
}

func GetAppVersion() string {
	var c Config
	_ = json.Unmarshal(wailsConfig, &c)
	return c.Info.ProductVersion
}

func main() {
	agentSvc := agent.NewService()
	agentSettingsSvc := agent.NewSettingsService()
	marketplaceSvc := marketplace.NewService()
	pluginSvc := plugin.NewService(marketplaceSvc, agentSvc, agentSettingsSvc)
	sessionHistorySvc, sessionHistoryErr := sessionhistory.NewService()
	if sessionHistoryErr != nil {
		println("session history service init failed:", sessionHistoryErr.Error())
	}
	hooksSvc, hooksErr := hooks.NewService()
	if hooksErr != nil {
		println("hooks service init failed:", hooksErr.Error())
	}
	preferencesSvc := preferences.NewService()
	trayMgr := tray.NewManager()

	exePath, exePathErr := os.Executable()
	if exePathErr != nil {
		println("updater: resolve exe path failed:", exePathErr.Error())
	}
	updaterSvc, updaterErr := updater.NewService(preferencesSvc, GetAppVersion(), exePath, updater.Options{})
	if updaterErr != nil {
		println("updater service init failed:", updaterErr.Error())
	}

	agentApp := app.NewAgentApp(agentSvc, agentSettingsSvc)
	marketplaceApp := app.NewMarketplaceApp(marketplaceSvc)
	pluginApp := app.NewPluginApp(pluginSvc)
	systemApp := app.NewSystemApp(GetAppVersion())
	sessionHistoryApp := app.NewSessionHistoryApp(sessionHistorySvc)
	hooksApp := app.NewHooksApp(hooksSvc)
	preferencesApp := app.NewAppPreferencesApp(preferencesSvc, trayMgr)
	var updaterApp *app.UpdaterApp
	if updaterSvc != nil {
		updaterApp = app.NewUpdaterApp(updaterSvc)
	}

	err := wails.Run(&options.App{
		Title:     "RedShell",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			agentApp.Startup(ctx)
			pluginApp.SetContext(ctx)
			systemApp.Startup(ctx)
			preferencesApp.Startup(ctx)
			if updaterApp != nil {
				updaterApp.Startup(ctx)
			}
			if trayMgr.Available() {
				if updaterApp != nil && updaterApp.AutoUpdateAvailable() {
					trayMgr.SetCheckForUpdates(updaterApp.HandleTrayOpen)
				}
				if err := trayMgr.Start(ctx, preferencesSvc); err != nil {
					println("tray manager start failed:", err.Error())
				}
			}
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if updaterApp != nil && updaterApp.InProgress() {
				return false
			}
			return preferencesApp.HandleBeforeClose(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			_ = ctx
			trayMgr.Stop()
			if updaterSvc != nil {
				updaterSvc.Stop()
			}
		},
		Bind: func() []interface{} {
			binds := []interface{}{
				agentApp,
				marketplaceApp,
				pluginApp,
				systemApp,
				sessionHistoryApp,
				hooksApp,
				preferencesApp,
			}
			if updaterApp != nil {
				binds = append(binds, updaterApp)
			}
			return binds
		}(),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
