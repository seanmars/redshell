## Context

Claude Code and Copilot CLI both support "hooks" — JSON-configured shell commands or HTTP/MCP/prompt handlers that run at lifecycle points (`PreToolUse`, `SessionStart`, `userPromptSubmitted`, …). Their configuration models diverge in important ways:

- **Claude** is user-global, multi-source, and merges across user / project / local / plugin / managed scopes. Hook entries carry a `matcher` (literal name, `|`-list, or regex) and a handler list of typed entries (`command`, `http`, `mcp_tool`, `prompt`, `agent`).
- **Copilot CLI** is per-`cwd` and uses a single file `<cwd>/.github/hooks/copilot-cli-policy.json`. Hook entries are flat — no matcher, type is always `command`, with platform-specific `bash` / `powershell` script paths.

RedShell already manages plugin installation and reads `~/.claude/plugins/installed_plugins.json`. The directory `~/.claude/plugins/marketplaces/<marketplaceID>/plugins/<pluginID>/hooks/hooks.json` is where each installed Claude plugin can ship its own hooks (verified by enumeration on a real `~/.claude/plugins/` tree). Plugins delivered via marketplaces can therefore extend hook behavior silently — there is no built-in "show me all my hooks" surface for the user inside RedShell.

The Session History viewer (`internal/sessionhistory/`, `frontend/src/views/SessionHistoryView.vue`) is the closest existing pattern: a strictly read-only viewer with per-agent tabs, a two-pane Source-collection / Detail layout, an `AppEmptyState` for "no enabled agents", and `internal/sessionhistory/{paths,service,types,service_test}.go` as the backend layout. The Hook Viewer mirrors this template.

`openspec/specs/os-path-opener/spec.md` already defines an OS-aware "reveal file in file manager" capability. The Hook Viewer's "Open settings file" affordance reuses that capability rather than introducing a new one.

## Goals / Non-Goals

**Goals:**

- Give users a single read-only surface that shows every hook configured for each enabled agent on this machine.
- Make every hook's source path visible and openable, so users can navigate to the file that owns it.
- Enumerate plugin-bundled hooks separately so users can attribute them to the plugin that introduced them.
- Reserve a future-proof service signature so the per-workspace scope (`B-route`) can be added later without breaking the Wails binding.
- Mirror the existing Session History viewer pattern so the new page feels native to the rest of the app.

**Non-Goals:**

- Editing, enabling, disabling, or deleting hooks. This is a viewer.
- Project / cwd workspace scope (`B-route`). The signature is reserved but not wired in v1.
- Copilot cloud-agent hooks, including any GitHub API integration to read a repo's `.github/hooks/`.
- Per-hook execution history (e.g. correlating events from session JSONL back to hooks).
- Managed / org-policy hooks.
- Matcher regex evaluation or "would this match Bash?" simulation.
- Showing Copilot user-level hooks — the concept does not exist.

## Decisions

### Decision 1: by-source > merged-with-source-tags

**Chosen**: Group rendered hooks by Source first, then by Event, in a single collapsible tree pane.

**Alternatives considered**:

