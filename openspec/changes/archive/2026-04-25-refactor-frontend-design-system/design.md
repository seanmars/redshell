## Context

The frontend stack is Vue 3.5 + Tailwind 4 + daisyUI 5, with Pinia stores wrapping Wails-generated bindings. A small set of `App*` primitives already exists in `frontend/src/components/ui/` (`AppButton`, `AppCard`, `AppAlert`, `AppConfirmModal`), but the rest of the codebase reaches for raw daisyUI classes whenever a primitive is missing. Concrete evidence:

- Two different tab implementations: `views/SettingsView.vue:55-68` (`tabs tabs-bordered`) and `views/InstalledPluginsView.vue:60-76` (`tabs tabs-boxed`). Same daisyUI feature, two hand-rolled markup blocks, no shared component.
- Three inline `<dialog class="modal modal-bottom sm:modal-middle">` blocks: `views/BrowsePluginsView.vue:167`, `components/settings/MarketplacesTab.vue:74`, plus the legitimate one inside `components/ui/AppConfirmModal.vue:19` — but `AppConfirmModal` itself reimplements modal markup rather than composing a primitive.
- Ad-hoc notification state in `views/InstalledPluginsView.vue:18-27` with manual `setTimeout` bookkeeping; the actual toast markup is at lines 110-117. There is no `useToast` composable, so the next view that needs notifications will copy this pattern.
- Raw form primitives: `<input type="url" class="input input-bordered">` at `MarketplacesTab.vue:79-85`, `<input type="checkbox" class="checkbox checkbox-primary">` at `BrowsePluginsView.vue:182-189`. No `AppInput` or `AppCheckbox` exists.
- Raw `<details class="collapse collapse-arrow bg-base-200">` at `BrowsePluginsView.vue:124-162`.
- `<span class="badge ...">` repeated across `PluginCard.vue`, `MarketplaceCard.vue`, `ProviderCard.vue` with subtly different modifier sets.
- `BrowsePluginsView.vue` carries 80+ lines of install-flow logic (`selectedMergedPlugins`, `availableProviders`, `targetProviders`, `handleInstall`, install-log rendering) inline in the view's `<script setup>` — none of it is reused, but the size makes the view hard to read and impossible to test independently of the modal markup.

The `components/ui/` folder also mixes pure primitives with view-specific cards (`PluginCard`, `MarketplaceCard`, `ProviderCard`, `InstalledPluginCard`). The CLAUDE.md convention already says primitives are pure and view-specific cards "live next to them" — currently they live *in* the same folder, which doesn't enforce anything.

Composition API setup-style is already used everywhere; the user's "use Composition API" requirement is really "lift shared logic into composables," not "convert from Options API."

## Goals / Non-Goals

**Goals:**

- Establish a single, written rule for what counts as a daisyUI design-system class (must be wrapped) vs. a Tailwind layout utility (free to use), and make that rule the standard CLAUDE.md reference.
- Eliminate every raw daisyUI component class outside `components/ui/App*.vue` — a `rg` for the listed class names against `frontend/src/views/` and `frontend/src/components/settings/` (and the relocated domain folders) returns no matches after the refactor.
- Ship the missing primitives (`AppModal`, `AppTabs` + `AppTab`, `AppInput`, `AppCheckbox`, `AppCollapse`, `AppBadge`, `AppToast`) and the `useToast` composable so views have a complete vocabulary.
- Lift duplicated or ad-hoc view logic into composables (`useToast` for notifications; `usePluginInstaller` for the install-modal state machine in `BrowsePluginsView`).
- Reorganize `components/` so the folder layout matches the convention: `components/ui/` for primitives only; `components/<domain>/` for domain cards.

**Non-Goals:**

- Redesigning any user-visible behavior. Every screen, flow, error state, and store interaction continues to work identically. This is a refactor of the implementation, not the product.
- Changing Pinia stores, Wails bindings, Go services, or routes. The proposal explicitly excludes those.
- Introducing a third-party component library (e.g., Headless UI, Radix Vue) on top of daisyUI. The whole point is to use daisyUI consistently, not replace it.
- Building a Storybook or component playground. The primitives stay close to their consumers; documentation lives in CLAUDE.md, not in a separate tool.
- Theming work beyond the existing daisyUI light/dark toggle. `AppToast` and other primitives consume the existing semantic colors (`primary`, `success`, `warning`, `error`); no new design tokens are introduced.
- Adding animations or transitions beyond what daisyUI already provides for modals, collapses, and toasts.

