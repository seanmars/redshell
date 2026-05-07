## Context

The current shell has four peer-level sidebar items: Providers, Marketplaces, Browse, Installed. The first two are configuration surfaces the user visits rarely (during setup, when adding/rotating tokens, or when registering a new marketplace). The last two are the day-to-day flow. Keeping all four at the same visual weight obscures the primary intent of the app.

The existing views (`ProvidersView.vue`, `MarketplacesView.vue`) already render against Pinia stores (`stores/provider.ts`, `stores/marketplace.ts`) that wrap Wails bindings; the views themselves contain no unique state logic that would complicate relocation — they orchestrate components like `ProviderCard.vue` and `MarketplaceCard.vue`.

`DefaultLayout.vue` already has a footer element conditionally rendered at the bottom of the main content column, but it is empty and is *not* a sidebar footer. We need a new element inside the `<aside>` element, below `<nav>`, that always renders (unlike the main content footer which supports `hideFooter`).

## Goals / Non-Goals

**Goals:**
- Consolidate Providers + Marketplaces into a single Settings page with a tabbed UI.
- Expose Settings via a discoverable icon button at the bottom of the left sidebar, not via a top-level nav item.
- Preserve deep-linking so existing bookmarks or documentation referencing tab content still resolve.
- Keep the change frontend-only — no Go, Wails bindings, or Pinia store changes.

**Non-Goals:**
- Redesigning the Providers or Marketplaces content itself. Cards, confirmations, store actions, and error surfaces remain unchanged.
- Introducing a generic tabs framework. daisyUI's `tabs tabs-bordered` class (already used elsewhere) is sufficient; no shared `Tabs.vue` primitive is added as part of this change.
- Adding new settings beyond relocating existing ones (e.g. no theme/shortcut settings added here — that is a future change if desired).
- Animations or transitions between tabs.

## Decisions

### Decision 1: Route structure — one `/settings` route with a `?tab=` query param vs. nested `/settings/providers` routes

**Chosen:** Single `/settings` route, tab selection via `?tab=providers|marketplaces` query param, defaulting to `providers` when absent.

**Rationale:** A query param keeps the route table flat and avoids needing a `<RouterView>` child. Both tabs mount cheaply — they each render a list from an already-hydrated Pinia store — so there is no lazy-loading benefit to making them separate routes. The `v-model` of the tab component reads and writes `route.query.tab`, which makes deep links and back-button history work naturally.

**Alternative considered:** Nested routes `/settings/providers` and `/settings/marketplaces` with a shared layout. Rejected because it adds a `<RouterView>` nesting level and two route entries without enabling anything the query param cannot.

### Decision 2: Backward-compatible redirects for `/providers` and `/marketplaces`

**Chosen:** Convert the old routes to redirect records that point at `/settings?tab=<name>`, rather than deleting them outright.

**Rationale:** There may be documentation, screenshots, or user muscle memory ("type /providers"). A 1-line redirect record is cheap insurance and keeps the app behavior continuous.

**Alternative considered:** Hard-delete the routes (404 on old URLs). Rejected — the cost of keeping redirects is trivial and they make the transition seamless.

### Decision 3: Sidebar footer button presentation — icon-only vs. icon + label

**Chosen:** Icon-only circular button (`btn btn-ghost btn-circle`), with a tooltip ("Settings") for discoverability.

**Rationale:** The sidebar width is fixed at `w-56`. A full-width `Settings` row at the bottom would visually compete with the primary nav list above it and re-introduce the top-level navigation item we are explicitly removing. An icon in a compact footer bar reads as utility/meta-navigation, which is the correct semantic weight.

**Alternative considered:** A full-width row styled the same as the nav items. Rejected — it defeats the purpose of demoting Providers + Marketplaces if the Settings entry itself reads as top-level nav.

### Decision 4: Default landing route

**Chosen:** Change the root redirect from `/providers` to `/browse`.

**Rationale:** Now that Providers is a configuration surface, defaulting new users to it misrepresents the app's primary purpose. The browse flow is the right first impression. First-run users who have no providers configured will still see empty states on `/browse`; that friction is acceptable and informative.

**Alternative considered:** Redirect to `/settings` on first run (when no providers configured) and `/browse` otherwise. Rejected — requires reading provider state in a router guard, which introduces async timing and a dependency on Wails being initialized before the first route resolves. Not worth the complexity for a cosmetic improvement.

### Decision 5: Tab component — inline in `SettingsView.vue` vs. extracted child components

**Chosen:** Extract `ProvidersTab.vue` and `MarketplacesTab.vue` alongside `SettingsView.vue`; the view renders the daisyUI tab bar and `<KeepAlive>` wraps the active tab component.

**Rationale:** The existing `ProvidersView.vue` and `MarketplacesView.vue` each contain enough template and composition-API setup that inlining both into a single `SettingsView.vue` would make the file unwieldy. Extracting to two focused components preserves the single-responsibility boundary. `<KeepAlive>` avoids refetching store state on every tab switch.

**Alternative considered:** Keep `ProvidersView.vue` and `MarketplacesView.vue` and have `SettingsView.vue` render them directly as tab panes. Rejected because those view files contain outer layout wrappers (page titles, spacing) that no longer apply inside a tab pane; the components need trimming anyway, so making the trimmed versions purpose-named (`*Tab.vue`) reads more clearly.

## Risks / Trade-offs

- **[Risk]** Users who bookmarked `/providers` or `/marketplaces` arrive at a redirect and briefly see a URL change. **Mitigation:** The redirect is instantaneous; Vue Router rewrites the URL bar before any paint.
- **[Risk]** The sidebar footer icon may be less discoverable than a top-level nav item, particularly for first-time users. **Mitigation:** Pair the icon with a tooltip and ensure the icon is a well-known gear (⚙️) glyph. If discoverability issues surface in practice, a first-run tooltip or coach mark can be added as a follow-up.
- **[Trade-off]** Two-tab settings feels slightly over-engineered today, but the structure pays off as soon as a third settings category (theme, API tokens, update channel) is added. The alternative — a flat settings page that lists both sections stacked vertically — scales worse.
- **[Risk]** Existing Playwright / E2E tests (if any) may hard-code `/providers` navigation. **Mitigation:** Grep for the old paths during implementation and update call sites; the redirect covers cases we miss.

## Migration Plan

1. Implement the new route + view + tab components.
2. Add the sidebar footer and settings icon button; wire the `<RouterLink>` to `/settings`.
3. Convert `/providers` and `/marketplaces` to redirect records.
4. Remove them from the `navItems` array in `DefaultLayout.vue`.
5. Update the root redirect from `/providers` to `/browse`.
6. Delete or reduce `ProvidersView.vue` and `MarketplacesView.vue` once the tab components are confirmed working.
7. Run `pnpm type-check`, `pnpm lint`, `pnpm format`; manually verify `wails dev` that all four surfaces (Browse, Installed, Settings → Providers, Settings → Marketplaces) still function.

**Rollback strategy:** Revert the branch. No data migrations, no backend changes, no schema touch — rollback is a pure code revert.

## Open Questions

- Should the tab bar use daisyUI's `tabs-bordered` or `tabs-lifted`? (Deferring to the existing UI vocabulary; `tabs-bordered` matches the rest of the app's flat style.)
- Is there appetite to add a third "General" tab in this change for theme/appearance settings currently persisted only in `stores/theme.ts`? (Deferred — keep this change tightly scoped to relocation; surfacing theme settings in UI is its own proposal.)
