## 1. Backend service

- [x] 1.1 In `internal/plugin/service.go`, add `UpdatePlugin(agentID, installName string, logFn func(string)) error` that calls `s.ensureAgentEnabled(agentID)` and then `runAgentCmdStreaming(agentID, []string{"plugin", "update", installName}, stream)` where `stream` prefixes lines with `[<agentID>] ` to match the pattern in `UpdateAgentMarketplace`.
- [x] 1.2 Return early with a wrapped error if `installName` is empty (defensive guard — never trigger `<agent> plugin update` with no argument).
- [x] 1.3 Run `go fmt ./internal/plugin/...` and `go vet ./internal/plugin/...`.

## 2. Backend Wails binding

- [x] 2.1 In `app/plugin.go`, add `(a *PluginApp) UpdatePlugin(prov, installName string) error` that calls `a.svc.UpdatePlugin(prov, installName, ...)` and emits each log line on `plugin:install-log` via `runtime.EventsEmit` (same closure pattern used by `Install` and `UpdateAgentMarketplace`).
- [x] 2.2 Re-run `wails dev` (or `wails generate module`) so `frontend/wailsjs/go/app/PluginApp.{js,d.ts}` and `frontend/wailsjs/go/models.ts` regenerate; commit the regenerated files.
- [x] 2.3 Confirm `main.go` does not need changes (the existing `app.NewPluginApp(...)` already binds the entire struct).

## 3. Backend tests

- [x] 3.1 In `internal/plugin/service_test.go`, add `TestUpdatePlugin_Success` that stubs `runAgentCmdStreaming` to capture args, asserts `args == ["plugin", "update", "demo@my-mkt"]`, and verifies `logFn` receives the streamed line with the `[claude] ` prefix.
- [x] 3.2 Add `TestUpdatePlugin_AgentDisabled` that constructs a service with a fake `settingsSvc` returning `enabled=false` for the requested agent and asserts the error matches `agent is disabled: <id>` and the streaming runner is not invoked.
- [x] 3.3 Add `TestUpdatePlugin_EmptyInstallName` asserting an early error and no CLI invocation.
- [x] 3.4 Run `go test ./internal/plugin/...`.

## 4. Frontend store

- [x] 4.1 In `frontend/src/stores/plugin.ts`, add `update(agentID: string, installName: string)` that awaits `UpdatePlugin(agentID, installName)` from `@wailsjs/go/app/PluginApp`, then calls `silentRefreshInstalled(agentID)` to refresh metadata.
- [x] 4.2 Track in-flight updates with a new reactive `Set<string>` keyed by `installName`; expose `isPluginBusy(installName)` and use it to drive per-row disabled state.
- [x] 4.3 Export the new action and busy helper from the store's return object.
- [x] 4.4 In `frontend/src/stores/__tests__/plugin.test.ts`, mock `UpdatePlugin` from `../../wailsjs/go/app/PluginApp` and add a test verifying `update()` invokes it with the right args, populates and clears the busy set, and re-reads `ListInstalled` on success.
- [x] 4.5 Add a failure-path test asserting the busy set is cleared even when `UpdatePlugin` rejects.

## 5. Frontend card component

- [x] 5.1 In `frontend/src/components/plugin/InstalledPluginCard.vue`, declare a new prop `busy?: boolean` (default `false`).
- [x] 5.2 Add an emit `update: [id: string]` alongside the existing `uninstall` emit.
- [x] 5.3 Insert an **Update** `AppButton` immediately to the left of the **Uninstall** button using `variant="ghost"` `size="md"`; bind its `:disabled` to `busy` and its `@click` to `emit('update', plugin.uninstallName)`. (Also wired `:loading="busy"` so the button shows the existing AppButton spinner during a long-running CLI call.)
- [x] 5.4 Bind the **Uninstall** button's `:disabled` to `busy` as well so the row's actions move together.
- [x] 5.5 Verify no inline daisyUI primitive classes are introduced (must continue to pass the leak-check `rg` from `CLAUDE.md`).

## 6. Frontend view wiring

- [x] 6.1 In `frontend/src/views/InstalledPluginsView.vue`, import the new `update` action and busy helper from the plugin store. (Accessed via the existing `store` proxy — no new imports needed.)
- [x] 6.2 Add an `async function handleUpdate(installName: string)` that calls `store.update(activeAgent.value, installName)` and surfaces a success or error toast via `useToast()` referencing `installName`.
- [x] 6.3 Pass `:busy="store.isPluginBusy(p.uninstallName)"` and `@update="handleUpdate"` to `InstalledPluginCard`.
- [x] 6.4 Manually verify in `wails dev`: clicking **Update** on a Claude card runs `claude plugin update name@mkt`, streams stdout into the install-log overlay, ends with a success toast, and leaves the **Uninstall** button enabled afterward.
- [x] 6.5 Repeat the manual check for a Copilot card.

## 7. Lint, types, and final checks

- [x] 7.1 Run `pnpm format` (Prettier) on changed frontend files.
- [x] 7.2 Run `pnpm lint` (oxlint + eslint).
- [x] 7.3 Run `pnpm type-check` (vue-tsc) — must pass with the new prop and emit typed.
- [x] 7.4 Run `pnpm test:unit` for the plugin store tests.
- [x] 7.5 Run `go test ./...` once more from the repo root.
- [x] 7.6 Run the daisyUI leak-check `rg` command from `CLAUDE.md` against `frontend/src/views`, `frontend/src/components/plugin`, etc., and confirm zero matches.

## 8. Documentation and review

- [x] 8.1 Sanity-read `openspec/specs/installed-plugins-view/spec.md` after the change is archived to confirm the Update requirement merged cleanly.
- [x] 8.2 In the PR description, link `openspec/changes/add-plugin-update-button/` and call out: new binding method, new card button, no preference / settings changes.
