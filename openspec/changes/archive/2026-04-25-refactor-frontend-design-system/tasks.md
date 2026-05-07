## 1. Document conventions in CLAUDE.md (Phase 0)

- [x] 1.1 Add a "Frontend Design System" section to repository-root `CLAUDE.md` listing the **inventoried** daisyUI component classes that must be wrapped (`btn`, `card`, `alert`, `modal`, `tabs`, `tab`, `input`, `checkbox`, `collapse`, `badge`, `loading`, `toast` and their `<base>-<modifier>` variants) versus Tailwind utility classes that may stay free in views
- [x] 1.1a In the same section, document the **explicit exceptions**: `menu`/`menu-*` allowed inside `frontend/src/layouts/` for sidebar nav; `swap`/`swap-*` allowed inside `frontend/src/components/ThemeToggle.vue`; `btn-ghost btn-circle` icon-shell allowed only on `<RouterLink>`/`<label>` hosts where `AppButton` (which renders `<button>`) cannot be substituted
- [x] 1.2 In the same section, document the rule that all `.vue` files use `<script setup lang="ts">` and that shared logic is extracted into composables under `frontend/src/composables/` (filename starts with `use*`)
- [x] 1.3 In the same section, list the canonical primitives that will live in `frontend/src/components/ui/`: `AppAlert`, `AppBadge`, `AppButton`, `AppCard`, `AppCheckbox`, `AppCollapse`, `AppConfirmModal`, `AppInput`, `AppModal`, `AppSpinner`, `AppTab`, `AppTabs`, `AppToast`
- [x] 1.4 In the same section, document the folder layout: `components/ui/` for primitives only; `components/plugin/`, `components/marketplace/`, `components/provider/`, `components/settings/` for domain components
- [x] 1.5 In the same section, add the `rg` command from the spec scenario as the mechanical check for daisyUI-class leakage (excluding `frontend/src/layouts/` and `frontend/src/components/ThemeToggle.vue`, which hold the documented exceptions) and require it in PR review for frontend changes

## 2. Add missing primitives in `components/ui/` (Phase 1a)

