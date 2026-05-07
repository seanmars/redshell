## 1. Backend (Go) — plugin service

- [x] 1.1 In `internal/plugin/service.go`, add a streaming variant of `runAgentCmd` (or extend it with an optional `stdoutFn func(string)` parameter) that wires `cmd.Stdout` to a line-splitting writer; keep stderr capture unchanged so `Install` / `Uninstall` behaviour is identical
- [x] 1.2 Add result types `AgentUpdateOutcome { AgentID, OK, Error }` and `UpdateAgentMarketplacesResult { Outcomes []AgentUpdateOutcome }` next to the existing `FetchAllResult`
- [x] 1.3 Add `Service.UpdateAgentMarketplaces(logFn func(string)) UpdateAgentMarketplacesResult` that iterates `s.enabledAgents()`, runs `<agent> plugin marketplace update` via the streaming runner with `logFn` prefixed `[<agentID>] `, and never aborts the loop on a per-agent error
- [x] 1.4 Format per-agent failure messages as `[<agentID>] <stderr or wrapped error>` and put the same message into `Outcome.Error`; `OK` mirrors `err == nil`
- [x] 1.5 Add unit tests in `internal/plugin/service_test.go` covering: all-success fan-out, one agent failing while another succeeds (use a stubbed runner injection point — extend `NewServiceWithCacheRoot`-style helper if needed), and the disabled-agent skip path

## 2. Backend (Go) — Wails wrapper

- [x] 2.1 In `app/plugin.go`, add `PluginApp.UpdateAgentMarketplaces() plugin.UpdateAgentMarketplacesResult` that calls `a.svc.UpdateAgentMarketplaces` with a `logFn` that emits `runtime.EventsEmit(a.ctx, "plugin:install-log", msg)` (matching the existing `Install` pattern)
- [x] 2.2 Run `wails generate module` (or `wails dev` once) to regenerate `frontend/wailsjs/go/app/PluginApp.{ts,js}` and `frontend/wailsjs/go/models.ts`; verify the new method and result type appear; do not hand-edit
- [x] 2.3 Run `go fmt ./internal/plugin ./app` and `go vet ./...` to confirm the backend changes are clean

## 3. Frontend — store action

- [x] 3.1 In `frontend/src/stores/marketplace.ts`, import `UpdateAgentMarketplaces` from `@wailsjs/go/app/PluginApp` and the result type from `@wailsjs/go/models`
- [x] 3.2 Add reactive `updating` flag and an `updateAll()` action that flips `updating` true/false around the call and returns the per-agent outcome list to the caller
- [x] 3.3 Subscribe to `plugin:install-log` if the marketplace store does not already (it currently lives only in `stores/plugin.ts`); decide between (a) reusing the plugin store's log buffer and (b) keeping a local one for marketplace-update flows — document the choice in the PR description
  - **Decision:** neither store subscribes specifically for the marketplace-update flow. The per-agent `error` string returned in `outcomes` already carries the actionable detail (delivered via toast). The existing `EventsOn('plugin:install-log', ...)` in `stores/plugin.ts` keeps live CLI lines flowing into `installLog` for any view that already renders them; no additional subscription needed in the marketplace store.

## 4. Frontend — Marketplaces tab UI

- [x] 4.1 In `frontend/src/components/settings/MarketplacesTab.vue`, add an "Update" `AppButton` to the left of the existing "Add Marketplace" button (use `AppIcon name="refresh"` or the closest existing icon; do not introduce a new daisyUI class outside the `AppButton` primitive)
- [x] 4.2 Bind the button's `loading` and `disabled` props to `store.updating`; on click call `store.updateAll()` and surface results via `useToast()`: one success toast on all-OK, otherwise one error toast per failing agent including the error message
- [x] 4.3 Confirm no daisyUI leak by running the documented mechanical leak check from CLAUDE.md across the changed files

## 5. Verification

- [x] 5.1 Run `go test ./internal/plugin -run UpdateAgentMarketplaces` and `go test ./...` — all pass
- [x] 5.2 Run `pnpm format`, `pnpm lint`, `pnpm type-check`, and `pnpm run test:unit` from `frontend/` — all pass
- [ ] 5.3 Manual smoke test via `wails dev`: with both agents enabled, click Update with a healthy marketplace registered (expect success toast and live `[claude] ...` / `[copilot] ...` log lines); then disable one agent and confirm it is skipped; finally simulate a failure (rename `claude` on `PATH` temporarily) and confirm the failure toast and that `copilot` still succeeds
- [ ] 5.4 Verify `~/.redshell/.cache/` and `~/.redshell/marketplace.json` are unchanged after Update runs (RedShell-side state is not touched)
