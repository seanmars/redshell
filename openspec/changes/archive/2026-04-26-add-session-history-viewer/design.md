## Context

Both supported agents persist conversation history on disk in completely different shapes:

- **Claude Code**: `~/.claude/projects/<encoded-cwd>/<sessionId>.jsonl`. Each line is one event (`user`, `assistant`, `system`, `attachment`, `permission-mode`, `file-history-snapshot`, etc). Same API turn can span multiple lines sharing a `message.id`. Lines can carry hundreds of KB of tool input/output. The `<encoded-cwd>` folder name is a lossy ASCII encoding of the working directory — the true `cwd` lives inside any event's `cwd` field. There can be 200+ sessions per project folder; a typical machine has 30+ project folders and several hundred sessions.
- **GitHub Copilot CLI**: `~/.copilot/session-state/<sessionId>/`. A directory per session containing a YAML manifest (`workspace.yaml`), `events.jsonl`, optional `session.db` (SQLite), `plan.md`, `checkpoints/`, `rewind-snapshots/`, `inuse.<pid>.lock`. Single events can contain ~hundreds of KB of `encryptedContent` / `reasoningOpaque` base64 blobs. The schema has visibly evolved across CLI versions (`0.0.x` → `1.0.x`); fields like `encryptedContent`, `assistant.reasoning`, `session.compaction_*`, `inbox_entries` only exist in newer versions.

The two stores share almost no structure. Each needs its own parser; the Wails layer abstracts behind a per-agent adapter so the frontend sees one shape.

The agent-detection layer (`internal/agent`) already exposes a `Configured` flag per agent based on `os.Stat` of `~/.claude` and `~/.copilot`; the frontend already follows the per-agent tab pattern (Browse / Installed). The reactive page header is currently `PageContainer.vue` with a static `title` prop — sessions need it dynamic.

Two reference docs in the repo (`claude-code-session.md`, `github-copilot-session.md`) catalogue every event subtype observed in real session data; this design intentionally treats them as the source of truth for parser behaviour, including all noted edge cases (encoded-cwd lossy folder names, `message.id`-spanning rows, encryptedContent buffer sizes, `tool_result` packaged inside `user` messages, schema drift).

## Goals / Non-Goals

**Goals:**

- Surface session history for any agent the user has configured, with a per-agent tab when more than one is configured (matching the existing Browse/Installed pattern).
- Two-pane layout: session list on the left, event timeline on the right.
- Reactive page header so the selected session's identifier appears in the title bar.
- Stream-friendly back-end so a 750 KB Claude jsonl or a Copilot `events.jsonl` with multi-MB `encryptedContent` rows does not OOM the app or block the UI.
- Strict read-only: viewer never writes to or deletes session state.
- Encrypted blobs (`thinking`, `reasoningOpaque`, `encryptedContent`) render as a "(encrypted)" placeholder; raw content is never decoded or displayed.
- Single source of truth for per-agent path resolvers in Go; frontend never touches paths.
- Failure isolation: a single corrupt jsonl line skips that line, never aborts the session render.

**Non-Goals (deliberately scoped out of this change):**

- Subagent / Task drill-in (Claude `subagents/agent-<agentId>.jsonl` directories) — surfaced as a future enhancement.
- Tool-results sidecar lazy load (`<sessionId>/tool-results/*.txt`).
- File-history-snapshot diff viewer (`~/.claude/file-history/<sessionId>/...`).
- Copilot `session.db` reader (todos / inbox), `plan.md` rendering, `checkpoints/` summary view, `rewind-snapshots/` diff view.
- Live tail / file-watch on running sessions (`~/.claude/sessions/<pid>.json`, `inuse.<pid>.lock` detection).
- Cross-session full-text search.
- Resume / fork / rewind actions; viewer is strictly read-only.
- Decryption or signature validation of `thinking` / `encryptedContent` payloads.

## Decisions

### Decision 1: Pagination, not server-push streaming, for event delivery

**Chosen:** `ListEvents(agentID, sessionID, offset, limit) -> { events, total, hasMore }` Wails method that returns parsed event objects in fixed-size chunks. Frontend uses a virtual list with an intersection-observer trigger that requests the next page when the user nears the end.

