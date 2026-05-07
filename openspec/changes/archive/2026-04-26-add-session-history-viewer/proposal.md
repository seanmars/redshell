## Why

Both supported agents persist rich, locally-stored session history — Claude Code under `~/.claude/projects/<encoded-cwd>/<sessionId>.jsonl` and GitHub Copilot under `~/.copilot/session-state/<sessionId>/` — but RedShell currently exposes none of it. To inspect a past conversation, users have to navigate the filesystem, hand-decode encoded `cwd` directory names, and read raw JSONL with two completely different schemas. Surfacing this history inside RedShell turns an existing-on-disk artifact into something users can actually skim, search, and learn from, and it leverages our agent-detection layer (`internal/agent`) to show only what is relevant for the agents the user has configured.

## What Changes

- Add a new `/sessions` route with a top-level sidebar entry ("Sessions") in `DefaultLayout.vue` between Plugins and Installed.
- Add a new Go domain `internal/sessionhistory/` with two adapters — `claude` (reads `~/.claude/projects/`) and `copilot` (reads `~/.copilot/session-state/`) — plus a façade service that the Wails app binds to.
- Add `app/sessionhistory.go` exposing `ListSessions(agentID)`, `GetSession(agentID, sessionID)`, and `StreamEvents(agentID, sessionID, offset, limit)` so the frontend can list and chunk-load potentially large `.jsonl` files.
- Render a two-pane layout in `SessionHistoryView.vue`: left pane is a virtualized session list grouped per agent, right pane is the session content viewer. When the user has more than one agent configured, the page hosts an `AppTabs` with one tab per configured agent (matching the Browse / Installed pattern); when only one agent is configured, the tab bar is omitted.
- Render each parsed JSONL event as a collapsed list item in the right pane, styled by role (`user`, `assistant`, `tool`, `system`, `attachment`, plus agent-specific role tags). Each item is collapsed by default and expands to show the full raw JSON via an `AppCollapse`. Encrypted blobs (`thinking`, `reasoningOpaque`, `encryptedContent`) SHALL render as a placeholder, never as raw text.
- Make the page header reactive: when no session is selected, the title is `Session History`; when a session is selected, the title becomes `Session History — <displayName>` where `displayName` is resolved per-agent (Claude: custom-title → agent-name → first user message → `sessionId[0:8]`; Copilot: `workspace.summary` → first user message → `repository` → `cwd` → `sessionId[0:8]`).
- Streaming JSONL parser SHALL use `bufio.Scanner` with a buffer raised to at least 4 MiB to tolerate long lines (Copilot `assistant.message` events can carry multi-hundred-KB `encryptedContent`).
- Session listing SHALL be read-only — no write paths, no deletion UI, no parsing of sidecar files (`tool-results/*.txt`, `rewind-snapshots/backups/*`) in this change.

## Capabilities

### New Capabilities

- `session-history-viewer`: Read-only browser for agent session histories. Provides per-agent session lists and an event-by-event timeline view with collapsible raw-JSON inspection, sourced directly from each agent's on-disk session store.

### Modified Capabilities

- `wails-app-shell`: Sidebar gains a new top-level "Sessions" nav entry routing to `/sessions`. The page title displayed by `PageContainer` becomes reactive to a child-component-driven dynamic suffix so that the selected session's identifier appears in the header.

## Impact

- **New backend**: `internal/sessionhistory/service.go`, `internal/sessionhistory/claude/`, `internal/sessionhistory/copilot/`, `internal/sessionhistory/testdata/`, plus `app/sessionhistory.go` Wails wrapper.
- **Wails bindings**: regenerated `frontend/wailsjs/go/app/SessionHistoryApp.*` and `frontend/wailsjs/go/models.ts` entries.
- **New frontend**: `frontend/src/views/SessionHistoryView.vue`, `frontend/src/components/sessions/{SessionList.vue,SessionListItem.vue,SessionEventList.vue,SessionEventItem.vue,SessionEventBadge.vue}`, `frontend/src/stores/sessionHistory.ts`, optionally `frontend/src/composables/useSessionEvents.ts` if the loader logic exceeds ~80 lines.
- **Modified frontend**: `frontend/src/router/index.ts` (route registration), `frontend/src/layouts/DefaultLayout.vue` (sidebar nav item), `frontend/src/layouts/PageContainer.vue` (support a dynamic-title slot or prop), `frontend/src/components/ui/AppIcon.vue` (new `sessions` icon).
- **Filesystem access**: read-only access to `~/.claude/projects/` and `~/.copilot/session-state/`. No git, network, or shell-out to agent CLIs is added.
- **No schema churn elsewhere**: marketplace registry, plugin install paths, agent detection — all untouched.
- **Tests**: new `internal/sessionhistory/*_test.go` covering both parsers against fixtures committed under `testdata/`; new Vitest specs for the `sessionHistory` Pinia store mocking the Wails bindings.
- **Performance budget**: `ListSessions` is O(number of session directories) with metadata-only reads; `GetSession`/`StreamEvents` MUST stream and SHALL NOT load whole files into memory. Frontend rendering virtualizes the event list to handle sessions with thousands of events.