- *Merged view* (Claude's `/hooks` style — one list per event with a "from X" tag per row). Rejected because it hides the structural fact that plugin hooks are bundled with a plugin: users care that "this PreToolUse hook came from the `hookify` plugin" more than "all my PreToolUse hooks regardless of where they live".
- *Source tabs* (one tab per Source). Rejected because the number of plugin sources can grow large and flat-tabbing them clutters the header. Collapsible groups in a single pane scale better.

### Decision 2: Plugin hooks scanned via `installed_plugins.json` `installPath`, no directory walk

**Chosen**: Enumerate Claude plugins by reading `~/.claude/plugins/installed_plugins.json` (v2 schema: `{ "version": 2, "plugins": { "<pluginID>@<marketplaceID>": [ { "scope": ..., "installPath": ..., "version": ... }, ... ] } }`). For each entry, load `<installPath>/hooks/hooks.json`. The viewer treats `installPath` as the runtime contract and does not hardcode any subdirectory of `~/.claude/plugins/`.

**Why this differs from the original spike conclusion**: a verification of `~/.claude/plugins/installed_plugins.json` shows that `installPath` for active installs points into `~/.claude/plugins/cache/<marketplaceID>/<pluginID>/<version>/`. The earlier "cache/ is just download staging" assumption was wrong — `cache/` IS where Claude's runtime plugin install lives, and the path under `marketplaces/` is the marketplace clone holding the marketplace manifest, not the runtime hook source. This decision rewrites the original Decision 2 to match that reality.

**Multi-entry keys**: `installed_plugins.json` v2 allows multiple entries per plugin key (one per scope, e.g. user + project). Each entry becomes its own Plugin source group in the viewer; the label disambiguates by scope when more than one entry is present.

**Alternatives considered**:

- *Walk `~/.claude/plugins/marketplaces/`* directly. Rejected — those files are the marketplace manifest's hook templates, not the runtime install. Loading them would show hooks that are not actually wired up.
- *Walk `~/.claude/plugins/cache/`* and infer plugins by directory layout. Rejected — `cache/` legitimately holds older versions for the same plugin (e.g. `superpowers/5.0.7/` and `superpowers/5.1.0/`); only the version named by `installPath` is active.
- *Hardcoding `~/.claude/plugins/cache/<m>/<p>/<v>/hooks/hooks.json`*. Rejected — `installPath` is the contract Claude itself maintains; hardcoding the cache layout couples our viewer to internal Claude paths that may change.

### Decision 3: ListOpts struct over scalar arguments

**Chosen**: `Service.ListHooks(agentID string, opts ListOpts) (Listing, error)` where `ListOpts` is `{ Workspace string }` in v1.

**Rationale**: Adding a workspace argument later as a positional parameter would change the Wails-generated TypeScript signature and force every frontend caller to update. With `ListOpts`, the frontend passes an object literal `{}` today and `{ Workspace: "..." }` later, so the binding stays stable. The Go service ignores non-empty `Workspace` in v1 (no error, no warning) — the field exists purely as a forward-compat anchor.

**Alternatives considered**:

- Add `Workspace` later when needed. Rejected because the Wails binding regenerates `frontend/wailsjs/go/app/HooksApp.d.ts` and the change becomes a breaking surface for callers.
- Variadic options pattern (Go idiomatic). Rejected because Wails reflection does not bind variadic Go functions cleanly to TypeScript.

### Decision 4: Copilot parser implemented in v1, but invoked only with a non-empty Workspace

**Chosen**: Ship `parser_copilot.go` immediately. The Copilot tab in v1 always renders an empty state because `ListOpts.Workspace` is unused, but the parser is unit-tested via fixtures so when `B-route` is added later only the wiring needs work.

**Rationale**: The parser logic is small (~50 lines) and writing it once now with fixtures avoids a second round of "rediscover the schema" later. The risk of the schema drifting in the meantime is bounded by the parser's tolerance to unknown fields.

### Decision 5: Dedup is informational, not structural

**Chosen**: Render every hook from every source, even when the same `command` string appears in multiple sources. The detail pane shows a small chip ("appears in 2 sources") when a duplicate is detected for the same agent.

**Rationale**: Claude's runtime dedupes on `(command string, http URL)` before execution. Surfacing that in the viewer would either require collapsing rows (loses source attribution) or strikethroughing duplicates (visually noisy and incorrect for partial-overlap cases like same command, different `if` filter). A passive indicator in the detail pane is honest about what the user sees on disk.

### Decision 6: `disableAllHooks` shown as banner, list still rendered

**Chosen**: When any Claude source has `"disableAllHooks": true` at top level, render a yellow banner at the top of the Claude tab ("Hooks are globally disabled by <source path>") and continue to render the full list normally.

**Rationale**: Hiding or graying out the list would mislead users who turned the flag on temporarily and want to verify what would run when they turn it back off. The banner makes the runtime state visible without erasing the configuration state.

### Decision 7: "Open settings file" delegates to `os-path-opener`

**Chosen**: The detail pane's "Open settings file" button calls the existing `os-path-opener` capability (which already abstracts Windows / macOS / Linux file-manager invocation). No new OS-shelling helper is introduced in `internal/hooks/`.

**Rationale**: Keeps OS-specific code in one place. The capability exists precisely for this kind of "reveal file" affordance.

### Decision 8: Plugin source label format

**Chosen**: Plugin source label is `Plugin: <pluginID>@<marketplaceID>` (matches the existing `installed_plugins.json` key convention), and the source path shown in the detail pane is the full absolute filesystem path.

**Rationale**: Two marketplaces can ship a plugin with the same `pluginID` (e.g. `explanatory-output-style` in both `claude-plugins-official` and `claude-code-plugins` on the same machine). Including the marketplace disambiguates without forcing the user to expand the path.

## Risks / Trade-offs

- **[Plugin scan misses non-marketplace install paths]** If a plugin installs hooks somewhere outside `marketplaces/<m>/plugins/<p>/hooks/hooks.json` (custom layout, manual install), the viewer will not see them. → Mitigation: scope is explicitly "plugins listed in `installed_plugins.json` with the marketplaces-tree layout"; the spec and design call this out so users with bespoke setups know what they will and will not see.
- **[Cache staleness]** `~/.claude/plugins/cache/` contains downloaded versions that may diverge from the active install. → Mitigation: cache is hard-excluded by the path-resolution rule and a unit test asserts it.
- **[`disableAllHooks` from a source we did not load]** A managed-policy or project source could carry the flag without us reading it. → Mitigation: v1 only honors the flag from sources we load (User / Local / Plugin). The banner copy reads "set in <source path>" so it is unambiguous which file is responsible.
- **[Schema drift]** The Claude hook schema is large and growing (new event names, new handler types). → Mitigation: parser tolerates unknown fields by passing through into the raw JSON view. Unknown handler `type` values render with a placeholder badge but never abort.
- **[Large hook count]** Active Claude users can register 50+ hooks across all sources. → Mitigation: the tree is collapsible by Source and by Event; only the selected hook row's detail is fully rendered. The list itself is a flat array client-side, so 50–500 entries are well within Vue's reactive overhead.
- **[Future B-route surface change]** Reserving `ListOpts.Workspace` is forward-compat only; the actual workspace-selection UX (manage list, watch invalidation, per-workspace settings persistence) is a separate design conversation. → Mitigation: the field is documented as reserved in the spec, and the `Service` constructor pattern leaves room for a `WatchedWorkspaces` collaborator to be injected without breaking callers.

## Migration Plan

This is purely additive — no existing behavior, route, or data file changes. Deploy is a normal `wails build`. No backfill, no data migration. Rollback is a code revert; no on-disk artifacts are written by the viewer.

## Open Questions

- Should the "Open settings file" button on a Plugin source row open the plugin's `hooks.json`, or the plugin's containing folder? Current spec says the file itself; revisit if user feedback prefers folder reveal.
- Future B-route: how is the watched-workspaces list persisted (settings.json key vs. a new file under `~/.redshell/`)? Out of scope for this change but flagged for the follow-up design.
