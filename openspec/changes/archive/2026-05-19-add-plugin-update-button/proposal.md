## Why

Users can install and uninstall plugins from the Installed Plugins page, but the only way to pull a newer plugin version today is to uninstall and reinstall it. Both supported agent CLIs (`claude` and `copilot`) expose a `plugin update <name>@<marketplace>` subcommand, so the shell can offer a one-click Update action that keeps the user inside the app and surfaces the CLI output through the existing install-log channel.

## What Changes

- Add a backend `plugin.Service.UpdatePlugin(agentID, installName, logFn)` method that runs `<agent> plugin update <installName>` via the same streaming helper used by marketplace updates.
- Expose it through Wails as `PluginApp.UpdatePlugin(prov, installName)` and forward log lines on the existing `plugin:install-log` event.
- Add an `update()` action on `usePluginStore` that calls the binding, awaits completion, and re-reads the installed list for the affected agent.
- Add an **Update** button to `InstalledPluginCard.vue`, placed immediately to the left of the existing **Uninstall** button; clicking it emits an `update` event consumed by `InstalledPluginsView.vue`.
- `InstalledPluginsView.vue` handles the new event with a per-card busy state, surfaces success/failure through `useToast()`, and reuses the install-log overlay if one is currently visible.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `installed-plugins-view`: adds an Update requirement alongside the existing Uninstall requirement so users can refresh an installed plugin in place.

## Impact

- **Backend Go**: `internal/plugin/service.go` (new method, reuses `runAgentCmdStreaming`), `app/plugin.go` (new binding + event emit), regenerated `frontend/wailsjs/go/app/PluginApp.*` bindings.
- **Frontend**: `frontend/src/stores/plugin.ts` (new `update` action), `frontend/src/components/plugin/InstalledPluginCard.vue` (new button + emit), `frontend/src/views/InstalledPluginsView.vue` (handler + toast + busy state).
- **Specs**: delta against `installed-plugins-view`.
- **External dependencies**: relies on the `claude` and `copilot` CLIs already exposing `plugin update`; no new dependency, no new files on disk, no preferences migration.
- **Tests**: extend `internal/plugin` unit tests with an `UpdatePlugin` case (via the existing fake-command pattern); frontend store test gains an `update` action case.
