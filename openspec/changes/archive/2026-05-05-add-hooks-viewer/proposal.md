## Why

Users configure Claude Code and Copilot CLI hooks across multiple files (user settings, local settings, plugin-bundled hooks) and currently have no way inside RedShell to see what is actually wired up — installed plugins can silently add their own hooks and there is no consolidated view. A read-only Hook Viewer turns these scattered files into a single inspectable surface so users can audit, understand, and trust their hook configuration before letting it run.

## What Changes

- Add a top-level `/hooks` page reachable from the sidebar, sitting between Sessions and Installed.
- Render a per-agent tab control identical to the Session History page (tabs only when more than one agent is enabled, single-agent view otherwise, empty state when zero).
- Render a two-pane layout: left pane is a collapsible tree grouped Source → Event → Hook; right pane is the selected hook's detail view.
- For Claude: read user-level (`~/.claude/settings.json`), local (`~/.claude/settings.local.json`), and per-plugin (`~/.claude/plugins/marketplaces/<m>/plugins/<p>/hooks/hooks.json`) sources. The `~/.claude/plugins/cache/` tree and any `.git/hooks/` directories are explicitly excluded.
- For Copilot: implement the parser for `.github/hooks/copilot-cli-policy.json` but show an empty state in v1 explaining that Copilot CLI hooks are project-scoped and that workspace selection is a future enhancement.
- Surface a `disableAllHooks` warning banner on Claude tabs when the flag is true.
- Detail pane shows the full absolute source path (not truncated), the resolved fields per hook type, and the raw JSON pretty-printed read-only.
- "Open settings file" affordance reveals the source file in the OS file manager via the existing `os-path-opener` capability.
- Service interface accepts a `ListOpts` struct with a reserved `Workspace` field so the future per-workspace scope can be added without breaking the Wails binding.
- Strictly read-only: no agent CLI invocation, no writes under `~/.claude` or `~/.copilot`.

## Capabilities

### New Capabilities

- `hooks-viewer`: Read-only viewer that lists hooks configured for each enabled agent, grouped by source then by event, with a detail pane for each hook entry. Covers source discovery rules, parser shapes for Claude and Copilot, plugin scanning, source ordering and visibility, and the read-only contract.

### Modified Capabilities

<!-- None. The hooks viewer reads files already owned by the agent CLIs and the existing `os-path-opener` capability without changing their requirements. -->

## Impact

- **New backend package**: `internal/hooks/` (types, paths, `parser_claude.go`, `parser_copilot.go`, service, fixtures, tests). Mirrors the `internal/sessionhistory/` layout.
- **New Wails wrapper**: `app/hooks.go`, bound from `main.go` alongside the existing `app.*App` wrappers, regenerating `frontend/wailsjs/go/app/HooksApp.*`.
- **New frontend route**: `/hooks` lazy-loaded from `router/index.ts` under the same setup-guard rules as the other routes.
- **Sidebar**: a new "Hooks" entry inserted between Sessions and Installed in `frontend/src/layouts/DefaultLayout.vue`.
- **New frontend modules**: `views/HooksView.vue`, `stores/hooks.ts`, and `components/hooks/{HookList,HookDetail,HookSourceBadge}.vue`. UI uses existing `App*` primitives only.
- **Reuse**: depends on the existing `os-path-opener` capability for the "Open settings file" action and on `internal/plugin`'s knowledge of `~/.claude/plugins/installed_plugins.json` for the plugin enumeration contract.
- **No breaking changes**: no existing exported API, route, or sidebar entry is renamed or removed.
- **Out of scope (non-goals)**: editing/enabling/disabling hooks, Copilot cloud-agent hooks via the GitHub API, project/cwd workspace scope (reserved via `ListOpts.Workspace` but unused), Copilot user-level hooks (does not exist), per-hook execution history, managed/policy-source hooks, matcher regex preview/simulation.