**Rationale:** Sessions can have thousands of events but the user's viewport renders ~30 at a time. A pull-based pagination API maps directly onto a virtual list and avoids the subscriber-lifecycle complexity of `runtime.EventsEmit` (cancel-on-route-change, dedupe across re-mounts, ordering across batches). The existing `plugin:install-log` pattern uses push because install logs arrive over time from a child process — session events already exist on disk, so push gains nothing.

**Alternative considered:** `runtime.EventsEmit("session:events", batch)` push streaming. Rejected: forces the frontend to maintain a buffer of events keyed by session, and to invalidate that buffer on session switch / unmount. Pagination keeps the store stateless per page.

**Consequence:** `internal/sessionhistory` parsers expose a `ParseRange(reader, offset, limit)` style interface, not a channel-based stream. They still read the file with `bufio.Scanner` so memory stays bounded; pagination is implemented by counting lines, not by random access.

### Decision 2: Claude session list groups by `cwd`, resolved from inside the jsonl

**Chosen:** Left pane for the Claude tab is a two-level tree: `cwd → sessions[]`. Each cwd group is a collapsible section (`AppCollapse`) sorted by most-recently-updated session inside it. Sessions inside a group are sorted by mtime descending.

**Rationale:** Sample data shows 38 cwd folders × 445 sessions on a typical developer machine. A flat 445-row list is unusable; users think in terms of projects ("the redshell session from Tuesday"), not in terms of session UUIDs. Grouping by cwd matches the on-disk layout and the user's mental model.

