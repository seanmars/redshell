## Context

The Marketplaces tab today supports `Add` and `Remove`. RedShell's existing `Refresh` action runs `git fetch --depth=1` + `git reset --hard FETCH_HEAD` against each cached clone under `~/.redshell/.cache/`. That keeps RedShell's plugin browser in sync with upstream manifests but does **not** touch the per-agent registries (`~/.claude/plugins/known_marketplaces.json`, `~/.copilot/config.json`). When an agent's CLI caches a stale plugin list, the user has to drop to a terminal.

The pattern for shelling out to agent CLIs is already in `internal/plugin/service.go`:
- `runAgentCmd(agentID, args)` runs `<agent> <args...>` with hidden console window (Windows) and folds stderr into a wrapped error.
- `Install` streams progress through a `logFn` callback that the Wails wrapper (`app/plugin.go`) bridges to the `plugin:install-log` Wails runtime event.
- Enabled-agent enumeration goes through `Service.enabledAgents()` (which respects `agent.SettingsService.GetEnabledAgents()`).

The frontend store side (`stores/marketplace.ts`) is a thin proxy with `fetchList`, `add`, `remove`. The view (`components/settings/MarketplacesTab.vue`) is roughly 130 lines today and already imports `AppButton`, `AppIcon`, `useConfirm`.

## Goals / Non-Goals

**Goals:**
- A single button on the Marketplaces tab updates the agent-side marketplace registry for every enabled agent.
- The action is observable: each agent's CLI output is streamed live to the user, not silently swallowed.
- Per-agent failures are surfaced individually; one failing agent does not abort the rest.
- Backend logic is unit-testable without Wails by keeping it in `internal/plugin/service.go`, mirroring how `Install` is structured.

**Non-Goals:**
- Updating individual marketplaces. The agent CLIs only expose `plugin marketplace update` (update-all), so per-row buttons would not map to a clean CLI command and add UI without value.
- Touching RedShell's own clone cache (`~/.redshell/.cache/`). That is the existing `Refresh`/`RefreshAll` job and stays untouched.
- Computing or showing a structured diff of which plugins changed. The CLI's stdout is enough for v1.
- Auto-running this on a schedule. User-triggered only.

## Decisions

### Decision 1: Place the new method on `plugin.Service`, not `marketplace.Service`

The new method shells out to agent CLIs (`claude`, `copilot`) and depends on `enabledAgents()`. `plugin.Service` already owns both that dependency and the `runAgentCmd` helper. `marketplace.Service` deliberately knows nothing about agents — its only agent awareness is the `AgentMarketplaceFiles` lookup table. Putting agent-CLI orchestration there would force a circular dependency or a duplicated runner.

**Alternative considered:** add it to `marketplace.Service` so the Marketplaces tab calls a single store. Rejected — colocation by UI is the wrong cut. The frontend can mix two stores in one view (it already does in BrowsePluginsView).

### Decision 2: Per-agent primitive plus a fan-out convenience

Backend exposes both a per-agent primitive and a fan-out wrapper:

```go
type AgentUpdateOutcome struct {
    AgentID string `json:"agentId"`
    OK      bool   `json:"ok"`
    Error   string `json:"error,omitempty"`
}

type UpdateAgentMarketplacesResult struct {
    Outcomes []AgentUpdateOutcome `json:"outcomes"`
}

func (s *Service) UpdateAgentMarketplace(agentID string, logFn func(string)) AgentUpdateOutcome
func (s *Service) UpdateAgentMarketplaces(logFn func(string)) UpdateAgentMarketplacesResult // loops over enabledAgents()
```

The frontend uses the per-agent primitive (`UpdateAgentMarketplace`) and fires the per-agent calls concurrently via `Promise.all`, with one sticky `info` "Updating <agentId>..." toast per agent dismissed individually as each call resolves. This is a deliberate revision after the initial single-fan-out design: a single round-trip hides which agent is currently running, so the user sees only the spinner. Per-agent calls give the frontend per-agent UI feedback for free via the existing await semantics, and `Promise.all` keeps wall-clock time at `max(claude, copilot)` instead of `sum(...)`. Wails dispatches each JS-initiated method into its own goroutine, so the parallel calls genuinely run concurrently on the backend; the two agents touch disjoint config dirs and disjoint binaries, so there is no shared-resource contention. Live `[claude] ...` and `[copilot] ...` log lines may interleave on the shared `plugin:install-log` event — that is the right behavior for a multi-process stream and the existing prefix already disambiguates them. The fan-out variant `UpdateAgentMarketplaces` stays for batch / non-UI callers and unit tests; it loops over the same per-agent primitive sequentially in Go.

