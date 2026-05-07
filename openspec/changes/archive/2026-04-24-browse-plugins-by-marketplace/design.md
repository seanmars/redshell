## Context

The Wails + Vue 3 desktop shell already separates concerns into three pinia stores and three corresponding Go services:

- `useMarketplaceStore` ↔ `app.MarketplaceApp` ↔ `internal/marketplace.Service` (reads `~/.redshell/marketplace.json`)
- `usePluginStore` ↔ `app.PluginApp` ↔ `internal/plugin.Service` (aggregates plugins across marketplaces via `FetchAll`)
- `useProviderStore` ↔ `app.ProviderApp` ↔ `internal/provider.Service`

`PluginApp.FetchAll()` already returns `FetchAllResult{Plugins, Errors}`, where each `MarketplacePlugin` carries `Marketplace` (stable ID) and `MarketplaceName` (display name per provider). Errors are pre-formatted as `[marketplaceID/provider] message`.

`BrowsePluginsView.vue` currently:
- fetches only `usePluginStore.fetchAll()` and installed lists on mount,
- renders a flat grid filtered by a local `providerFilter` ref with tabs (All / claude / copilot),
- shows a single empty state keyed solely on `store.plugins.length === 0`, with copy that always points users at the Marketplaces page,
- dumps `store.fetchErrors` as a warning list at the bottom of the page.

The user-reported bug is that after adding a marketplace, the Browse Plugins screen still says "No plugins found. Add a marketplace." The empty-state copy is the proximate cause of confusion; any silent per-marketplace failure (404 `plugins/` dir, API rate-limit, token missing) produces the same message because the view never distinguishes "registry empty" from "registry populated but fetch yielded nothing".

## Goals / Non-Goals

**Goals:**
- Make Browse Plugins reflect registered marketplaces independently of whether plugin fetches succeed.
- Group plugin cards under a section header per marketplace so users can tell which source each plugin came from and which marketplace is failing.
- Remove the provider filter tabs; show all plugins regardless of provider.
- Surface per-marketplace fetch errors on the offending section header, not at the bottom.
- Keep the change frontend-only: no backend API or persisted-data changes.

**Non-Goals:**
- Redesigning the install flow, provider selection in the install modal, or the install log UI.
- Adding per-marketplace refresh, reorder, or collapse controls (future work).
- Changing how installed-state is looked up or how the plugin card itself renders.
- Changing backend `FetchAll` to return a grouped structure; grouping happens client-side.

## Decisions

### Decision 1: Group client-side off `MarketplacePlugin.marketplace`
**Choice:** Build a `pluginsByMarketplace` getter in `usePluginStore` that groups the existing flat `plugins` array by the `marketplace` (ID) field, using the marketplace list from `useMarketplaceStore` as the canonical ordering and display-name source.

**Why:** `FetchAllResult` already contains every field needed for grouping. Moving the transformation to the store keeps the view declarative and avoids a backend contract change. Using `useMarketplaceStore` as the source of truth for the section list means a just-added marketplace appears as an (empty or loading) section even if its fetch fails.

**Alternatives considered:**
- *Return `map[marketplaceID][]MarketplacePlugin` from backend.* Rejected — larger API-surface change; the Wails-generated model churn isn't worth it for a view-level concern.
- *Group in the view template only.* Rejected — view would still need marketplace metadata; a derived store getter is easier to test and reuse.

### Decision 2: Section status = union of plugins, errors, and marketplace presence
**Choice:** Each section computes its status from three inputs:

1. `marketplace` entry (always present from the registry, provides `id`, `url`, `name`).
2. `plugins` filtered to that marketplace ID (may be empty).
3. errors from `FetchAllResult.Errors` that start with `[<marketplaceID>/`.

Rendering rules:
- loading (global `store.loading` true) → show skeleton for every section
- plugins present → render grid of `PluginCard`s
- plugins empty + errors present → render error message (one per provider that errored)
- plugins empty + no errors → render "No plugins available in this marketplace"

**Why:** Directly answers the reported bug: an added marketplace is visible even with no plugins, and the reason (error vs genuinely empty) is discoverable.

### Decision 3: Remove the provider filter tabs entirely, not just hide them
**Choice:** Delete the `providerFilter` ref, the tab markup, and the `providers` array from `BrowsePluginsView.vue`. `PluginCard` already displays the provider per plugin.

**Why:** The proposal removes this capability from the spec. Keeping dead markup behind a flag invites drift. Provider selection still happens in the install modal, so users aren't losing the ability to target a provider at install time.

**Alternative considered:** Keep the tabs as an optional filter. Rejected — user explicitly asked to drop the split; tabs over a grouped-by-marketplace list would compound visual complexity.

### Decision 4: Keep the existing page-level "no marketplaces" empty state, reworded
**Choice:** When `marketplaceStore.marketplaces.length === 0`, render a single page-level empty state with a CTA linking to `/marketplaces`. This replaces the plugin-centric "No plugins found" message.

**Why:** Distinguishing "registry empty" from "registry populated but plugins empty" is the core fix. Two empty states (page-level vs section-level) naturally expresses this.

### Decision 5: Fetch marketplaces on mount, alongside plugins
**Choice:** `onMounted` in `BrowsePluginsView.vue` calls `marketplaceStore.fetchList()`, `pluginStore.fetchAll()`, and the two `fetchInstalled` calls in parallel. No ordering dependency in the store (plugin backend re-reads the registry on each call).

**Why:** Browse Plugins must render marketplace sections before (or independently of) plugin fetch completing. Parallel fire-and-forget is cheapest; the loading flag already handles the in-flight case.

## Risks / Trade-offs

- **Risk:** A marketplace registered in `~/.redshell/marketplace.json` but not actually reachable will now render a permanent empty/error section the user cannot clear except via the Marketplaces page. → **Mitigation:** Error text cites the marketplace ID, and the section header can link to Marketplaces for removal. Acceptable because it is strictly more informative than today's silent drop.
- **Risk:** Grouping view loses the at-a-glance "all plugins" feel when many marketplaces are registered. → **Mitigation:** Section headers are compact; each section still uses the same `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3` grid. If density becomes a problem, future work can add collapse/expand controls.
- **Trade-off:** The `fetchErrors` list at the bottom of the page is removed. Users looking only at the bottom will no longer see global errors. → **Mitigation:** Errors now appear on the section that produced them, which is closer to the cause.
- **Risk:** Frontend grouping diverges from backend if marketplace IDs in plugins do not exactly match registry IDs (e.g., trailing `.git`, case differences). → **Mitigation:** Plugin service already uses `m.ID` both to tag plugins and to name errors, so they round-trip identically; no normalization needed.

## Migration Plan

No data migration. No flags. Ship the view and store changes together. Because the change is purely UI-layer and the backend API is unchanged, rollback is a single-commit revert.
