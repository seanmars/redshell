## 1. Backend types and paths

- [x] 1.1 Create `internal/hooks/types.go` with `Source`, `Hook`, `HookGroup`, `Listing`, and `ListOpts` structs (fields per design.md decision 3 and 8)
- [x] 1.2 Create `internal/hooks/paths.go` with helpers that resolve `~/.claude/settings.json`, `~/.claude/settings.local.json`, and the plugin hook path template `~/.claude/plugins/marketplaces/<m>/plugins/<p>/hooks/hooks.json`, plus the Copilot path `<workspace>/.github/hooks/copilot-cli-policy.json` (consumer ignores when Workspace is empty in v1)
- [x] 1.3 Add `paths_test.go` covering home expansion, missing-file results, and that `~/.claude/plugins/cache/` and `.git/hooks/` segments are explicitly rejected by the path resolver

## 2. Claude parser

- [x] 2.1 Create `internal/hooks/parser_claude.go` that flattens `{ "hooks": { "<Event>": [{ "matcher": ..., "hooks": [...]}]}}` into `[]Hook`, preserving the original entry as `Raw`
- [x] 2.2 Detect top-level `disableAllHooks: true` and report it on `Listing` (per-source, with the source path)
- [x] 2.3 Tolerate unknown handler `type` values and unknown fields without dropping entries (matches "Streaming reads tolerate unknown handler types and unknown fields" requirement)
- [x] 2.4 Add `parser_claude_test.go` with fixtures under `internal/hooks/testdata/claude/` covering: command/http/mcp_tool/prompt/agent handlers, missing matcher, regex matcher, multiple matcher groups, `disableAllHooks` true and absent, malformed JSON

## 3. Copilot parser (implemented but not invoked in v1)

- [x] 3.1 Create `internal/hooks/parser_copilot.go` that flattens `{ "version": 1, "hooks": { "<event>": [...] } }` into `[]Hook`, mapping `bash`/`powershell`/`cwd`/`timeoutSec`/`comment` into the shared `Hook` struct and leaving `Matcher` empty
- [x] 3.2 Add `parser_copilot_test.go` with fixtures under `internal/hooks/testdata/copilot/` covering: each of the six event names, both `bash` and `powershell` populated, single-platform entries, and malformed JSON

## 4. Plugin scanner (Claude only)

- [x] 4.1 Implement v2-schema parser for `~/.claude/plugins/installed_plugins.json` (`{ "version": 2, "plugins": { "<key>": [ { "scope", "installPath", "version", ... }, ... ] } }`) that yields `(key, scope, installPath)` tuples
- [x] 4.2 Resolve each tuple to `<installPath>/hooks/hooks.json`; skip silently when the file does not exist
- [x] 4.3 Defensively reject any resolved path containing a `.git/hooks/` segment (assert in tests); do NOT scan `~/.claude/plugins/marketplaces/` and do NOT walk `installPath` looking for hook files
- [x] 4.4 Generate plugin source labels in the form `Plugin: <pluginID>@<marketplaceID>`, splitting the key on the last `@`; when more than one entry exists for the same key, include `(<scope>)` suffix in the label
- [x] 4.5 Add `plugin_scan_test.go` with a fake `~/.claude/plugins/` tree under `testdata/plugins/` exercising: single-entry plugin with `<installPath>/hooks/hooks.json` present, single-entry plugin with hooks file missing, multi-entry plugin with two scopes, marketplaces-tree hooks file present (must be ignored), `.git/hooks/` directory present in `installPath` (must NOT cause scanner to walk into it), malformed `installed_plugins.json`

## 5. Service and Wails wrapper

- [x] 5.1 Create `internal/hooks/service.go` with `Service` struct, `NewService()`, and `NewServiceWithRoot(home string)` for tests (mirrors marketplace/plugin service constructor pattern)
- [x] 5.2 Implement `ListHooks(agentID string, opts ListOpts) (Listing, error)`; for Claude run user/local/plugin parsers, sort sources User → Local → Plugin (plugins alpha by label), drop empty sources
- [x] 5.3 For Copilot when `opts.Workspace == ""` return an empty Listing with no error and an explanatory marker the frontend can use to render the empty state
- [x] 5.4 For Copilot with non-empty `opts.Workspace` (v1) ignore the value, behave identically to the empty case, do not error
- [x] 5.5 Compute the cross-source duplicate count map (by `command` string, per agent) and include the per-Hook `appears in N sources` count in the returned Listing
- [x] 5.6 Surface per-source errors on `Listing.Errors` rather than aborting the whole call
- [x] 5.7 Add `service_test.go` covering: end-to-end happy path with user + local + two plugin sources, empty source hidden, parse error rendering inline, source ordering, duplicate detection across sources, Copilot empty-state shape
- [x] 5.8 Create `app/hooks.go` thin wrapper holding `context.Context` (mirrors `app/sessionhistory.go` shape)
- [x] 5.9 Bind `app.HooksApp` from `main.go` alongside the existing `app.*App` wrappers; verify `frontend/wailsjs/go/app/HooksApp.{js,d.ts}` regenerates after `wails dev`

