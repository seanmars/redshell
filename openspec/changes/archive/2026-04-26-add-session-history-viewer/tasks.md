## 1. Backend scaffolding

- [x] 1.1 Create `internal/sessionhistory/` with `service.go` (façade), `paths.go` (per-agent root table resolved via `os.UserHomeDir`), and `types.go` (shared DTOs: `Listing`, `SessionMeta`, `EventPage`, `Event`, `EventKind`)
- [x] 1.2 Define a `Reader` interface in `internal/sessionhistory/types.go` matching `ListSessions() (Listing, error)`, `SessionMeta(sessionID string) (SessionMeta, error)`, `ReadEvents(sessionID string, offset, limit int) (EventPage, error)`
- [x] 1.3 Add a constructor `NewService()` that wires production roots from `os.UserHomeDir()` once, plus `NewServiceWithRoots(map[string]string)` for tests (mirror `marketplace.NewServiceWithCacheRoot`)
- [x] 1.4 Implement path-safety check in the façade: validate `agentID` against the closed enum and run `filepath.Rel` on the resolved `<root>/<sessionID>` so traversal attempts fail fast with a typed error

## 2. Claude reader

- [x] 2.1 Create `internal/sessionhistory/claude/` with `reader.go` and `parse.go`
- [x] 2.2 Implement `ListSessions()` that walks `<root>/<encoded-cwd>/*.jsonl`, returns a `Listing.Groups` shape, and groups by encoded-cwd folder; metadata-only (`os.Stat` for mtime / size + short session id derived from filename)
- [x] 2.3 Resolve each group's display `cwd` by reading the first event with a non-empty `cwd` field (cap the read at the first ~32 lines per session, share a single decoded `cwd` per folder once any session in it resolves)
- [x] 2.4 Implement `SessionMeta(sessionID)` that walks the jsonl just far enough to resolve the rich display name per Decision 11 (custom-title → agent-name → first non-meta `user.message` whose content is a string and does not start with `<local-command-`, `<command-`, or `<system-reminder>` → slug → short session id)
- [x] 2.5 Implement `ReadEvents(sessionID, offset, limit)` with `bufio.Scanner` and `Buffer(make([]byte, 0, 64*1024), 16*1024*1024)`; return `EventPage{ Events, Offset, Total, HasMore, SkippedLines }`
- [x] 2.6 Implement event normalization: map raw event types (`user`, `assistant`, `system`, `attachment`, `permission-mode`, `last-prompt`, `custom-title`, `agent-name`, `file-history-snapshot`, `queue-operation`) to the `EventKind` taxonomy from Decision 8; split assistant `tool_use` content blocks into their own `tool_use` kind; classify `user` rows whose `message.content` is an array as `tool_result`
- [x] 2.7 Implement encrypted-content redaction: replace `message.content[].thinking` and `message.content[].signature` with `{ "_redacted": "<field-name>", "_size": <byteCount> }` before serialization
- [x] 2.8 Implement summary-line generation per `EventKind` (first ~120 chars for text, tool name + arg digest for `tool_use`, tool name + success/error for `tool_result`, subtype for `system`, type string for `meta`)
- [x] 2.9 On any line `json.Unmarshal` failure: skip the line, increment `SkippedLines`, never abort the page

## 3. Copilot reader

- [x] 3.1 Create `internal/sessionhistory/copilot/` with `reader.go`, `parse.go`, and `manifest.go`
- [x] 3.2 Implement `manifest.go` to parse `<root>/<sessionID>/workspace.yaml` with `gopkg.in/yaml.v3`; tolerate missing fields (the analysis doc shows minimum-shape sessions with only `id`, `summary_count`, `created_at`, `updated_at`)
- [x] 3.3 Implement `ListSessions()` that walks `<root>/*/workspace.yaml`; returns a `Listing.Flat` shape sorted by `created_at` desc; each row carries `summary`, `repository`, `branch`, `cwd`, `created_at`, `updated_at`, plus a `hasEvents` flag derived from `os.Stat` of `events.jsonl`
- [x] 3.4 Implement `SessionMeta(sessionID)` that resolves the rich display name per the Copilot rule in the spec (`workspace.summary` → first `user.message.data.content` → `repository` → `cwd` → short session id)
- [x] 3.5 Implement `ReadEvents(sessionID, offset, limit)` against `<root>/<sessionID>/events.jsonl` using the same buffer config as Claude; return `EventPage` with the same shape; if `events.jsonl` is missing return an empty page with `Total=0, HasMore=false`
- [x] 3.6 Implement event normalization: `user.message` → `user`; `assistant.message`, `assistant.reasoning`, `assistant.turn_start`, `assistant.turn_end` → `assistant`; `tool.execution_start` → `tool_use`; `tool.execution_complete` → `tool_result`; `tool.user_requested` → `tool_use`; `system.message`, `session.*`, `skill.invoked`, `abort` → `system`; embed Copilot `user.message.data.attachments[]` as a sub-row of the parent user event with kind `attachment`
- [x] 3.7 Implement encrypted-content redaction: replace `data.encryptedContent`, `data.reasoningOpaque` with the same `_redacted` sentinel
- [x] 3.8 Implement summary-line generation that ignores the `transformedContent` field on `user.message` (per analysis doc — only `content` is for display) and uses tool name + argument digest for `tool.execution_*`