## Decisions

### Decision 1: Where to draw the line — wrap daisyUI classes that have `App*` primitives; carve explicit exceptions for single-use utilities

**Chosen:** A class MUST live inside an `App*` primitive if it appears in the **inventoried list**: `btn` (AppButton), `card` (AppCard), `alert` (AppAlert), `modal` (AppModal), `tabs` / `tab` (AppTabs / AppTab), `input` (AppInput), `checkbox` (AppCheckbox), `collapse` (AppCollapse), `badge` (AppBadge), `loading` (AppSpinner), `toast` (AppToast) — plus their `<base>-<modifier>` variants (`btn-primary`, `tabs-bordered`, `alert-warning`, `loading-spinner`, etc.). A class is "utility" (free anywhere) if it is a Tailwind layout/spacing/typography/color utility — `flex`, `grid`, `gap-*`, `p-*`, `m-*`, `space-*`, `max-w-*`, `min-w-*`, `truncate`, `text-*`, `font-*`, `bg-*`, `border-*`, `rounded-*`, `shadow-*`, `cursor-*`, `opacity-*`, `transition-*`, `hover:*`, `sm:*`, etc.

**Documented exceptions (allowed inline despite being daisyUI classes):**

- `menu` and its `menu-*` modifiers in `frontend/src/layouts/DefaultLayout.vue` — the sidebar nav uses `<ul class="menu menu-md">` once. Wrapping it in an `AppMenu` primitive that took an array of nav items would be more rigid than the current declarative `<RouterLink>` template, and there is exactly one consumer.
- `swap` and `swap-*` modifiers in `frontend/src/components/ThemeToggle.vue` — daisyUI's icon-toggle animation utility used by a single component. No reuse, no primitive needed.
- `btn-ghost btn-circle` icon-shell when applied to a `<RouterLink>`, `<label>`, or other non-`<button>` host element — `AppButton` renders a `<button>`, so it cannot serve as a `<RouterLink>` or `<label class="swap">` host. The two known cases (sidebar settings link in `DefaultLayout.vue`; theme toggle label in `ThemeToggle.vue`) are kept inline. `btn` and its color/size modifiers on actual `<button>` elements are NOT exempt — those go through `AppButton`.
- `dropdown`, `avatar`, `select`, `textarea`, `radio`, `toggle`, `range`, `footer` — daisyUI classes the codebase does not currently use. If a future view introduces one, the contract is to add a primitive at that point, not to leak the class. The leak-check `rg` pattern includes these by name so accidental introduction is caught immediately.

**Rationale:** The "wrap everything daisyUI ever shipped" version of the rule generated a real conflict — `DefaultLayout.vue` already uses `menu` and `btn-circle` legitimately, and forcing primitives for single-use markup is the over-engineering CLAUDE.md elsewhere warns against. Narrowing to "wrap classes that have a primitive in the inventory" matches the actual problem (duplicate modals, divergent tabs, ad-hoc toasts) and leaves room for genuinely one-off markup. The exceptions list is finite and checked into CLAUDE.md so it does not grow informally.

**Alternative considered:** Build `AppMenu` and an `AppButton.as="link"` polymorphic wrapper to absorb every daisyUI class. Rejected — adds two primitives and a polymorphic API for one consumer each, and `AppMenu` taking an array of nav items would be less expressive than the current declarative template.

**Alternative considered:** Forbid all class names in views and require every visual property to flow through a prop on a primitive. Rejected — this is what UI libraries like MUI do, and it is the wrong fit for a daisyUI + Tailwind stack whose entire premise is utility-first composition. It would balloon the primitive count and make layout work painful.

### Decision 2: `AppModal` is a slot-shell, not a prop-driven dialog

**Chosen:** `AppModal` exposes three slots (`header`, default body, `actions`) plus props `isOpen` (boolean), `size` (`'sm' | 'md' | 'lg'`), and emits `close`. It owns the `<dialog class="modal modal-bottom sm:modal-middle">` markup, the modal-backdrop close behavior, and `Escape` / backdrop-click dismissal. It does not know about forms, install logs, or confirmations.

```vue
<AppModal :is-open="open" size="md" @close="open = false">
  <template #header>Install Plugins</template>
  <!-- arbitrary body content -->
  <template #actions>
    <AppButton variant="ghost" @click="open = false">Cancel</AppButton>
    <AppButton :loading="busy" @click="submit">Install</AppButton>
  </template>
</AppModal>
```

