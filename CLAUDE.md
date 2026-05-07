# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RedShell (window title: "RedShell", output binary: `redshell`) is a Wails v2 desktop application that manages plugin marketplaces for AI coding agents — currently Claude Code (`claude` CLI) and GitHub Copilot (`copilot` CLI). The Go backend owns the marketplace registry and shells out to `git` and the agent CLIs; the Vue 3 frontend renders a four-screen flow (Agents → Marketplaces → Browse → Installed). On Windows, a system tray icon keeps the app resident; the close button consults a persisted `closeBehavior` preference (`exit` vs. `minimize-to-tray`) collected via a one-time prompt.

## Common Commands

Root (Wails build orchestrator — requires `wails` CLI, Go 1.23+, pnpm):

```sh
wails dev          # run dev app with hot-reload; frontend served via vite on auto port
wails build        # produce build/bin/redshell(.exe); frontend is `pnpm run build` then embedded via //go:embed
go test ./...      # run backend unit tests (no Wails runtime required)
go test ./internal/plugin -run TestFetchAll_CacheMiss    # single test
go run ./cmd/cli   # debug utility: print ~/.redshell/marketplace.json as JSON
```

Frontend (`cd frontend`):

```sh
pnpm install
pnpm run dev          # standalone Vite dev (usually driven by `wails dev` instead)
pnpm run build        # vue-tsc --build + vite build; wails dev/build invokes this
pnpm run test:unit    # vitest (watch mode)
pnpm run lint         # oxlint --fix, then eslint --fix --cache
pnpm run format       # prettier on src/
```

## Architecture

### Wails binding boundary

`main.go` constructs three services and three thin `app.*App` wrappers, then binds the wrappers via `options.App.Bind`. Wails scans exported methods on the bound structs and generates TypeScript bindings into `frontend/wailsjs/go/app/` (do not hand-edit — these are regenerated). Frontend code imports them like:

```ts
import { FetchAll, Install } from '@wailsjs/go/app/PluginApp';
import { EventsOn } from '@wailsjs/runtime/runtime';
```

The `@wailsjs/*` path alias is configured in `frontend/vite.config.ts` and `frontend/tsconfig.app.json`. Use it instead of relative `../../wailsjs/...` imports — the alias depth-independent and stays correct when files move between folders.

Rule of thumb: `internal/<domain>/service.go` holds all logic and is unit-testable without a Wails context. `app/<domain>.go` is the only place that may hold `context.Context` and call `runtime.EventsEmit` — it forwards logs from long-running ops (`plugin.Install`) into the frontend via `plugin:install-log` events.

### Domain layering