## 6. Frontend store and routing

- [x] 6.1 Create `frontend/src/stores/hooks.ts` (Pinia setup-style) exposing `fetchHooks(agentID)`, `selectHook(hookID)`, and reactive state for `listings`, `loading`, `errors`, `currentAgent`, `currentHookID` (mirrors `stores/sessionHistory.ts` shape)
- [x] 6.2 Mock `@wailsjs/go/app/HooksApp` and `@wailsjs/runtime/runtime` in a new `frontend/src/stores/__tests__/hooks.spec.ts` covering: fetchHooks success, fetchHooks parse-error path, selectHook clearing on agent switch
- [x] 6.3 Register `/hooks` route in `frontend/src/router/index.ts` (lazy-loaded, `name: 'hooks'`, gated by the existing setup guard)
- [x] 6.4 Add the "Hooks" entry between "Sessions" and "Installed" in `frontend/src/layouts/DefaultLayout.vue` (use `AppIcon` with an icon that matches the existing iconography)

## 7. Frontend components and view

- [x] 7.1 Create `frontend/src/components/hooks/HookSourceBadge.vue` rendering `User`/`Local`/`Plugin: <label>` chips using the existing `AppBadge` primitive
- [x] 7.2 Create `frontend/src/components/hooks/HookList.vue` for the left pane: collapsible source groups (use `AppCollapse`), nested collapsible event groups, hook rows with the summary format from the spec ("Hook list rows show summary metadata"), with a parse-error inline row when a source has an error
- [x] 7.3 Create `frontend/src/components/hooks/HookDetail.vue` for the right pane: header (source kind + event + full absolute source path, no truncation), resolved-fields region (per-handler-type sections), raw JSON region (read-only pretty-printed), `appears in N sources` chip when applicable, "Open settings file" button wired to `os-path-opener`
- [x] 7.4 Create `frontend/src/views/HooksView.vue` mirroring `SessionHistoryView.vue` exactly: per-agent tabs when >1 enabled, single-agent view when ==1, `AppEmptyState` linking to `/settings?tab=agents` when 0
- [x] 7.5 Render the `disableAllHooks` banner above the source tree on the Claude tab when the Listing reports the flag, including the source path that set it
- [x] 7.6 Render the Copilot tab as an `AppEmptyState` in v1 explaining "Copilot CLI hooks are project-scoped; workspace selection is coming"
- [x] 7.7 Verify zero leaks: run the mechanical leak check from `CLAUDE.md` against `frontend/src/views/HooksView.vue` and `frontend/src/components/hooks/`; fix any inline daisyUI class usage by extracting to App* primitives

## 8. Integration with existing capabilities

- [x] 8.1 Wire the "Open settings file" button to the existing `os-path-opener` capability binding (do not introduce a new OS-shell helper inside `internal/hooks/`)
- [x] 8.2 Confirm the agent ordering used on the Hooks page matches the order returned by `useAgentStore`, i.e. identical to Browse / Installed / Sessions

## 9. Tests, docs, and rules

- [x] 9.1 Run `go fmt ./internal/hooks/... ./app/hooks.go ./main.go` and `go vet ./...` (per project convention)
- [x] 9.2 Run `go test ./internal/hooks/...` and confirm all unit tests pass
- [x] 9.3 Run `pnpm format` and `pnpm lint` and `pnpm type-check` from `frontend/`
- [x] 9.4 Run `pnpm run test:unit -- src/stores/__tests__/hooks.spec.ts` and confirm new store tests pass
- [ ] 9.5 Smoke test under `wails dev`: with at least one Claude plugin that ships hooks installed, navigate to `/hooks`, verify User / Local / Plugin groupings render as expected, click into a hook, verify the detail pane shows the full path and resolved fields, click "Open settings file" and verify the file is revealed in OS file manager (manual; requires `wails dev` interactive run by user)
- [ ] 9.6 Smoke test the empty paths: temporarily delete `~/.claude/settings.json`, verify the User group is hidden and other sources still render; restore the file (manual)
- [ ] 9.7 Smoke test the disableAllHooks banner: temporarily add `"disableAllHooks": true` to `~/.claude/settings.json`, verify the banner renders with the correct source path, verify the list still shows every hook; remove the flag (manual)
- [ ] 9.8 Smoke test the Copilot tab: confirm it renders the explanatory empty state and that no `~/.copilot/*` file is opened (check via filesystem audit or logs) (manual)
- [x] 9.9 Update the leak-check section of `CLAUDE.md` to add `frontend/src/components/hooks` to the list of directories that must show zero matches
- [x] 9.10 Verify `openspec validate add-hooks-viewer --strict` still passes after any spec adjustments made during implementation
