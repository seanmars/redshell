## Why

The frontend already depends on daisyUI 5 + Tailwind 4, but daisyUI component classes (`modal`, `tabs`, `alert`, `input`, `checkbox`, `badge`, `collapse`, `toast`, `card`, raw `btn`) are scattered directly across views and tab components instead of being encapsulated by the `App*` primitives in `components/ui/`. The result is duplicated markup (two different tab bars in `SettingsView.vue` vs `InstalledPluginsView.vue`, two inline `<dialog class="modal">` blocks, hand-rolled toast state with `setTimeout`), inconsistent styling, and view files that mix layout, state, and design-system concerns. The team also lacks a written rule that distinguishes "design-system class" (must be wrapped) from "Tailwind utility" (free to use), so even well-intentioned new code drifts.

## What Changes

- Codify a frontend convention in `CLAUDE.md`: daisyUI **component** classes that have an `App*` primitive (`btn`, `card`, `alert`, `modal`, `tabs`, `tab`, `input`, `checkbox`, `collapse`, `badge`, `loading`, `toast`) only live inside that primitive; views may use Tailwind **utility** classes freely. Single-use daisyUI utilities without a primitive (`menu` for the sidebar nav, `swap` for icon-toggle widgets, `btn-circle` icon-shells on `<RouterLink>` / `<label>` hosts) are documented exceptions and may stay inline. Vue 3 Composition API (`<script setup>`) is mandatory, and shared logic must extract into composables under `src/composables/`.
- Add the missing daisyUI primitives in `components/ui/`: `AppModal` (slot-shell with `header` / default body / `actions` slots), `AppTabs` + `AppTab`, `AppInput`, `AppCheckbox`, `AppCollapse`, `AppBadge`, `AppSpinner`, `AppToast` + matching `useToast` composable.
- Refactor `AppConfirmModal` to compose `AppModal` instead of re-emitting `<dialog class="modal">` itself.
- Extract shared view logic into composables: `useToast` (replacing the ad-hoc `setTimeout` notification block in `InstalledPluginsView.vue`) and `usePluginInstaller` (the `selectedMergedPlugins` / `availableProviders` / install-modal flow currently inlined in `BrowsePluginsView.vue`).
- Migrate every view and tab component to consume only the primitives + composables: `BrowsePluginsView.vue`, `InstalledPluginsView.vue`, `SettingsView.vue`, `components/settings/MarketplacesTab.vue`, `components/settings/ProvidersTab.vue`. After migration, `rg "class=\"[^\"]*\\b(btn|card|alert|modal|tabs|tab|input|checkbox|collapse|badge|loading|toast)\\b" frontend/src/views frontend/src/components/settings frontend/src/components/plugin frontend/src/components/marketplace frontend/src/components/provider` returns nothing. `frontend/src/layouts/` and `frontend/src/components/ThemeToggle.vue` are deliberately excluded from this scan as the documented-exception scope.
- Reorganize `components/` to enforce the primitive vs. domain split that CLAUDE.md already describes but the folder layout currently violates: keep only `App*.vue` primitives in `components/ui/`; move domain cards (`PluginCard.vue`, `MarketplaceCard.vue`, `ProviderCard.vue`, `InstalledPluginCard.vue`) into per-domain folders (`components/plugin/`, `components/marketplace/`, `components/provider/`).
- **BREAKING (internal only)** Import paths for the moved cards change (e.g. `@/components/ui/PluginCard.vue` → `@/components/plugin/PluginCard.vue`). No external API surface is affected — Wails bindings, Pinia store signatures, and Go services are untouched.

## Capabilities

### New Capabilities

- `frontend-design-system`: The codified rules and primitive/composable inventory governing how daisyUI is consumed in the Vue frontend — what counts as a design-system class vs. a layout utility, where primitives live, when to extract a composable, and the Composition API style requirements.

### Modified Capabilities

<!-- None. The user-visible behavior of plugin-browser, installed-plugins-view,
     settings-page, marketplace-management, provider-management, and wails-app-shell
     does not change. This refactor is internal: same screens, same flows, same store
     signatures — just consistent component composition underneath. -->

## Impact

- **Frontend components** (`frontend/src/components/`): adds `ui/AppModal.vue`, `ui/AppTabs.vue`, `ui/AppTab.vue`, `ui/AppInput.vue`, `ui/AppCheckbox.vue`, `ui/AppCollapse.vue`, `ui/AppBadge.vue`, `ui/AppSpinner.vue`, `ui/AppToast.vue`; rewrites `ui/AppConfirmModal.vue` on top of `AppModal`; relocates `PluginCard.vue`, `MarketplaceCard.vue`, `ProviderCard.vue`, `InstalledPluginCard.vue` out of `ui/` into per-domain folders.
- **Frontend composables** (`frontend/src/composables/`): adds `useToast.ts`, `usePluginInstaller.ts`. Existing `useConfirm.ts`, `usePageTitle.ts` are unchanged.
- **Frontend views** (`frontend/src/views/`, `frontend/src/components/settings/`): `BrowsePluginsView.vue`, `InstalledPluginsView.vue`, `SettingsView.vue`, `MarketplacesTab.vue`, `ProvidersTab.vue` are migrated to consume only primitives + composables. No router or store-shape changes.
- **Layout cleanup** (`frontend/src/layouts/DefaultLayout.vue`): the unused `hideFooter` prop and the empty `<footer class="footer footer-center">` element are removed. No consumer ever passed `hideFooter` and the footer rendered nothing visible — keeping it would have required an exception for the `footer` daisyUI class. Removing it is the smaller change.
- **Documentation** (`CLAUDE.md` at repo root): adds a "Frontend Design System" section enumerating the primitives, the composable rule, the daisyUI-class boundary, and the folder layout. This becomes the authoritative reference for future frontend work.
- **Backend, Wails bindings, Pinia stores, tests**: untouched. Existing store unit tests in `frontend/src/stores/__tests__/` continue to pass without modification because store APIs do not change.
- **Tests**: new minimal Vitest specs for the primitives that have non-trivial logic (`AppTabs` keyboard handling, `useToast` auto-dismiss timer) under `frontend/src/components/ui/__tests__/` and `frontend/src/composables/__tests__/`.