**Rationale:** The three modals in the codebase have wildly different bodies — `BrowsePluginsView`'s modal embeds checkboxes, a plugin list, an install log, and an error line; `MarketplacesTab`'s modal embeds a single URL input; `AppConfirmModal` shows static text. Trying to encode that variety as props (e.g., `:fields="[]"`, `:actions="[]"`) ends up reinventing a render function. Slot-shell keeps the primitive small and lets each consumer compose what it needs.

**Alternative considered:** A prop-driven `AppModal` with `:title`, `:fields`, `:confirmLabel`, `:cancelLabel`. Rejected — `BrowsePluginsView`'s modal alone has too many bespoke regions for that to scale, and we would end up with parallel render paths inside the primitive.

`AppConfirmModal` is then a thin wrapper that fills the three slots with the standard confirm pattern (title, message, Cancel/Confirm buttons) — it composes `AppModal` instead of emitting its own `<dialog>`.

### Decision 3: Tabs use a `v-model:active` + slot pattern, not a `:tabs="[]"` array

**Chosen:** `AppTabs` accepts `v-model:active` (the current tab id) and a `variant` prop (`'bordered' | 'boxed'`). Each tab is declared with `<AppTab id="..." label="...">` as a child component; the active tab's default slot is rendered. Keyboard arrow handling and `role="tablist"` / `role="tab"` ARIA attributes live inside `AppTabs`.

```vue
<AppTabs v-model:active="activeTab" variant="bordered">
  <AppTab id="providers" label="Providers"><ProvidersTab /></AppTab>
  <AppTab id="marketplaces" label="Marketplaces"><MarketplacesTab /></AppTab>
</AppTabs>
```

**Rationale:** Two existing tab implementations diverge in `variant` (`tabs-bordered` vs. `tabs-boxed`) and in how they wire active state (one to `route.query.tab`, one to a local `ref`). A `v-model` keeps the consumer in control of the source of truth — `SettingsView` can still bind to the route query, `InstalledPluginsView` can still bind to a local ref — while the primitive owns the markup and accessibility plumbing.

**Alternative considered:** A flat `<AppTabs :tabs="[{id, label, component}]">` API. Rejected — passing component references through a prop array forces consumers to import the components and pass them in via the array, which is awkward and breaks scoped slots; the component-tag pattern reads more naturally.

