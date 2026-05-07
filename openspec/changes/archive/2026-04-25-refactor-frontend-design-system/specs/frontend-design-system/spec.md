## ADDED Requirements

### Requirement: DaisyUI component classes with `App*` primitives SHALL only be used inside those primitives

The frontend codebase SHALL NOT use the **inventoried** daisyUI component classes — `btn`, `card`, `alert`, `modal`, `tabs`, `tab`, `input`, `checkbox`, `collapse`, `badge`, `loading`, `toast` (and any of their `<base>-<modifier>` variants such as `btn-primary`, `tabs-bordered`, `alert-warning`, `loading-spinner`) — anywhere outside `frontend/src/components/ui/App*.vue`. Tailwind utility classes (`flex`, `grid`, `gap-*`, `p-*`, `m-*`, `space-*`, `max-w-*`, `min-w-*`, `truncate`, `text-*`, `font-*`, `bg-*`, `border-*`, `rounded-*`, `shadow-*`, `cursor-*`, `opacity-*`, `transition-*`, responsive prefixes, hover/focus modifiers) MAY be used freely in views, layouts, and domain components.

The following daisyUI classes are **documented exceptions** allowed inline outside primitives:

- `menu` and `menu-*` modifiers — only inside `frontend/src/layouts/` (sidebar nav).
- `swap` and `swap-*` modifiers — only inside `frontend/src/components/ThemeToggle.vue` (icon-toggle animation).
- `btn-ghost btn-circle` icon-shell — only when applied to a `<RouterLink>`, `<label>`, or other non-`<button>` host where `AppButton` cannot be substituted. The two known sites are the sidebar settings link in `DefaultLayout.vue` and the theme-toggle label in `ThemeToggle.vue`.

DaisyUI classes the codebase does not currently use (`dropdown`, `avatar`, `select`, `textarea`, `radio`, `toggle`, `range`, `footer`) SHALL NOT be introduced inline; if a view needs one, a primitive is added at that point and the class joins the inventoried list.

#### Scenario: View grep returns no inventoried daisyUI classes

- **WHEN** running `rg "class=\"[^\"]*\\b(btn|card|alert|modal|tabs|tab|input|checkbox|collapse|badge|loading|toast|dropdown|avatar|select|textarea|radio|toggle|range|footer)\\b" frontend/src/views frontend/src/components/settings frontend/src/components/plugin frontend/src/components/marketplace frontend/src/components/provider`
- **THEN** the command exits with no matches (the only legitimate occurrences are inside `frontend/src/components/ui/`)
- **AND** `frontend/src/layouts/` and `frontend/src/components/ThemeToggle.vue` are deliberately excluded from this scan because they hold the documented exceptions

#### Scenario: Documented exceptions in layouts are accepted

- **WHEN** running `rg "class=\"[^\"]*\\b(menu|swap)\\b" frontend/src/layouts frontend/src/components/ThemeToggle.vue`
- **THEN** any matches found correspond to one of the documented exceptions above (sidebar nav `menu`, theme-toggle `swap`, or `btn-circle` icon-shell on non-`<button>` hosts)

#### Scenario: Tailwind utilities remain free in views

- **WHEN** a view applies `class="flex items-center gap-3 max-w-3xl mx-auto p-6"`
- **THEN** no convention is violated, because every class is a Tailwind utility, not an inventoried daisyUI component class

### Requirement: All Vue single-file components SHALL use `<script setup lang="ts">`

Every `.vue` file under `frontend/src/` SHALL declare its component logic with `<script setup lang="ts">`. Options-API components (`export default { data, methods, ... }`) and JavaScript-only `<script>` blocks SHALL NOT be added or kept. Two `<script>` blocks (one `<script lang="ts">` for module-level exports, one `<script setup lang="ts">` for component logic) are permitted only when the component must export a const for type purposes.

#### Scenario: Codebase grep finds no Options API

- **WHEN** running `rg "export default \\{" frontend/src --glob "*.vue"`
- **THEN** the command returns no matches

#### Scenario: Every component declares setup explicitly

- **WHEN** running `rg "<script" frontend/src --glob "*.vue" -l` to list component files
- **AND** for each file checking that at least one `<script setup` block is present
- **THEN** every file matches

### Requirement: Shared view logic SHALL live in composables under `src/composables/`