- [x] 2.1 Create `frontend/src/components/ui/AppModal.vue` as a slot-shell with `header` and `actions` named slots, default body slot, props `isOpen: boolean` and `size: 'sm' | 'md' | 'lg'` (default `'md'`), emits `close`; owns `<dialog class="modal modal-bottom sm:modal-middle">` markup, backdrop `<form method="dialog">`, and Escape-to-close behavior
- [x] 2.2 Create `frontend/src/components/ui/AppTabs.vue` accepting `v-model:active` (string), prop `variant: 'bordered' | 'boxed'` (default `'bordered'`); renders `role="tablist"` and discovers child `AppTab` components via `provide`/`inject` (parent provides a `registerTab` / `unregisterTab` API; each `AppTab` registers on `onMounted` and unregisters on `onBeforeUnmount`). Do not introspect slot vnodes — that fights SSR and stale-children edge cases.
- [x] 2.3 Create `frontend/src/components/ui/AppTab.vue` accepting `id: string` and `label: string` props; registers itself with the parent `AppTabs` via `inject` and exposes its default slot to the parent for conditional rendering when active
- [x] 2.4 Create `frontend/src/components/ui/AppInput.vue` wrapping `<input class="input input-bordered">` with `v-model`, props `type` (default `'text'`), `placeholder`, `disabled`, `size: 'sm' | 'md' | 'lg'` (default `'md'`), and a `label` prop that wraps the input in a daisyUI `form-control`/`label` block when provided
- [x] 2.5 Create `frontend/src/components/ui/AppCheckbox.vue` wrapping `<input type="checkbox" class="checkbox">` with `v-model` (must support both boolean binding **and** array binding — when bound to an array via `v-model` with a `:value` prop, toggling the checkbox adds or removes that value from the array, matching native `<input type="checkbox">` behavior under Vue's `v-model`), props `variant: DaisyColor` (default `'primary'`), `size`, `disabled`, and a default slot for the visible label
- [x] 2.6 Create `frontend/src/components/ui/AppCollapse.vue` wrapping `<details class="collapse collapse-arrow bg-base-200">`, props `title: string`, `defaultOpen: boolean` (default `true`); exposes `title` slot (overrides the prop) and a default slot for body
- [x] 2.7 Create `frontend/src/components/ui/AppBadge.vue` wrapping `<span class="badge">` with props `variant: DaisyColor | 'outline'`, `size: 'xs' | 'sm' | 'md' | 'lg'` (default `'sm'`), pass-through `class`/`style` so callers can apply per-domain colors when no semantic slot fits
- [x] 2.7a Create `frontend/src/components/ui/AppSpinner.vue` wrapping `<span class="loading">` with props `variant: 'spinner' | 'dots' | 'ring' | 'ball' | 'bars'` (default `'spinner'`) and `size: DaisySize` (default `'md'`); renders the appropriate `loading-<variant>` and `loading-<size>` modifier classes
- [x] 2.8 Create `frontend/src/components/ui/AppToast.vue` rendering a `toast toast-top toast-end` container that consumes the toast queue from `useToast()` and renders each as an `alert` with the appropriate `alert-<type>` class; meant to be mounted exactly once in `DefaultLayout.vue`

## 3. Refactor `AppConfirmModal` to compose `AppModal` (Phase 1b)

- [x] 3.1 Rewrite `frontend/src/components/ui/AppConfirmModal.vue` so its template renders `<AppModal :is-open="isOpen" size="sm" @close="emit('cancel')">`, fills the `header` slot with the title, the body with the message, and the `actions` slot with the Cancel/Confirm `AppButton` pair; preserve the existing emits (`confirm`, `cancel`) and props (`isOpen`, `title`, `message`, `confirmLabel`, `cancelLabel`)
- [x] 3.2 Remove the inline `<dialog class="modal modal-bottom sm:modal-middle">` markup and the `<form method="dialog" class="modal-backdrop">` block from `AppConfirmModal.vue`; both responsibilities now live inside `AppModal`
- [x] 3.3 Verify the existing call site in `frontend/src/views/InstalledPluginsView.vue` and `frontend/src/components/settings/MarketplacesTab.vue` still works without code changes (props and emits are unchanged)

## 4. Add composables (Phase 1c)

- [x] 4.1 Create `frontend/src/composables/useToast.ts` exporting `useToast()` that returns `{ toasts, push, dismiss }`; back the queue with a module-level `ref<Toast[]>([])` so all callers share one queue; auto-dismiss after `toast.duration` (default 3000 ms) using `setTimeout` and clear the timer on manual `dismiss`
- [x] 4.2 Add a `Toast` type to `useToast.ts`: `{ id: string; type: 'success' | 'error' | 'info' | 'warning'; message: string; duration?: number }`; generate `id` via `crypto.randomUUID()` or a monotonic counter
- [x] 4.3 Create `frontend/src/composables/usePluginInstaller.ts` extracting the install flow from `BrowsePluginsView.vue`: state (`showInstallModal`, `targetProviders`, `installError`), computed (`selectedMergedPlugins`, `availableProviders`), watcher (auto-select sole provider when modal opens), and `handleInstall(installFn)` that walks providers and calls the supplied installer; the composable depends on `usePluginStore` and returns the state/functions for the view to bind
- [x] 4.4 Add minimal Vitest specs under `frontend/src/composables/__tests__/useToast.spec.ts` covering: push adds a toast, auto-dismiss after the default timeout removes it, dismiss removes immediately, multiple stacked toasts coexist
- [x] 4.5 Run `pnpm type-check` and `pnpm lint` in `frontend/` and resolve any errors before proceeding to Phase 2

## 5. Mount `<AppToast />` in the layout (Phase 2a)

- [x] 5.1 Import `AppToast` in `frontend/src/layouts/DefaultLayout.vue` and render `<AppToast />` once at the bottom of the layout's root container, outside the `<aside>` and `<main>` elements so toasts position correctly relative to the viewport

## 5b. Verify `DefaultLayout.vue` falls within documented exceptions

- [x] 5b.1 Confirm the only daisyUI classes in `frontend/src/layouts/DefaultLayout.vue` are `menu`/`menu-md` (sidebar nav `<ul>`) and `btn btn-ghost btn-circle` (settings `<RouterLink>` icon-shell). The previously-existing empty `footer footer-center` element and its unused `hideFooter` prop were removed (vestigial — no consumer ever passed `hideFooter`, and the footer rendered nothing visible). Both remaining classes fall under the documented exceptions from Phase 1.

## 6. Migrate `MarketplacesTab.vue` to primitives (Phase 2b)

- [x] 6.1 In `frontend/src/components/settings/MarketplacesTab.vue`, replace the inline `<dialog class="modal ...">` add-marketplace modal with `<AppModal :is-open="showAddModal" @close="showAddModal = false; addError = null">`, filling `header` with "Add Marketplace", body with the URL input, and `actions` with Cancel/Add `AppButton`s
- [x] 6.2 Replace the raw `<input v-model="newURL" type="url" class="input input-bordered ...">` with `<AppInput v-model="newURL" type="url" placeholder="https://github.com/owner/repo" label="Repository URL" @keyup.enter="handleAdd" />`
- [x] 6.3 Verify `<AppConfirmModal>` block at the bottom is unchanged (props and emits are preserved by Phase 3)
- [x] 6.4 Replace the `<span class="loading loading-spinner loading-lg" />` empty-state spinner with `<AppSpinner size="lg" />`
- [x] 6.5 Confirm `rg "class=\"[^\"]*\\b(modal|input|loading)\\b" frontend/src/components/settings/MarketplacesTab.vue` returns no matches

## 7. Migrate `ProvidersTab.vue` to primitives (Phase 2c)

- [x] 7.1 In `frontend/src/components/settings/ProvidersTab.vue`, replace `<span class="loading loading-spinner loading-lg" />` with `<AppSpinner size="lg" />`; no other raw daisyUI component classes are present in this file
- [x] 7.2 Confirm `rg "class=\"[^\"]*\\b(btn|card|alert|modal|tabs|input|checkbox|badge|collapse|loading|toast)\\b" frontend/src/components/settings/ProvidersTab.vue` returns no matches

## 8. Migrate `SettingsView.vue` to `AppTabs` (Phase 2d)

- [x] 8.1 In `frontend/src/views/SettingsView.vue`, replace the hand-rolled `<div role="tablist" class="tabs tabs-bordered">` block with `<AppTabs v-model:active="activeTab" variant="bordered">` containing two `<AppTab id="marketplaces" label="Marketplaces"><MarketplacesTab /></AppTab>` and `<AppTab id="providers" label="Providers"><ProvidersTab /></AppTab>` entries
- [x] 8.2 Adapt the existing `activeTab` computed and `selectTab` writer to drive `v-model:active` (read from `route.query.tab`, write back via `router.replace`); KeepAlive removed because `AppTab` uses `v-if="isActive"` per spec ("only the active tab's default slot content SHALL be rendered") — switching tabs unmounts/remounts the inactive pane, which means each tab's onMounted re-fires. Acceptable because the stores cache results and the refetch is cheap.
- [x] 8.3 Confirm `rg "class=\"[^\"]*\\btabs?\\b" frontend/src/views/SettingsView.vue` returns no matches

## 9. Migrate `InstalledPluginsView.vue` to primitives + `useToast` (Phase 2e)

- [x] 9.1 In `frontend/src/views/InstalledPluginsView.vue`, replace the hand-rolled provider tab bar (`<div class="tabs tabs-boxed">` plus two `<button class="tab">`) with `<AppTabs v-model:active="activeProvider" variant="boxed">` and two `<AppTab id="claude" label="Claude Code">` and `<AppTab id="copilot" label="GitHub Copilot">` entries; the empty-state message and the `InstalledPluginCard` list move into the active tab's default slot
- [x] 9.2 Replace the local `notification` ref, `notifyTimer`, `notify()` function, and inline `toast toast-top toast-end` markup with `useToast` calls: `const toast = useToast()` and `toast.push({ type: 'success', message: ... })` / `toast.push({ type: 'error', message: ... })` in the success and catch branches of `handleUninstall`
- [x] 9.3 Remove the `<div v-if="notification" class="toast ...">` block at the bottom of the template entirely; the layout-mounted `<AppToast />` now renders all notifications
- [x] 9.3a Replace the `<span class="loading loading-spinner loading-lg" />` empty-state spinner with `<AppSpinner size="lg" />`
- [x] 9.4 Confirm `rg "class=\"[^\"]*\\b(tabs?|toast|alert|loading)\\b" frontend/src/views/InstalledPluginsView.vue` returns no matches and that `notifyTimer` and `notification` no longer appear in `<script setup>`

## 10. Migrate `BrowsePluginsView.vue` to primitives + `usePluginInstaller` (Phase 2f)

- [x] 10.1 In `frontend/src/views/BrowsePluginsView.vue`, replace the local `showInstallModal`, `targetProviders`, `installError`, `selectedMergedPlugins`, `availableProviders`, the `watch(showInstallModal, ...)` block, and `handleInstall` function with a single call to `usePluginInstaller(store)` that returns those bindings; the view's `<script setup>` shrinks to data fetching, refresh handler, and small helpers
- [x] 10.2 Replace the `<details class="collapse collapse-arrow bg-base-200">` per-marketplace section with `<AppCollapse :title="marketplaceDisplayName(m)" :default-open="true">`; keep the slot body identical. Behavior change to flag in PR description: the existing `<details ... open>` attribute forces the section open on every render, while `:default-open="true"` only sets initial state — users can collapse a section after this change. Treat as a small UX improvement.
- [x] 10.3 Replace the inline `<div class="alert alert-warning ...">` refresh-warning block with `<AppAlert type="warning">`
- [x] 10.4 Replace the inline `<dialog class="modal ...">` install modal with `<AppModal :is-open="showInstallModal" @close="showInstallModal = false">`, filling `header` with "Install Plugins", body with the provider checkboxes / selected-plugins list / install-log / error line, and `actions` with the Cancel/Install `AppButton` pair
- [x] 10.5 Replace the two raw `<input type="checkbox" class="checkbox checkbox-primary">` provider toggles with `<AppCheckbox v-model="targetProviders" :value="..." :disabled="...">` (or use a small `AppCheckbox` group helper if the v-model array binding requires it); ensure the disabled-state styling continues to match `availableProviders.has(prov)`
- [x] 10.5a Replace the `<span class="loading loading-spinner loading-lg" />` empty-state spinner with `<AppSpinner size="lg" />`
- [x] 10.6 Confirm `rg "class=\"[^\"]*\\b(modal|alert|collapse|checkbox|loading|toast|tabs?)\\b" frontend/src/views/BrowsePluginsView.vue` returns no matches and that the install flow still produces the same install-log and error behavior in `wails dev`

## 11. Audit remaining domain card components (Phase 2g)

- [x] 11.1 Read `frontend/src/components/ui/PluginCard.vue` and replace the `<span class="badge badge-sm gap-1 border" :style="...">` with `<AppBadge size="sm" :style="...">`; the inline `:style` for per-provider hsl colors stays per the open-question decision
- [x] 11.2 Read `frontend/src/components/ui/MarketplaceCard.vue` and replace `<span class="badge badge-outline badge-xs">` with `<AppBadge variant="outline" size="xs">`
- [x] 11.3 Read `frontend/src/components/ui/ProviderCard.vue` and replace `<span class="badge badge-success badge-sm">` / `<span class="badge badge-neutral badge-sm">` with `<AppBadge variant="success" size="sm">` / `<AppBadge variant="neutral" size="sm">`
- [x] 11.4 Read `frontend/src/components/ui/InstalledPluginCard.vue` — already uses `AppButton` and a plain Tailwind card layout; verify no daisyUI component classes remain
- [x] 11.5 Verify `frontend/src/components/ThemeToggle.vue` falls within the documented exceptions from Phase 1 (`swap`/`swap-*` allowed here; `btn-ghost btn-circle` allowed because the host is a `<label>`, not a `<button>`). No code change required — but if Phase 1's CLAUDE.md draft did not capture this exception precisely, amend the docs now before declaring task 13.5 green.

## 12. Folder reorganization (Phase 3)

- [x] 12.1 Create `frontend/src/components/plugin/`, `frontend/src/components/marketplace/`, `frontend/src/components/provider/` directories
- [x] 12.2 Move `frontend/src/components/ui/PluginCard.vue` → `frontend/src/components/plugin/PluginCard.vue`
- [x] 12.3 Move `frontend/src/components/ui/InstalledPluginCard.vue` → `frontend/src/components/plugin/InstalledPluginCard.vue` (also migrated raw `card`/`card-body` markup to `<AppCard compact>` — Phase 2g audit at task 11.4 missed this)
- [x] 12.4 Move `frontend/src/components/ui/MarketplaceCard.vue` → `frontend/src/components/marketplace/MarketplaceCard.vue`
- [x] 12.5 Move `frontend/src/components/ui/ProviderCard.vue` → `frontend/src/components/provider/ProviderCard.vue`
- [x] 12.6 Update every import of the four moved files across `frontend/src/`; run `rg "components/ui/(Plugin|InstalledPlugin|Marketplace|Provider)Card" frontend/src` and verify no matches remain
- [x] 12.7 Run `pnpm type-check` and resolve any dangling imports

## 13. Validation (Phase 4)

- [x] 13.1 Run `pnpm format` in `frontend/`
- [x] 13.2 Run `pnpm lint` in `frontend/` and resolve any warnings
- [x] 13.3 Run `pnpm type-check` in `frontend/` and confirm zero errors
- [x] 13.4 Run `pnpm run test:unit` in `frontend/` and confirm existing store tests plus new `useToast` test pass (11/11 pass)
- [x] 13.5 Run the leak check: `rg "class=\"[^\"]*\\b(btn|card|alert|modal|tabs|tab|input|checkbox|collapse|badge|loading|toast|dropdown|avatar|select|textarea|radio|toggle|range|footer)\\b" frontend/src/views frontend/src/components/settings frontend/src/components/plugin frontend/src/components/marketplace frontend/src/components/provider` and confirm zero matches. Spot-checked `frontend/src/layouts/DefaultLayout.vue` and `frontend/src/components/ThemeToggle.vue` — the only matches are `menu`/`menu-md`, `swap`/`swap-rotate`/`swap-on`/`swap-off`, and `btn btn-ghost btn-circle` icon-shells on `<RouterLink>` and `<label>` hosts, all within the documented exceptions.
- [ ] 13.6 Run `wails dev` and manually verify each flow: settings tabs switch and deep-link via `?tab=`, marketplace add/remove works, plugin browse loads and displays per-marketplace collapses, plugin selection toggles, install modal opens with provider checkboxes and shows live install log, installed-plugin uninstall fires a toast notification, theme toggle still works
- [x] 13.7 Run `openspec validate refactor-frontend-design-system --strict` and resolve any validation issues
- [x] 13.8 Run `openspec status --change refactor-frontend-design-system` and confirm all artifacts are `done`

## 14. Finalize

- [ ] 14.1 Open commits grouped by phase: (a) CLAUDE.md update, (b) primitives + composables, (c) view migrations (one commit per migrated view if diffs are large), (d) folder reorg, (e) any leftover lint/format-only changes
- [ ] 14.2 Confirm the working tree is clean and the branch is ready for PR review
