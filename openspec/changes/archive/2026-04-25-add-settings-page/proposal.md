## Why

The Providers and Marketplaces pages are both configuration surfaces — the user only visits them during setup or when changing environment-level settings, not during the primary plugin-browsing workflow. Giving each its own top-level sidebar entry clutters the main navigation and dilutes the intent of the sidebar, which is meant to expose the day-to-day flow (Browse → Installed). Consolidating them into a single Settings page with tabs mirrors the conventional desktop-app settings pattern and frees the sidebar for primary actions.

## What Changes

- Add a new `/settings` route with a tabbed layout containing two tabs: **Providers** and **Marketplaces**.
- Move the existing provider management UI (from `ProvidersView.vue`) into a `ProvidersTab.vue` component rendered inside Settings.
- Move the existing marketplace management UI (from `MarketplacesView.vue`) into a `MarketplacesTab.vue` component rendered inside Settings.
- **BREAKING** Remove the top-level `/providers` and `/marketplaces` routes from the sidebar navigation. The routes themselves may redirect to `/settings?tab=<name>` for backward compatibility with bookmarked URLs.
- Add a sidebar footer region to `DefaultLayout.vue` containing a settings icon button that navigates to `/settings`. The layout currently has no sidebar footer, so the footer container is also new.
- Update the default redirect in `router/index.ts` from `/providers` to `/browse` so a fresh launch lands on the primary flow rather than a configuration screen.
- Preserve deep-link behavior: `/settings` opens the Providers tab by default; `/settings?tab=marketplaces` opens the Marketplaces tab directly.

## Capabilities

### New Capabilities

- `settings-page`: Tabbed Settings view that hosts configuration surfaces (Providers, Marketplaces) and is reached via a sidebar-footer icon button rather than a top-level nav entry.

### Modified Capabilities

- `wails-app-shell`: Sidebar no longer lists Providers or Marketplaces as top-level items; a new sidebar footer contains a settings icon button; default landing route changes from `/providers` to `/browse`.
- `provider-management`: The provider list now renders inside the Providers tab of the Settings page instead of a standalone `/providers` route.
- `marketplace-management`: The marketplace list now renders inside the Marketplaces tab of the Settings page instead of a standalone `/marketplaces` route.

## Impact

- Frontend routes (`frontend/src/router/index.ts`): `/settings` added, default redirect updated, `/providers` and `/marketplaces` either removed or converted to redirects.
- Frontend views: new `frontend/src/views/SettingsView.vue`; `ProvidersView.vue` and `MarketplacesView.vue` either deleted or reduced to thin wrappers that redirect.
- Frontend components: new `frontend/src/components/settings/ProvidersTab.vue` and `MarketplacesTab.vue`; existing provider/marketplace card components (`ProviderCard.vue`, `MarketplaceCard.vue`) are reused unchanged.
- Layout: `frontend/src/layouts/DefaultLayout.vue` gains a sidebar footer region with a settings icon button.
- No backend changes. Wails bindings, Pinia stores (`stores/provider.ts`, `stores/marketplace.ts`), and Go services are untouched.
- Tests: existing Pinia store tests in `frontend/src/stores/__tests__/` are unaffected; may add a route/nav snapshot test.