Logic that is reused across two or more views, OR a coherent state machine that exceeds approximately 80 lines of `<script setup>` in a single view, SHALL be extracted into a composable function exported from a file under `frontend/src/composables/`. Composable filenames SHALL begin with `use` (e.g., `useToast.ts`, `usePluginInstaller.ts`). Each composable SHALL export a single function and SHALL NOT depend on the host component's `this` context.

#### Scenario: Toast notifications use a shared composable

- **WHEN** any view needs to show a transient notification
- **THEN** it imports `useToast` from `@/composables/useToast` and calls `push({ type, message })`
- **AND** it does not maintain its own `setTimeout` or `notification` ref

#### Scenario: BrowsePluginsView install flow is in a composable

- **WHEN** reading `frontend/src/views/BrowsePluginsView.vue`
- **THEN** the install-flow state (`selectedMergedPlugins`, `availableProviders`, `targetProviders`, install-modal handlers) is consumed from `usePluginInstaller`
- **AND** the view's `<script setup>` does not redefine that state inline

### Requirement: The frontend SHALL provide a complete primitive vocabulary in `components/ui/`

The folder `frontend/src/components/ui/` SHALL contain `App*` primitive components that wrap each inventoried daisyUI component class. At minimum, the following primitives SHALL exist: `AppAlert`, `AppBadge`, `AppButton`, `AppCard`, `AppCheckbox`, `AppCollapse`, `AppConfirmModal`, `AppInput`, `AppModal`, `AppSpinner`, `AppTab`, `AppTabs`, `AppToast`. Each primitive SHALL expose its variants (color, size, state) through props rather than requiring callers to pass daisyUI modifier class strings. `AppCheckbox` SHALL additionally support array `v-model` binding (i.e., when bound to an array via `v-model` with a `:value` prop, toggling the checkbox adds or removes that value from the array, matching native `<input type="checkbox">` behavior under Vue's `v-model`).

#### Scenario: All required primitives exist

- **WHEN** listing files under `frontend/src/components/ui/`
- **THEN** the listing contains `AppAlert.vue`, `AppBadge.vue`, `AppButton.vue`, `AppCard.vue`, `AppCheckbox.vue`, `AppCollapse.vue`, `AppConfirmModal.vue`, `AppInput.vue`, `AppModal.vue`, `AppSpinner.vue`, `AppTab.vue`, `AppTabs.vue`, `AppToast.vue`

#### Scenario: Variants come from props, not pass-through classes

- **WHEN** a caller wants a primary button
- **THEN** it writes `<AppButton variant="primary">` rather than `<AppButton class="btn-primary">`

#### Scenario: AppCheckbox supports array v-model

- **WHEN** two `<AppCheckbox>` instances are bound to the same array via `v-model="targetProviders"` with `:value="'claude'"` and `:value="'copilot'"` respectively
- **THEN** toggling the first checkbox adds or removes `'claude'` from `targetProviders`
- **AND** toggling the second checkbox adds or removes `'copilot'` from `targetProviders`

### Requirement: `AppModal` SHALL be a slot-shell primitive

`AppModal` SHALL accept the props `isOpen` (boolean, required), `size` (`'sm' | 'md' | 'lg'`, default `'md'`), and emit a `close` event when the user dismisses the modal via the backdrop, the Escape key, or any explicit close affordance the consumer wires. It SHALL render named slots `header` and `actions`, and a default slot for body content. It SHALL NOT accept props for title text, button labels, form fields, or other body shape — those belong in slots so each consumer composes its own modal body.

#### Scenario: AppModal renders the body slot when open

- **WHEN** `<AppModal :is-open="true">Hello</AppModal>` is mounted
- **THEN** the rendered DOM contains a `<dialog>` element with the `modal` class set, the `open` attribute, and the text "Hello" in the modal body

#### Scenario: AppModal emits close on Escape

- **WHEN** the modal is open and the user presses Escape
- **THEN** the component emits a `close` event
- **AND** the consumer is responsible for setting `isOpen` to `false` in response

#### Scenario: AppModal does not accept title or footer props

- **WHEN** reading the `defineProps` declaration of `AppModal`
- **THEN** the prop list contains exactly `isOpen` and `size` and no `title`, `confirmLabel`, `cancelLabel`, `fields`, or `actions` array prop

### Requirement: `AppTabs` SHALL support `v-model:active` and child `AppTab` declarations

`AppTabs` SHALL accept `v-model:active` (string id of the active tab) and a `variant` prop (`'bordered' | 'boxed'`, default `'bordered'`). Tab panes SHALL be declared as child `<AppTab id="..." label="...">` components inside the `AppTabs` default slot; only the active tab's default slot content SHALL be rendered. `AppTabs` SHALL render a `role="tablist"` container and each `AppTab` SHALL render with `role="tab"` and `aria-selected` reflecting the active state.

#### Scenario: Active tab content is rendered

- **WHEN** `<AppTabs v-model:active="'one'"><AppTab id="one">A</AppTab><AppTab id="two">B</AppTab></AppTabs>` is mounted
- **THEN** the rendered DOM contains the text "A" and does not contain the text "B"

#### Scenario: Tab change updates v-model

- **WHEN** the user clicks the tab labelled "two"
- **THEN** the `update:active` event fires with payload `'two'`

#### Scenario: ARIA attributes are correct

- **WHEN** rendering with `active="one"`
- **THEN** the tablist contains a `role="tab"` element with `aria-selected="true"` for tab `one` and `aria-selected="false"` for tab `two`

### Requirement: `useToast` SHALL be the only mechanism for transient notifications

The composable `useToast()` SHALL return an object with `toasts` (a readonly ref to an array of active toasts), `push(toast)` (queues a toast and schedules auto-dismiss), and `dismiss(id)` (removes a toast immediately). Each toast SHALL have a `type` (`'success' | 'error' | 'info' | 'warning'`), a `message` string, an internally generated `id`, and a default auto-dismiss timeout of 3000 ms (overridable per toast via a `duration` field). Toast state SHALL be a module-level singleton so multiple call sites share one queue. The `AppToast` component SHALL be mounted exactly once in `DefaultLayout.vue` and SHALL render every toast in the queue.

#### Scenario: Pushing a toast renders it

- **WHEN** a view calls `useToast().push({ type: 'success', message: 'done' })`
- **THEN** the `<AppToast />` mounted in the layout shows a success toast with the text "done"

#### Scenario: Toast auto-dismisses after timeout

- **WHEN** a toast is pushed without an explicit `duration`
- **THEN** after 3000 ms the toast is removed from the queue automatically

#### Scenario: Multiple toasts stack

- **WHEN** two toasts are pushed in quick succession
- **THEN** both are visible simultaneously until each individually times out
- **AND** the existing `setTimeout`-based single-slot pattern in `InstalledPluginsView.vue` is no longer used

### Requirement: `components/` folder SHALL separate primitives from domain components

`frontend/src/components/ui/` SHALL contain only primitive `App*` components — components that do not import from `@/stores/*`, do not call Wails bindings, and contain no business logic. Domain-specific components (cards or widgets bound to a specific feature) SHALL live in per-domain folders: `components/plugin/`, `components/marketplace/`, `components/provider/`, and the existing `components/settings/` for tab panes. Components that cut across multiple domains and are not primitives MAY live at the `components/` root.

#### Scenario: ui folder contains only primitives

- **WHEN** listing files under `frontend/src/components/ui/`
- **THEN** every file matches the pattern `App*.vue`
- **AND** none of them import from `@/stores/*`

#### Scenario: Domain cards live in domain folders

- **WHEN** locating `PluginCard.vue` and `InstalledPluginCard.vue`
- **THEN** they reside under `frontend/src/components/plugin/`
- **AND** `MarketplaceCard.vue` resides under `frontend/src/components/marketplace/`
- **AND** `ProviderCard.vue` resides under `frontend/src/components/provider/`

### Requirement: CLAUDE.md SHALL document the Frontend Design System

The repository-root `CLAUDE.md` SHALL contain a section titled "Frontend Design System" (or equivalent) that captures: (a) the daisyUI-component-class rule and the daisyUI/Tailwind boundary, (b) the list of primitives in `components/ui/`, (c) the rule that shared logic moves into composables and that all components use `<script setup lang="ts">`, (d) the folder layout for primitives vs. domain components. This section SHALL be the authoritative reference for future frontend work.

#### Scenario: Frontend Design System section exists

- **WHEN** reading `CLAUDE.md` from the repository root
- **THEN** the file contains a heading whose text begins with "Frontend Design System"
- **AND** the section lists the daisyUI component classes that must be wrapped
- **AND** the section names the primitives that exist in `components/ui/`
- **AND** the section names the per-domain folders for domain components
