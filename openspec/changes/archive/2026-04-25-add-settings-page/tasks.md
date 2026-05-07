## 1. Router changes

- [x] 1.1 Add a `/settings` route in `frontend/src/router/index.ts` that lazy-loads `views/SettingsView.vue`
- [x] 1.2 Convert the existing `/providers` route to a redirect record pointing at `/settings?tab=providers`
- [x] 1.3 Convert the existing `/marketplaces` route to a redirect record pointing at `/settings?tab=marketplaces`
- [x] 1.4 Change the root `/` redirect target from `/providers` to `/browse`

## 2. Settings view and tab components

- [x] 2.1 Create `frontend/src/views/SettingsView.vue` rendering a daisyUI `tabs tabs-bordered` bar with two tabs (Providers, Marketplaces) wrapped in `layouts/DefaultLayout.vue`
- [x] 2.2 Implement tab-state sync in `SettingsView.vue`: read `route.query.tab` on mount, default to `providers` if absent or unknown, and call `router.replace` when the user switches tabs
- [x] 2.3 Wrap the active tab content in `<KeepAlive>` so state and scroll position survive tab switches
- [x] 2.4 Create `frontend/src/components/settings/ProvidersTab.vue` by extracting the body of `ProvidersView.vue` (keep the call to `stores/provider.ts`; drop outer page-title/layout wrappers not appropriate inside a tab pane)
- [x] 2.5 Create `frontend/src/components/settings/MarketplacesTab.vue` by extracting the body of `MarketplacesView.vue` (keep the call to `stores/marketplace.ts`; drop outer page-title/layout wrappers)

## 3. Sidebar footer and settings button

- [x] 3.1 In `frontend/src/layouts/DefaultLayout.vue`, remove `Providers` and `Marketplaces` entries from the `navItems` array (leave `Browse`, `Installed`)
- [x] 3.2 Add a new footer region inside the `<aside>` element (below `<nav>`, above the sidebar's bottom edge) styled with a top divider
- [x] 3.3 Inside the sidebar footer, add a `<RouterLink to="/settings">` rendered as an icon-only `btn btn-ghost btn-circle` with a gear/settings glyph and a `title`/`aria-label` of "Settings"
- [x] 3.4 Verify the main content `footer` element (with `v-if="!hideFooter"`) is untouched — the new footer lives inside the sidebar column, not the main column

## 4. Clean up legacy view files

- [x] 4.1 Delete `frontend/src/views/ProvidersView.vue` (content moved into `ProvidersTab.vue`)
- [x] 4.2 Delete `frontend/src/views/MarketplacesView.vue` (content moved into `MarketplacesTab.vue`)
- [x] 4.3 Search the frontend for any remaining imports or references to `ProvidersView`/`MarketplacesView` and remove them
- [x] 4.4 Search frontend sources and docs for hardcoded path strings `'/providers'` and `'/marketplaces'`; update links to `/settings?tab=<name>` where appropriate (router redirect still covers missed cases)

## 5. Validation

- [x] 5.1 Run `pnpm type-check` in `frontend/` and fix any type errors surfaced by the component extractions
- [x] 5.2 Run `pnpm lint` and `pnpm format` in `frontend/`
- [x] 5.3 Run `wails dev` and manually verify: (a) app lands on `/browse`, (b) clicking the sidebar footer settings button opens `/settings` with Providers tab active, (c) switching to Marketplaces updates the URL to `?tab=marketplaces`, (d) refreshing the page on the Marketplaces tab keeps it active, (e) typing `/providers` in-app redirects to `/settings?tab=providers`, (f) Providers and Marketplaces still interact correctly with their Pinia stores (add marketplace, remove marketplace, install plugin still works end to end)
- [x] 5.4 Run `pnpm run test:unit` and confirm store tests still pass (they should — no store signatures changed)

## 6. Finalize

- [x] 6.1 Run `openspec status --change "add-settings-page"` and confirm all artifacts are `done`
- [x] 6.2 Run `openspec validate add-settings-page --strict` and resolve any validation issues
- [x] 6.3 Open a commit grouped by concern: router, settings view + tabs, sidebar footer, cleanup
