package main

import (
	"context"
	"embed"
	"encoding/json"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

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

// singleInstanceUniqueID namespaces the production single-instance lock. It
// MUST stay stable across releases so two different installed versions cannot
// run at once (Wails derives the OS mutex name from it).
const singleInstanceUniqueID = "com.seanmars.redshell"

// waitParentTimeout bounds how long an updater-relaunched binary waits for the
// outgoing process to exit before proceeding to acquire the lock.
const waitParentTimeout = 10 * time.Second

// appCtx is the Wails runtime context captured in OnStartup so the
// single-instance second-launch callback can raise the existing window.
var appCtx context.Context

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
	// Updater relaunch handshake: when the auto-updater spawns this binary it
	// passes --wait-parent-pid=<oldpid>. Wait for that process to exit (and
	// release the single-instance lock) before continuing into wails.Run,
	// otherwise this fresh instance would detect the still-alive old one and
	// terminate itself, leaving zero instances running.
	if pid, ok := parseWaitParentPID(os.Args[1:]); ok {
		waitForParentExit(pid, waitParentTimeout)
	}

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
		SingleInstanceLock: newSingleInstanceLock(func(options.SecondInstanceData) {
			if appCtx == nil {
				return
			}
			// Cover both hidden-to-tray (WindowShow) and minimized-to-taskbar
			// (WindowUnminimise -> Restore) states; WindowShow ends with
			// SetForegroundWindow + SetFocus on Windows so the window comes
			// to the front.
			runtime.WindowUnminimise(appCtx)
			runtime.WindowShow(appCtx)
		}),
		OnStartup: func(ctx context.Context) {
			appCtx = ctx
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