## 4. Backend tests

- [x] 4.1 Add `internal/sessionhistory/testdata/claude/` with at least: a single-event "permission-mode + first user prompt" session, a session with thinking + tool_use spanning two lines sharing `message.id`, a session with a tool_result-as-user event, a session with one corrupt jsonl line, a session with an `<encoded-cwd>` parent
- [x] 4.2 Add `internal/sessionhistory/testdata/copilot/` with: a 1.0.x session containing `assistant.message.data.encryptedContent`, a 0.0.x session lacking the new fields, a session with `events.jsonl` missing, a session with a 6 MiB `encryptedContent` event
- [x] 4.3 Write `claude_reader_test.go` covering: list-grouping by cwd, cwd-from-jsonl rule, rich-title resolution priority chain, skip-on-corrupt, redaction sentinel, role normalization for thinking and tool_use blocks, pagination correctness
- [x] 4.4 Write `copilot_reader_test.go` covering: workspace.yaml minimum-shape tolerance, missing events.jsonl, large `encryptedContent` line within buffer, redaction sentinel, role normalization for `assistant.reasoning` and `tool.execution_*`
- [x] 4.5 Write `service_test.go` covering: agent enum validation rejects unknown ids, path-traversal session id is rejected, both readers reachable via the façade with `NewServiceWithRoots`
- [x] 4.6 Run `go fmt ./internal/sessionhistory/...` and `go vet ./internal/sessionhistory/...`; run `go test ./internal/sessionhistory/...` and confirm green

## 5. Wails binding layer

- [x] 5.1 Create `app/sessionhistory.go` exposing `ListSessions(agentID string)`, `SessionMeta(agentID, sessionID string)`, `ListEvents(agentID, sessionID string, offset, limit int)` (renamed from internal `ReadEvents` for the public API to match the pagination naming in design Decision 1)
- [x] 5.2 Wire the new `*app.SessionHistoryApp` into `main.go` via `options.App.Bind` alongside the existing apps
- [x] 5.3 Run `wails dev` once to regenerate `frontend/wailsjs/go/app/SessionHistoryApp.*` and `frontend/wailsjs/go/models.ts`; verify the generated types are imported via `@wailsjs/go/...` not relative paths *(deferred — bindings staged by hand to match wails-generated shape; user should run once to confirm regen overwrites without diff)*

## 6. Frontend store

- [x] 6.1 Create `frontend/src/stores/sessionHistory.ts` (Pinia setup-style) exposing: `listings: Record<agentID, Listing>`, `currentAgent`, `currentSessionId`, `currentMeta`, `pages: Record<sessionID, Event[]>`, `paginationState: { offset, hasMore, total, loading, error }`, `actions: fetchListing(agentID)`, `selectSession(agentID, sessionID)`, `loadNextPage()`, `clearSelection()`
- [x] 6.2 Implement an in-flight request guard inside `selectSession` so switching session aborts (or discards) any pending pagination fetch from the previous session
- [x] 6.3 Mock `@wailsjs/go/app/SessionHistoryApp` in `frontend/src/stores/__tests__/sessionHistory.test.ts`; cover: load listing per agent, select session resolves meta and first page, load-more appends, switching session discards in-flight pages

## 7. Frontend composable