- `internal/agent/` — enumerates the two supported agents. Detects `~/.claude` and `~/.copilot` for the `Configured` flag. No I/O beyond `os.Stat`.
- `internal/marketplace/` — owns the registry at `~/.redshell/marketplace.json` and shallow-clone caches under `~/.redshell/.cache/<CacheDirName(id)>/`. `Add` normalizes git URLs (accepts SSH and bare `host/owner/repo`), computes an ID of `<host>::<group>@<repo>`, clones once (`--depth=1`), and reads each agent's manifest filename from `AgentMarketplaceFiles` to populate per-agent display names. `Refresh` does `git fetch --depth=1` + `git reset --hard FETCH_HEAD`. A per-ID mutex map (`cacheMu`) serializes cache mutations.
- `internal/plugin/` — depends on marketplace + agent. `FetchAll` iterates every registered marketplace × agent, reads the cached manifest, and emits `MarketplacePlugin` rows. `Install` first calls `EnsureMarketplace` (which reads the agent's own config to decide if `<agent> plugin marketplace add <url>.git` is needed) then `<agent> plugin install <name>@<marketplaceName>`. `ListInstalled` reads the agent-owned config directly rather than tracking state itself — Claude from `~/.claude/plugins/installed_plugins.json`, Copilot from `~/.copilot/config.json`.

### Agent-specific paths

`AgentMarketplaceFiles` in `internal/marketplace/service.go` is the single source of truth:

| Agent | In-repo manifest | Local registry | Installed list |
|---|---|---|---|
| claude  | `.claude-plugin/marketplace.json` | `~/.claude/plugins/known_marketplaces.json` | `~/.claude/plugins/installed_plugins.json` |
| copilot | `.github/plugin/marketplace.json`  | `~/.copilot/config.json` (`marketplaces` key) | `~/.copilot/config.json` (`installed_plugins` key) |

Adding a third agent means extending `AgentMarketplaceFiles`, the switch in `plugin.EnsureMarketplace`, and the `ListInstalled` dispatch.

### Shell-level preferences

`~/.redshell/preferences.json` (managed by `internal/preferences`) holds shell-level UX state, currently just `closeBehavior` (`unset` | `exit` | `minimize-to-tray`). It is intentionally separate from `~/.redshell/settings.json` (agent setup) so the two evolve independently. The Windows tray (`internal/tray`, behind a `windows` build tag with a no-op stub elsewhere) reads / writes this preference and offers a checkable "Close button minimizes to tray" item; the close-intercept (`OnBeforeClose` in `main.go`) consults the same preference and emits the `tray:close-behavior-prompt` runtime event when it is still `unset`.

### Frontend state shape

Four routes, lazy-loaded from `router/index.ts`: `/agents` (default) → `/marketplaces` → `/browse` → `/installed`.

Stores (Pinia setup-style):

- `stores/plugin.ts` — calls `FetchAll`/`Install`/`ListInstalled`/`Uninstall` and subscribes to the `plugin:install-log` event. The non-obvious piece is `mergedPlugins`: the backend returns one `MarketplacePlugin` row per (marketplace × agent) pair, and this computed regroups them by `${name}@${marketplace}` so a single card shows agent badges for each agent that exposes it. `errorsByMarketplace` parses backend errors shaped as `[<marketplaceID>/<agent>] <msg>` back into structured buckets — preserve that format in the Go layer when adding new error paths.
- `stores/marketplace.ts`, `stores/agent.ts` — straight proxies to their Wails bindings.
- `stores/theme.ts` — daisyUI light/dark toggle, persisted to localStorage.

All views wrap their content in `layouts/DefaultLayout.vue`. UI conventions are defined below.

## Frontend Design System

The frontend stack is Vue 3 (Composition API) + Tailwind 4 + daisyUI 5. The rules below are authoritative for every `.vue` file under `frontend/src/`.

### DaisyUI class boundary — what must be wrapped

A daisyUI **component** class that has an `App*` primitive in `components/ui/` SHALL only appear inside that primitive. The **inventoried** classes (and their `<base>-<modifier>` variants such as `btn-primary`, `tabs-bordered`, `alert-warning`, `loading-spinner`):

| Class | Wrapping primitive |
|---|---|
| `btn` | `AppButton` |
| `card` | `AppCard` |
| `alert` | `AppAlert` |
| `modal` | `AppModal` (slot-shell) |
| `tabs`, `tab` | `AppTabs` + `AppTab` |
| `input` | `AppInput` |
| `checkbox` | `AppCheckbox` (supports array `v-model`) |
| `collapse` | `AppCollapse` |
| `badge` | `AppBadge` |
| `loading` | `AppSpinner` |
| `toast` | `AppToast` (queue rendered via `useToast()`) |

Tailwind utility classes (`flex`, `grid`, `gap-*`, `p-*`, `m-*`, `space-*`, `max-w-*`, `min-w-*`, `truncate`, `text-*`, `font-*`, `bg-*`, `border-*`, `rounded-*`, `shadow-*`, `cursor-*`, `opacity-*`, `transition-*`, responsive prefixes, hover/focus modifiers) MAY be used freely in views, layouts, and domain components.

### Documented exceptions

The following daisyUI classes MAY appear inline outside primitives, but only at the listed locations:

- `menu` and `menu-*` modifiers — `frontend/src/layouts/DefaultLayout.vue` only (sidebar nav `<ul>`).
- `swap` and `swap-*` modifiers — `frontend/src/components/ThemeToggle.vue` only (icon-toggle animation).
- `btn-ghost btn-circle` icon-shell — only when the host element is a `<RouterLink>`, `<label>`, or other non-`<button>` element where `AppButton` (which renders `<button>`) cannot be substituted. Two known sites: sidebar settings link in `DefaultLayout.vue`, theme-toggle label in `ThemeToggle.vue`.

DaisyUI classes the codebase does not currently use (`dropdown`, `avatar`, `select`, `textarea`, `radio`, `toggle`, `range`, `footer`) SHALL NOT be introduced inline — add a primitive when a view needs one.

### Composition API and composables

- Every `.vue` file under `frontend/src/` SHALL use `<script setup lang="ts">`. Options API and JS-only `<script>` blocks are not permitted.
- Logic shared across two or more views — or a coherent state machine that exceeds ~80 lines of `<script setup>` in a single view — SHALL be extracted to a composable under `frontend/src/composables/`. Filenames start with `use` (`useToast.ts`, `usePluginInstaller.ts`).
- Existing composables: `useConfirm` (imperative confirm modal), `usePageTitle` (browser tab title), `useToast` (transient notifications, module-singleton queue), `usePluginInstaller` (BrowsePluginsView install flow).

### Component folder layout

```
frontend/src/components/
  ui/                     # App* primitives only — no business logic, no store imports
  plugin/                 # PluginCard, InstalledPluginCard
  marketplace/            # MarketplaceCard
  agent/                  # AgentCard
  settings/               # tab panes (MarketplacesTab, AgentsTab)
  ThemeToggle.vue         # cross-cutting; documented swap/btn-circle exception
```

Domain components import primitives from `@/components/ui/`; primitives never import from `@/stores/*` or `@/composables/`.

### Mechanical leak check (required for frontend PRs)

```sh
rg "class=\"[^\"]*\b(btn|card|alert|modal|tabs|tab|input|checkbox|collapse|badge|loading|toast|dropdown|avatar|select|textarea|radio|toggle|range|footer)\b" \
  frontend/src/views \
  frontend/src/components/settings \
  frontend/src/components/plugin \
  frontend/src/components/marketplace \
  frontend/src/components/agent \
  frontend/src/components/hooks
```

Zero matches is the contract. `frontend/src/layouts/` and `frontend/src/components/ThemeToggle.vue` are deliberately excluded — they hold the documented exceptions and should be spot-checked by hand.

## Testing Notes

- Go tests for `internal/plugin` use `NewServiceWithCacheRoot` + `seedCache` helpers to avoid touching real `~/.redshell` or running `git`. Mirror that pattern — never call `marketplace.NewService()` (which resolves `os.UserHomeDir`) from a test.
- Fixtures are in `internal/plugin/testdata/`. When adding manifest-shape tests, add a JSON fixture there rather than building strings inline.
- Frontend tests live in `frontend/src/stores/__tests__/` (Vitest + jsdom). Mock `../../wailsjs/go/app/*` and `../../wailsjs/runtime/runtime` at the top of each test — the real modules only exist after a Wails build.

## Spec / Plan Workflow

Two parallel documentation systems coexist here:

- `openspec/` — spec-driven workflow with `specs/<feature>/spec.md` for current behavior and `changes/archive/` for historical proposals. Consumed by the `openspec-*` skills (`openspec-propose`, `openspec-apply-change`, `openspec-archive-change`, `openspec-explore`) and the `opsx:*` slash commands.
- `docs/superpowers/` — plan + spec pairs (e.g. `plans/2026-04-24-plugins-ux-redesign.md`) driven by the `superpowers:executing-plans` sub-skill pattern. These are the implementation roadmaps.

If the user's request maps to an active plan under `docs/superpowers/plans/`, follow that plan's task order rather than inventing your own.

## Project Conventions (from user global rules)

- Use half-width punctuation only (ASCII `,` `.` `;` `:` `!` `?` `(` `)`) — never full-width variants, in code or docs.
- Never use `--no-verify` to skip commit hooks, and never disable tests to make a failure go away.
- Documentation / README content defaults to Traditional Chinese (zh-TW); code identifiers and technical terms stay in English.
- Prefer `rg` for text, `fd` for files, `ast-grep` for syntax-aware code search, `jq`/`yq` for JSON/YAML.

## Rules for Changes

- For Go code, always run `go fmt` and `go vet` on changed files.
- For frontend code, run `pnpm format` and `pnpm lint` and `pnpm type-check` on changed files.