**`cwd` resolution rule:** the displayed cwd is read from the first event in the jsonl that has a non-empty `cwd` field, **never** decoded from the folder name. The reference doc explicitly warns that the encoding (`\` `/` `:` `.` → `-`) is lossy and irreversible. The resolver caches the decoded `cwd` per session in the metadata index.

**Alternative considered:** Flat list, cwd as sublabel; or cwd as filter chip. Rejected — both make scrolling 445 rows the primary interaction, which is worse than a click-to-expand cwd group.

**Consequence:** `ListSessions("claude")` returns `{ groups: [{ cwd, sessions: [meta] }] }`. Copilot returns `{ sessions: [meta] }` (no grouping — Copilot's `~/.copilot/session-state/` is already flat and ~40 sessions is fine to scroll).

### Decision 3: Lazy display-title resolution with metadata-only listing

**Chosen:** `ListSessions` returns metadata only — `sessionID`, `mtime`, `byteSize`, plus whatever the agent-specific cheap title source provides. Rich titles (first user prompt, custom-title) are resolved on a per-session basis when the user opens that session, not during list-build.

Per agent:

| Agent | Cheap title sources during list-build | Resolved on open |
|---|---|---|
| Claude  | `cwd` folder name (encoded), `sessionId[0:8]` | `custom-title` event → `agent-name` event → first non-meta `user.message` content → `slug` → `sessionId[0:8]` |
| Copilot | `workspace.yaml`'s `summary`, `repository`, `branch`, `cwd`, `created_at` | first `user.message.content` → `repository` → `cwd` → `sessionId[0:8]` |

**Rationale:** Copilot's `workspace.yaml` is small (1 KB), parses fast, contains `summary`, so listing all Copilot sessions with rich titles is cheap. Claude has no manifest — the title information is buried in `custom-title`, `agent-name`, or the first user message inside the jsonl. To list 445 sessions cheaply we cannot open all 445 jsonls. So we accept a lower-fidelity Claude list title (cwd-encoded folder + short session id) and resolve the better title only when needed.

**Alternative considered:** Eager parse-on-list with a disk cache (`~/.redshell/session-history-index.json`, invalidated by mtime). Rejected for v1: it adds an index file format, an upgrade path, and a cache-coherency story for a feature whose value is browsing, not searching. We can add the cache as a perf improvement once we measure the actual list-render time.

**Consequence:** Claude session list rows show `<encoded-cwd> · <sessionId-short> · <mtime>` until clicked; on click, the right pane resolves the rich title and the page header updates.

### Decision 4: `wails-app-shell` modification is narrow; reactive title lives in `session-history-viewer`

**Chosen:** Modify `wails-app-shell` only to add the "Sessions" sidebar nav entry. Add the reactive-title behaviour (`PageContainer` accepts a dynamic suffix slot) as an **ADDED Requirement** under `session-history-viewer`, since it is the only consumer.

**Rationale:** The reactive title is a session-viewer concern, not an app-shell concern. Putting it in the shell spec implies every page can vary its header — that's not what we're building, and other pages have static titles. Keeping the shell delta minimal also reduces archive-time churn.

**Alternative considered:** Bundle both modifications under `wails-app-shell`. Rejected — couples unrelated capabilities.

**Consequence:** Spec layout is:
- `specs/session-history-viewer/spec.md`: ADDED Requirements (the bulk of this change).
- `specs/wails-app-shell/spec.md`: a delta with one MODIFIED Requirement (the sidebar list now contains "Sessions" alongside the existing entries) — copied verbatim from the existing spec with only the affected scenario edited.

### Decision 5: Two parser packages behind a façade — `claude` and `copilot`

**Chosen:** `internal/sessionhistory/` exposes a `Service` with the Wails-bound methods. Internally it dispatches by `agentID` to `internal/sessionhistory/claude.Reader` and `internal/sessionhistory/copilot.Reader`. Each reader has the same Go interface:

```go
type Reader interface {
    ListSessions() (Listing, error)
    SessionMeta(sessionID string) (Meta, error)
    ReadEvents(sessionID string, offset, limit int) (EventPage, error)
}
```

`Listing` is a discriminated union shape (`Groups` for Claude, `Flat` for Copilot) so the frontend renders the right component without a feature-detect on the agent.

**Rationale:** The two formats are too different to share parsing code (`workspace.yaml` vs jsonl-with-permission-mode-prelude); but the surface contract is identical. The seam at `Reader` matches the existing `internal/<domain>/service.go` pattern.

**Path resolution:** each reader takes a root override in its constructor for testability (`NewReader(rootDir string)`); the `Service` wires production roots from `os.UserHomeDir()` once at boot. This mirrors `marketplace.NewService` / `NewServiceWithCacheRoot`.

### Decision 6: Streaming JSONL parser with raised buffer; corrupt-line tolerance

**Chosen:** Use `bufio.Scanner` with `Buffer(make([]byte, 0, 64*1024), 16*1024*1024)` (16 MiB max). On a `json.Unmarshal` error for a single line, the parser logs and skips that line; the page count and event index account for skipped lines so pagination remains stable.

**Rationale:** The Copilot reference doc explicitly cites > 64 KB lines for `assistant.message.encryptedContent`. The Claude reference doc cites attachments and tool inputs of similar size. 16 MiB is a generous ceiling — a single jsonl line over 16 MiB is a defect, not a payload. Skip-on-error is essential: a partial last-line write during an active session must not break the viewer.

**Alternative considered:** `json.Decoder` over the file. Rejected — `bufio.Scanner` plus `json.Unmarshal` per line gives clearer error attribution (which line failed) and handles BOM / CRLF cleanly with the default split function.

### Decision 7: Encrypted-content placeholder, not raw display

**Chosen:** When emitting events, the parser **strips** the following fields and replaces them with a sentinel `{ "_redacted": "<field-name>", "_size": <byteCount> }`:

| Agent | Fields redacted |
|---|---|
| Claude  | `message.content[].thinking`, `message.content[].signature` |
| Copilot | `assistant.message.data.reasoningOpaque`, `assistant.message.data.encryptedContent` |

The stripped event is what the frontend sees, both in the summary view and in the "expand to see raw JSON" view. There is no toggle to reveal the raw bytes.

**Rationale:** These fields are encrypted/redacted on purpose — they are not human-readable, leaking them in a UI surface (or worse, copying them to clipboard) is a misuse. Displaying the size lets the user understand "there was hidden content here, of approximately N bytes" without any forensic value loss.

**Alternative considered:** Keep raw bytes, render as collapsed `<pre>`. Rejected — forfeits the privacy benefit and bloats UI memory for content that is not viewable anyway.

### Decision 8: Per-agent role taxonomy, normalized for display

**Chosen:** The Go layer normalizes each event into a small union `EventKind` for the frontend's rendering:

| Frontend `kind` | Claude source | Copilot source |
|---|---|---|
| `user`        | `type=user` with string `message.content`, not `isMeta` | `user.message` |
| `assistant`   | `type=assistant` with text/thinking/tool_use blocks | `assistant.message`, `assistant.reasoning`, `assistant.turn_*` |
| `tool_use`    | content block `tool_use` (split out into its own kind) | `tool.execution_start` |
| `tool_result` | `type=user` with array `message.content` containing `tool_result` blocks | `tool.execution_complete` |
| `system`      | `type=system` (with subtype) | `system.message`, `session.info`, `session.compaction_*`, `session.shutdown`, etc |
| `attachment`  | `type=attachment` | (Copilot embeds attachments inside `user.message.attachments`; surfaced as a sub-row of the parent user event) |
| `meta`        | `type=permission-mode`, `type=last-prompt`, `type=custom-title`, `type=agent-name`, `type=file-history-snapshot`, `type=queue-operation`, `isMeta=true` rows | n/a |

Each event also carries the original `type` string so the frontend can show secondary badges (e.g. `system.compaction_complete`).

**Rationale:** The viewer's primary axis is "who said what" (user vs assistant vs tool vs system); the secondary axis is "what subtype" (informational, compaction, attachment, etc). A small normalized `kind` set keeps the styling rules simple — one colour/icon per kind, plus a free-form badge for the subtype.

### Decision 9: Frontend layout via existing primitives + one new domain folder

**Chosen:** Reuse `DefaultLayout`, `PageContainer`, `AppTabs`, `AppCollapse`, `AppBadge`, `AppButton`, `AppEmptyState`, `AppSkeleton`, `AppIcon`. Add a new domain folder `frontend/src/components/sessions/` containing:

- `SessionList.vue` — left pane, dispatches to per-agent list shape (grouped vs flat).
- `SessionListItem.vue` — single row with icon + display title + metadata line.
- `SessionEventList.vue` — right pane, virtualized with `vue-virtual-scroller` (already in lockfile? — verify in tasks; if not, build a simple `IntersectionObserver`-driven loader to avoid pulling a new dep).
- `SessionEventItem.vue` — one event row, role-styled, with `AppCollapse` for raw JSON expansion.
- `SessionEventBadge.vue` — small role/kind badge wrapping `AppBadge`.

A composable `useSessionEvents(agentID, sessionID)` encapsulates pagination state and is shared between the view and any future export feature.

**Rationale:** Stays inside the existing daisyUI primitive boundaries documented in `CLAUDE.md`. New components live under `components/sessions/` matching `components/plugin/`, `components/marketplace/`, `components/agent/`.

**Verify before introducing a dependency:** if `vue-virtual-scroller` is not in the existing `frontend/package.json`, fall back to lazy-load via `IntersectionObserver` plus a "load more" sentinel; do not add a new dep just for this view.

### Decision 10: Per-agent path resolvers are an internal table

**Chosen:** `internal/sessionhistory/paths.go` holds a single source of truth:

```go
var agentRoots = map[string]string{
    "claude":  "~/.claude/projects",
    "copilot": "~/.copilot/session-state",
}
```

Resolved once via `os.UserHomeDir()` at `Service` construction. Reader instances receive the absolute root.

**Rationale:** Mirrors the `AgentMarketplaceFiles` pattern in `internal/marketplace/service.go`. Adding a third agent in the future is one map entry plus a new reader package.

### Decision 11: First user message extraction rules (Claude)

**Chosen:** When resolving the rich display title for a Claude session, walk the jsonl line-by-line and select the first event satisfying ALL of:
1. `type === "user"`.
2. `isMeta !== true`.
3. `message.content` is a string (not an array — array means tool_result).
4. The string does not start with `<local-command-`, `<command-`, or `<system-reminder>`.
5. The string is not a system caveat injection (`/^Caveat:/` etc).

Truncate to ~80 characters for display.

**Rationale:** The Claude jsonl prepends multiple system-injected user messages (caveat, slash-command echo, hook outputs). The reference doc enumerates these explicitly; ignoring them is required to surface the user's actual first prompt.

## Risks / Trade-offs

- **[Risk]** Schema drift: Copilot's events.jsonl shape changed visibly between `0.0.394` and `1.0.36`; new fields will appear post-ship. **Mitigation:** parsers use feature-detection (optional fields, `omitempty` in DTOs); unknown event types render as a generic `system` kind with the original `type` string in a badge rather than failing.
- **[Risk]** Large sessions slow first paint. A 750 KB Claude jsonl with 1000 events takes meaningful time to parse on cold disk. **Mitigation:** pagination (Decision 1) means we only parse the first `limit` lines on open; subsequent pages parse incrementally. We measure this in tasks before deciding whether to add a metadata cache (Decision 3 alternative).
- **[Risk]** Rich Claude title resolution requires a second jsonl read per opened session (cheap-list vs rich-on-open). **Mitigation:** the rich-title read can stop at the first matching user message — typically within the first ~20 lines.
- **[Risk]** Path-traversal: `agentID` and `sessionID` arrive from the frontend. **Mitigation:** the Reader resolves `<root>/<sessionID>` and rejects with a typed error if the resolved path escapes the root (`filepath.Rel` check). `agentID` is a closed enum (`claude` | `copilot`) validated at the façade.
- **[Risk]** Concurrent access: Claude Code may be appending to a jsonl while the user views it. **Mitigation:** the parser tolerates partial last lines (Decision 6 skip-on-error); we expose this only as a footnote in the UI ("session may be running"), no live tail in v1.
- **[Trade-off]** Pagination instead of streaming means the user cannot see brand-new events without reload. Accepted — live tail is on the non-goals list and can be added later via push events.
- **[Trade-off]** Two reader packages without shared parsing code means more lines of Go. Accepted — the formats genuinely don't overlap; a forced abstraction would obscure both.
- **[Trade-off]** Encrypted blobs render as a placeholder, even though some users might want to copy them out for debugging. Accepted — viewer is for human review, not forensics.

## Migration Plan

This is a pure additive change with no schema or filesystem migration:

1. Implement `internal/sessionhistory/` (façade, paths table, claude reader, copilot reader, fixtures, tests).
2. Implement `app/sessionhistory.go` and bind it in `main.go`.
3. Run `wails dev` once to regenerate `frontend/wailsjs/go/app/SessionHistoryApp.*`.
4. Implement frontend store, composable, components, view in that order.
5. Register the route in `router/index.ts` and add the sidebar entry to `DefaultLayout.vue`.
6. Extend `PageContainer.vue` to accept an optional dynamic title suffix; default behaviour unchanged.
7. Run `go test ./...`, `pnpm test:unit`, `pnpm lint`, `pnpm format`, `pnpm type-check`.
8. Manual smoke: with both agents configured, open a Claude session and a Copilot session each; verify role styling, raw-JSON expansion, dynamic header, encrypted-blob placeholder.

**Rollback:** revert the branch. No on-disk state, no Wails binding contract changes elsewhere, no migration to undo.

## Open Questions

- **Virtual scrolling library**: prefer to verify `frontend/package.json` for an existing virtual-scroller before adding `vue-virtual-scroller`. If none exists, the fallback (`IntersectionObserver` + page sentinel) is acceptable but slightly less smooth on very large sessions.
- **Display-title cache**: not adding one in v1 (Decision 3). Revisit if list-render p95 is over ~250 ms on a machine with 500 sessions.
- **Subagent navigation**: deferred. The viewer renders Task / Agent tool_use events as ordinary tool rows in v1; future enhancement adds a "view subagent transcript" link that opens `subagents/agent-<agentId>.jsonl` in a sub-view.
- **Live session indicator**: deferred. Could be added cheaply via a static check of `~/.claude/sessions/*.json` and `inuse.<pid>.lock` at list time, but tail/refresh is out of scope.