**Implementation note:** the recommended discovery pattern is `provide`/`inject` — `AppTabs` exposes a `registerTab` / `unregisterTab` API via `provide`, and each `AppTab` calls `inject` then `registerTab` on `onMounted` and `unregisterTab` on `onBeforeUnmount`. This is the standard Vue 3 idiom and avoids fighting SSR (which slot-children introspection via `useSlots` does, since the slot's children may be cached vnodes rather than mounted instances).

### Decision 4: `AppToast` + `useToast` composable, not a global toast store

**Chosen:** Provide a `useToast()` composable that returns `{ toasts, push(toast), dismiss(id) }` with internal auto-dismiss timer logic. Provide an `AppToast` component that renders the toast container and consumes the same composable via `provide`/`inject` (or a module-singleton `ref` array — see open question below). Mount one `<AppToast />` instance once in `DefaultLayout.vue`. Views call `useToast().push({ type: 'success', message: '...' })`.

**Rationale:** The current implementation in `InstalledPluginsView.vue` holds a single `notification` ref and a `notifyTimer`, which means only one toast can be visible at a time and stacking is impossible. It also can't be reused outside that view. A composable centralizes the timer logic, supports multiple stacked toasts, and decouples "fire a toast" from "where the toast renders." Keeping the data in a composable instead of a Pinia store avoids pulling toast state into the global store catalog when it has no persistence or cross-tab needs.

**Alternative considered:** A Pinia `useToastStore`. Rejected — toasts are ephemeral UI state, not domain state. Putting them in Pinia adds a store entry, devtools noise, and ceremony for no benefit.

### Decision 5: Composables only when shared OR when a view exceeds ~80 lines of `<script setup>`

**Chosen:** Extract a composable when (a) two or more views share the logic, or (b) a single view's `<script setup>` block grows past ~80 lines and the logic forms a coherent unit (e.g., a state machine for a flow). For this refactor that means `useToast` (shared) and `usePluginInstaller` (single-view-but-large). Tiny one-off helpers stay inline.

**Rationale:** Composables have a real cost — they fragment context across files and make jump-to-definition slower. The user's directive to "use composables" is best honored by extracting where extraction pays off, not by dogmatically pulling every `computed` into its own file. The 80-line heuristic catches the `BrowsePluginsView` case (currently ~85 lines of script) without forcing trivial views like `InstalledPluginsView` (~50 lines) to split prematurely.

**Alternative considered:** Extract a composable for every view. Rejected — `MarketplacesTab.vue` and `ProvidersTab.vue` are small and self-contained; splitting them adds files without aiding comprehension.

### Decision 6: Folder reorg — primitives vs. domain cards

**Chosen:** Final layout:

```
frontend/src/components/
  ui/                     # App* primitives only — no business logic, no store imports
    AppAlert.vue
    AppBadge.vue
    AppButton.vue
    AppCard.vue
    AppCheckbox.vue
    AppCollapse.vue
    AppConfirmModal.vue
    AppInput.vue
    AppModal.vue
    AppTab.vue
    AppTabs.vue
    AppToast.vue
  plugin/
    PluginCard.vue
    InstalledPluginCard.vue
  marketplace/
    MarketplaceCard.vue
  provider/
    ProviderCard.vue
  settings/               # tab panes (already exists)
    MarketplacesTab.vue
    ProvidersTab.vue
  ThemeToggle.vue          # cross-cutting; stays at components/ root
```

**Rationale:** The convention CLAUDE.md already documents — primitives in `ui/`, domain cards "next to them" — is currently violated by having domain cards inside `ui/`. Per-domain folders make the layered dependency direction visible (domain folders import from `ui/`; `ui/` imports from nothing in `components/`). It also scales: a future `BrowsePluginsView` filter widget would land in `components/plugin/PluginFilters.vue` without re-debating where it goes.

**Alternative considered:** Flatter layout with all domain cards in one `components/cards/` folder. Rejected — it preserves the "everything in one bag" feel that the current `ui/` already has, just renamed.

### Decision 7: Migration is one PR/commit per phase, with type-check + lint green at each phase boundary

**Chosen:** Tasks are ordered so each phase compiles, lints, and passes existing tests before the next phase starts. Specifically: (0) docs first, (1) primitives + composables added (no consumers yet — they exist in isolation), (2) views migrated one at a time to consume the new primitives, (3) folder reorg done last with a single mass `mv` + import-rewrite step.

**Rationale:** Doing the folder reorg first would force every subsequent commit to carry the import-path changes, polluting diffs and making review harder. Doing it last means everything between phases 1 and 3 lands at the current paths and the final phase is a mechanical rename. Adding primitives before consuming them keeps phase 1 pure-additive — easy to revert if something is wrong.

**Alternative considered:** Big-bang refactor in a single commit. Rejected — review is impractical, and any failing test bisect against this refactor would have nothing to bisect to.

## Risks / Trade-offs

- **[Risk]** New primitives accidentally lose accessibility behavior present in raw daisyUI markup (focus trap, `aria-labelledby`, `role="dialog"`, ESC-to-close). **Mitigation:** Each primitive's spec scenario calls out the accessibility expectation explicitly; manual verification in `wails dev` checks tab focus + ESC behavior on `AppModal` and `AppConfirmModal`; `AppTabs` preserves `role="tablist"` / `role="tab"` which the existing `SettingsView` already gets right.
- **[Risk]** `usePluginInstaller` extraction subtly changes the install-modal flow (e.g., timing of `installError` reset, which providers get auto-selected). **Mitigation:** Extract by lift-and-shift, not rewrite — the composable's first version is the same code in a different file. Smoke-test the install flow end-to-end via `wails dev` against a real marketplace before declaring the migration done.
- **[Risk]** Toast container mounted in `DefaultLayout.vue` doesn't render for views that don't use `DefaultLayout` (currently none, but future-proofing). **Mitigation:** All current routes wrap content in `DefaultLayout`; if that ever changes, mount `AppToast` at `App.vue` instead — note this in the toast composable docstring.
- **[Risk]** Folder reorg breaks tests, store imports, or auto-generated Wails bindings. **Mitigation:** Wails bindings under `frontend/wailsjs/` are not touched by this change. Vitest specs live under `frontend/src/stores/__tests__/` and import from `@/stores/*`, not from the components being moved. After the `mv` step, run `pnpm type-check` to flag any missed import; rg-rewrite any survivors.
- **[Trade-off]** Adding eight new primitive files and two composables increases the surface area of the `components/ui/` and `composables/` folders. Worth it: the per-screen complexity drops far more than the per-folder count rises, and the rules in CLAUDE.md make new contributors reach for the right tool.
- **[Trade-off]** The slot-shell `AppModal` is more flexible than a prop-driven modal but also more verbose at call sites. Acceptable: the alternative (a god-mode modal with `:fields`, `:actions` arrays) is worse on every axis once the consumer has any custom content.
- **[Behavior change, intentional]** `BrowsePluginsView`'s per-marketplace section currently writes `<details ... open>` as an unconditional attribute, which forces the section open on every render. The `AppCollapse` migration uses `:default-open="true"`, which sets only the **initial** state — the user can then collapse a section. This is a small UX improvement (collapses become user-controllable) and the spec documents it; QA should expect collapses to remember their open/closed state across re-renders within the same view session, but not across navigation.
- **[Implementation requirement]** `AppCheckbox` must support array `v-model` binding so `BrowsePluginsView`'s install modal can keep `v-model="targetProviders"` with `:value="'claude'"` / `:value="'copilot'"` on two separate checkbox instances and have the array auto-toggle. Native `<input type="checkbox">` does this when the same `v-model` array is bound across multiple checkboxes with distinct `:value`s; the wrapper must forward both `modelValue` and `value` correctly and emit `update:modelValue` with the new array.

## Migration Plan

1. Phase 0 — Update CLAUDE.md with the Frontend Design System section. Becomes the reference for everything that follows.
2. Phase 1 — Add new primitives (`AppModal`, `AppTabs`, `AppTab`, `AppInput`, `AppCheckbox`, `AppCollapse`, `AppBadge`, `AppToast`) and rewrite `AppConfirmModal` to compose `AppModal`. Add `useToast` and `usePluginInstaller` composables. No consumer changes yet. `pnpm type-check` + `pnpm lint` green at end of phase.
3. Phase 2 — Migrate views one at a time: `MarketplacesTab.vue`, `ProvidersTab.vue`, `SettingsView.vue`, `InstalledPluginsView.vue`, `BrowsePluginsView.vue`. After each view, `rg "class=\"[^\"]*(?:\\bmodal\\b|\\btabs\\b|\\balert\\b|\\binput\\b|\\bcheckbox\\b|\\bbadge\\b|\\bcollapse\\b|\\btoast\\b)" frontend/src/views/<file>` returns nothing. Mount `<AppToast />` in `DefaultLayout.vue` once during this phase.
4. Phase 3 — Folder reorg: move `PluginCard.vue`, `InstalledPluginCard.vue` to `components/plugin/`; `MarketplaceCard.vue` to `components/marketplace/`; `ProviderCard.vue` to `components/provider/`. Update all imports in one pass. `pnpm type-check` confirms no dangling imports.
5. Phase 4 — Validation: `pnpm type-check`, `pnpm lint`, `pnpm format`, `pnpm run test:unit`. Manual `wails dev` smoke: settings tabs switch, marketplace add/remove, plugin browse/select/install with install log, installed-plugin uninstall + toast notification, theme toggle. Then `openspec validate refactor-frontend-design-system --strict`.

**Rollback strategy:** Each phase is its own commit (or set of commits). Rollback is `git revert` of the phase that broke; no migrations, no data shape changes, no schema touch.

## Open Questions

- Should `useToast` use module-level singleton state (a top-level `ref([])` exported from the composable file) or `provide`/`inject` keyed at `App.vue`? Module-singleton is simpler and standard for global ephemeral UI; `provide`/`inject` is more "Vue-idiomatic" but adds setup ceremony in `App.vue`. **Tentative:** module-singleton — simpler, matches Pinia's pattern of "one instance globally," and no test currently exercises multi-root toast isolation.
- Should `AppCollapse` expose a `v-model:open` for controlled use, or always rely on the native `<details>` toggle? The current `BrowsePluginsView` always opens collapses (`open` attribute) — uncontrolled is enough today, but a controlled mode is cheap to add later. **Tentative:** uncontrolled-only in the first version, with a note in the spec scenario that controlled mode is a future addition if needed.
- Should `AppBadge` accept arbitrary `style` for the per-provider color override `PluginCard` currently does inline (`hsl(15, 62%, 59%)` for Claude, `hsl(262, 51%, 48%)` for Copilot)? Or should those colors move to a `providerColors` map exposed by the primitive? **Tentative:** keep the inline-style escape hatch on `AppBadge` (via `:style`) for now — the provider colors are domain-specific and don't belong inside a generic primitive. Document in CLAUDE.md that inline `:style` is acceptable on primitives when no semantic color slot fits.