**Alternative considered (rejected):** one Wails method per *named* agent (`UpdateClaudeMarketplaces`, `UpdateCopilotMarketplaces`). Forces a TS switch on agent ID and adds a method every time we onboard a new agent.

**Alternative considered (rejected):** keep a single fan-out and emit a structured `marketplace:update-progress` event per agent. Heavier — requires Vue lifecycle subscription/teardown logic in the component. Per-agent calls give the frontend sequential progress for free via the existing await semantics.

### Decision 3: Reuse the `plugin:install-log` event channel

The frontend already has `EventsOn('plugin:install-log', ...)` wired in `stores/plugin.ts`. The update logs are short-lived (one CLI invocation per agent), so introducing a separate event topic just for them is overhead. We prefix lines (`[claude] ...`, `[copilot] ...`) so the existing log viewer stays readable.

**Alternative considered:** a new `marketplace:update-log` topic. Rejected as premature — if log noise becomes an issue we can split later without breaking callers (it's an emit, not a contract).

### Decision 4: Frontend disables the button while in flight, no per-agent UI

A single boolean `updating` on `useMarketplaceStore` gates the button. When the call resolves, we render a single `useToast` notification: success if all outcomes are OK, otherwise an error toast that lists which agents failed and a separate one per success. We do **not** add a modal log viewer for v1 — the existing install-log container in BrowsePluginsView is the closest precedent if we need it later.

### Decision 5: `runAgentCmd` needs to forward stdout to the log callback

Today `runAgentCmd` only captures stderr. For `Install`, stdout is currently dropped, and that is fine because the calling code emits its own status lines (`Installing: %s`, `Installed: %s`). For `marketplace update`, the *only* useful output is from the CLI itself — there is no per-marketplace work for RedShell to narrate. So the implementation will:

1. Add an optional `stdoutFn func(string)` parameter to `runAgentCmd` (or a sibling `runAgentCmdStreaming`), wiring `cmd.Stdout` to a line-splitting writer that calls back per line.
2. Keep stderr captured into a buffer for the wrapped error message (no behavior change for `Install` / `Uninstall`).
3. The `UpdateAgentMarketplaces` method calls the streaming variant so each line of `claude plugin marketplace update` reaches the frontend live.

**Alternative considered:** capture stdout into a buffer and emit it at the end. Rejected — defeats the purpose of progress streaming for an operation that may take seconds per agent.

### Decision 6: Error format follows the existing `[<scope>] <msg>` convention

The frontend's `errorsByMarketplace` parser in `stores/plugin.ts` expects errors prefixed `[<marketplaceID>/<agent>] <msg>`. For this action there is no marketplace context (it is fan-out per agent), so the analogous format is `[<agentID>] <msg>`. The new store action doesn't need to feed `errorsByMarketplace`; it surfaces failures via toasts directly.

## Risks / Trade-offs

- [Risk] **`<agent> plugin marketplace update` is a long-running command (network + git fetch per registered marketplace)** → Mitigation: stream stdout live, disable the button, no spinner timeout. Worst case the user navigates away — the command continues in the backend; we don't kill it.
- [Risk] **Agent CLI not installed** → Mitigation: `runAgentCmd` already returns a friendly `"agent CLI '%s' is not installed"` error. We surface it in the per-agent outcome and skip remaining agents only if user has none enabled.
- [Risk] **CLI output formatting differs across agent versions** → Mitigation: we just pipe lines through, prefixed with `[<agent>]`. We don't parse — no parser, no breakage.
- [Trade-off] **Stdout streaming via line-splitting writer adds ~20 lines of plumbing to `runAgentCmd`** → Acceptable; the current `bytes.Buffer` capture stays for stderr, and `Install` keeps working unchanged.
- [Trade-off] **No partial-update UI (per-marketplace)** → Documented non-goal; revisit if/when a CLI exposes per-marketplace update.

## Migration Plan

Purely additive. No data migration, no config migration. Rollback = revert the commit; the existing `Refresh` action keeps working throughout.

## Open Questions

- Should the button be a primary or secondary variant? Leaning secondary (it is a maintenance action, not the primary CTA) — confirm during implementation against the design system.
- Toast wording for the all-success case: "Updated 2 agents" vs. "Marketplaces updated"? Will pick during implementation; not a contract decision.