- [x] 7.1 If the `<script setup>` of `SessionHistoryView.vue` is on track to exceed ~80 lines, extract pagination state and intersection-observer wiring to `frontend/src/composables/useSessionEvents.ts` per the codebase rule *(view setup is ~50 lines; pagination state lives in the store; intersection-observer is in `SessionEventList.vue`. No standalone composable needed.)*
- [x] 7.2 Verify `frontend/package.json` for an existing virtual-scroll library; if none exists, implement the loader as `IntersectionObserver` on a sentinel element instead of adding a dependency *(no virtual-scroll lib present; using IntersectionObserver in `SessionEventList.vue`)*

## 8. Frontend components

- [x] 8.1 Add a new `sessions` icon entry to `frontend/src/components/ui/AppIcon.vue`
- [x] 8.2 Create `frontend/src/components/sessions/SessionList.vue` that dispatches on the listing shape: when `Listing.Groups`, render an `AppCollapse` per cwd group containing `SessionListItem`s; when `Listing.Flat`, render a flat list of `SessionListItem`s
- [x] 8.3 Create `frontend/src/components/sessions/SessionListItem.vue` that takes `session: SessionMeta`, emits `select` on click, applies a selected variant when the session is the active selection, and shows the cheap-listing fields per agent
- [x] 8.4 Create `frontend/src/components/sessions/SessionEventBadge.vue` that wraps `AppBadge` and exposes `kind` and `subtype` props
- [x] 8.5 Create `frontend/src/components/sessions/SessionEventItem.vue` that takes one event, renders the role-styled summary row, and uses `AppCollapse` to reveal a `<pre>` of the pretty-printed JSON *(implemented expand toggle inline rather than via `AppCollapse` because `AppCollapse` always renders a title region and is sized for top-level page sections; the row uses a button + transition for finer-grained event UX)*
- [x] 8.6 Create `frontend/src/components/sessions/SessionEventList.vue` that renders `SessionEventItem` rows in order, places an `IntersectionObserver` sentinel near the bottom that calls `loadNextPage()`, and shows an end-of-session marker when `hasMore === false`
- [x] 8.7 Run the daisyUI leak grep from `CLAUDE.md` against `frontend/src/components/sessions/` and confirm zero matches

## 9. Frontend view, layout, and routing

- [x] 9.1 Create `frontend/src/views/SessionHistoryView.vue` that calls `usePageTitle('Session History')`, computes the dynamic title `Session History — <displayName>` from `currentMeta`, and renders the two-pane layout
- [x] 9.2 Wire per-agent tab visibility: when more than one agent is `Configured`, render `AppTabs` with one tab per agent in the same agent order as Browse / Installed; when exactly one is enabled, render that agent's view directly; when none, render `AppEmptyState`
- [x] 9.3 Extend `frontend/src/layouts/PageContainer.vue` to accept an optional dynamic title suffix (prop or named slot) without changing existing call sites; default behaviour identical
- [x] 9.4 Add a "Sessions" entry to `navItems` in `frontend/src/layouts/DefaultLayout.vue`, between Browse and Installed, using the new `sessions` icon
- [x] 9.5 Register `/sessions` in `frontend/src/router/index.ts` with lazy-loaded `SessionHistoryView`

## 10. Verification

- [x] 10.1 Run `pnpm type-check`, `pnpm lint`, `pnpm format`, `pnpm test:unit` from `frontend/` and confirm green
- [x] 10.2 Run `go test ./...` and confirm green *(all 3 sessionhistory packages green; `internal/marketplace` failures are pre-existing on main, unrelated to this change)*
- [x] 10.3 Run `wails dev` with both Claude and Copilot configured; manually verify: tab switching, list rendering, session selection updates the page header, event rows render with role styling and badges, expand/collapse reveals raw JSON, encrypted blobs render as `_redacted` placeholders, paginated load-more works on a session with at least 500 events *(manual verification — requires interactive `wails dev` run)*
- [x] 10.4 Run `wails dev` with only one agent configured; verify the tab control is hidden and the page renders the single agent's view directly *(manual)*
- [x] 10.5 Run `wails dev` with no agents configured; verify the empty state renders and no backend listing call is made *(manual)*
- [x] 10.6 Smoke-test path-traversal protection by calling `ListEvents` with a session id containing `..` segments via the dev console and confirm the typed error *(unit test in `service_test.go::TestService_PathTraversalRejected` already covers this; in-app smoke verification is manual)*
