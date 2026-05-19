## Context

`InstalledPluginsView.vue` lists plugins per agent and currently exposes a single per-card action (**Uninstall**) wired via `InstalledPluginCard.vue`'s `uninstall` emit → `plugin.uninstall(agentID, id)` → `app.PluginApp.Uninstall` → `internal/plugin/Service.Uninstall` → `<agent> plugin uninstall <id>`.

The same code path already exists for a related operation: `Service.UpdateAgentMarketplace` runs `<agent> plugin marketplace update` via `runAgentCmdStreaming`, with stdout streamed line-by-line back to the frontend over the `plugin:install-log` Wails event. The install flow uses the same event.

Adding **Update** is therefore a vertical slice that re-uses existing primitives (binding wrapper, streaming runner, install-log event, toast composable, confirm modal) rather than introducing new infrastructure.

## Goals / Non-Goals

**Goals:**

- One-click in-app update of an installed plugin, per agent, using the agent CLI as the source of truth.
- Stream CLI output to the user using the same overlay/event the install flow already uses, so there is one consistent log surface.
- Keep the installed list authoritative — after an update succeeds, re-read the agent's `installed_plugins.json` / `config.json` rather than mutating local state.
- Cleanly disable the row's actions while a request is in flight so a user cannot fire overlapping `update`/`uninstall` calls against the same plugin.

**Non-Goals:**

- No version comparison or "update available" badge — we cannot determine whether an update is required without first running the CLI; this change ships an unconditional Update affordance only.
- No bulk update across plugins (out of scope; could be a follow-up that reuses `UpdatePlugin` per row).
- No new preference flags or settings UI changes.
- No new spec for `plugin-browser` or `marketplace-management` — those flows are unaffected.

## Decisions

### Decision 1: Reuse the existing `name@marketplace` install identifier

`InstalledPlugin.UninstallName` already carries the exact `name@marketplace` string the CLI accepts (`pluginName@marketplaceName`). The same string is what `<agent> plugin update` takes, so the card does not need a new field — `update` and `uninstall` share the identifier.

**Alternative considered:** add a distinct `UpdateName` field. Rejected — the two CLIs use the same identifier for both subcommands, so a second field would only duplicate data.

### Decision 2: Add a single per-plugin method, not a bulk variant

Expose `Service.UpdatePlugin(agentID, installName, logFn)` and bind it through Wails as `PluginApp.UpdatePlugin`. Bulk updates (if ever added) can iterate from the frontend, just like the install flow iterates `Install` for each selected plugin.

**Alternative considered:** `UpdatePlugins(agentID, installNames []string, logFn)`. Rejected — premature; only the per-card button is required, and a bulk call would add error-aggregation logic with no current consumer.

### Decision 3: Stream CLI output on the existing `plugin:install-log` event

Use `runAgentCmdStreaming` (already used for `UpdateAgentMarketplace`) and re-use the `plugin:install-log` event channel. The store already subscribes once at construction and pushes lines into `installLog`. No new event name is introduced.

**Trade-off:** The event name `plugin:install-log` is slightly misleading for an update — but renaming would be a breaking-rename that touches the marketplace-update flow too, and the channel is internal to the app. We accept the imprecise name in exchange for not multiplying event channels.

### Decision 4: Re-read the installed list after success rather than patching local state

After a successful update the card's metadata (e.g. version) may change. Instead of trying to derive the new metadata from CLI output, call `silentRefreshInstalled(agentID)` — the same helper the install flow uses — and let the view recompute.

**Alternative considered:** parse CLI stdout. Rejected — output format is agent-specific and undocumented; re-reading the agent's own config file is robust and matches the existing install pattern.

### Decision 5: Disable both row actions while an update is in flight, per plugin

Track a `Set<string>` of in-flight `uninstallName`s on the view (or in the store) and pass a `busy` boolean to the card. Both **Update** and **Uninstall** are disabled when the row is busy. This avoids races where the user clicks Update then Uninstall before the first call resolves.

**Alternative considered:** rely on the existing global `store.installing` flag. Rejected — that flag is used by the install flow's modal overlay and is too coarse; multiple cards could legitimately update in parallel, so the busy state must be per-plugin.

### Decision 6: UI placement and styling

Place **Update** immediately to the left of **Uninstall** in the existing `flex flex-row items-center justify-between gap-3` row. Use `AppButton` with `variant="ghost"` `size="md"` (same as **Uninstall**) so the two read as a coherent action group; if visual emphasis becomes necessary, the **Update** button can adopt `variant="outline"` in a follow-up without changing the data flow.

**Alternative considered:** put Update inside a dropdown menu on the card. Rejected — the codebase deliberately avoids the `dropdown` daisyUI primitive (CLAUDE.md "daisyUI classes the codebase does not currently use … SHALL NOT be introduced inline"), and Update is a first-class action that deserves a visible button.

### Decision 7: No confirmation modal for Update

Update is non-destructive and idempotent at the CLI level (re-running it when already current is a no-op or fast refresh). Skip the `useConfirm` step that Uninstall uses, to keep the action one click. Toast on success/failure is sufficient feedback.

## Risks / Trade-offs

- **[Risk] Agent CLI does not support `plugin update`** → If a future agent is added that lacks the subcommand, `runAgentCmdStreaming` will surface the CLI's stderr verbatim through the existing toast. Mitigation: extend `Service.UpdatePlugin` only after confirming the CLI surface for any new agent.
- **[Risk] Long-running updates leave the card visually unresponsive** → The row is disabled while busy but does not show a spinner. Mitigation: render an `AppSpinner` inside the **Update** button when the row is busy, matching how the install button surfaces work in `BrowsePluginsView`.
- **[Risk] Stdout streaming order vs. completion order** → Two updates fired in parallel will interleave log lines on the shared `plugin:install-log` channel. Mitigation: the CLI already prefixes its own output; if interleaving becomes confusing, the backend can prepend `[<agentID>/<installName>] ` like `UpdateAgentMarketplace` does. Out of scope for this change unless we find it cryptic in practice.
- **[Trade-off] Event name reuse (`plugin:install-log`)** → see Decision 3.
- **[Trade-off] No "update available" indicator** → see Non-Goals. Users update unconditionally; an idempotent update is cheap.
