## Purpose

Define the Session History viewer that lets the user browse, select, and inspect past sessions for each configured AI coding agent (Claude Code, GitHub Copilot CLI). The viewer reads agent-owned session files on disk, redacts encrypted/signature-bearing fields, and renders an event timeline with safe paginated reads.

## Requirements

### Requirement: Session history page is reachable from the sidebar
The system SHALL expose a top-level Session History page reachable via the route `/sessions` and via a sidebar navigation entry labelled "Sessions".

#### Scenario: Sidebar entry navigates to the session history page
- **WHEN** the user clicks the "Sessions" entry in the sidebar
- **THEN** the router SHALL navigate to `/sessions` and the main content area SHALL render the Session History page

#### Scenario: Direct route resolves to the session history page
- **WHEN** the user navigates the application to `/sessions`
- **THEN** the Session History page SHALL render without redirecting to another route

### Requirement: Per-agent tab visibility follows enabled agents
The system SHALL render a per-agent tab control on the Session History page only when more than one agent is enabled, and SHALL render the single agent's view directly without a tab control when exactly one agent is enabled.

#### Scenario: Two or more agents enabled
- **WHEN** the user has both Claude and Copilot configured (`Configured == true`)
- **THEN** the page SHALL render an `AppTabs` with one tab per configured agent, in the same agent order used by the Browse Plugins and Installed Plugins pages

#### Scenario: Exactly one agent enabled
- **WHEN** the user has exactly one agent configured
- **THEN** the page SHALL render that agent's session history viewer directly without a tab control

#### Scenario: No agents enabled
- **WHEN** no agent is configured
- **THEN** the page SHALL render an `AppEmptyState` directing the user to set up an agent and SHALL NOT call any session-listing backend method

### Requirement: Two-pane layout with session list and event viewer
The system SHALL render the Session History page as a two-pane layout where the left pane is a session list and the right pane is the session content viewer.

#### Scenario: Initial render with no session selected
- **WHEN** the user opens the Session History page
- **THEN** the left pane SHALL render the session list and the right pane SHALL render an empty state prompting the user to select a session

#### Scenario: Selecting a session populates the right pane
- **WHEN** the user clicks a session in the left pane
- **THEN** the right pane SHALL render the event timeline for that session and the selected row SHALL display a visually distinct selected state

#### Scenario: Switching agents resets the selection
- **WHEN** the user switches the active agent tab
- **THEN** the previously selected session SHALL be cleared and the right pane SHALL return to its empty state until a session in the new agent's list is selected

### Requirement: Claude session list is grouped by working directory
The system SHALL group Claude sessions in the left pane by their working directory (`cwd`), with each group rendered as a collapsible section.

#### Scenario: Sessions grouped under their cwd
- **WHEN** the Claude session list renders
- **THEN** sessions SHALL appear under a collapsible group whose header shows a shortened form of the working directory in the form `{parent}/{root}` (the last two path segments), and each group SHALL contain only sessions whose `cwd` matches the group header
- **AND** the `{parent}` segment SHALL be rendered with reduced opacity so that the trailing `{root}` segment reads as the dominant label
- **AND** the full working directory path SHALL be available via the header element's native tooltip (`title` attribute)

#### Scenario: cwd is resolved from inside the session file
- **WHEN** the Claude session list resolves a group's display `cwd`
- **THEN** the displayed `cwd` SHALL be read from the first event inside the session file that carries a non-empty `cwd` field, and SHALL NOT be derived by reverse-decoding the `<encoded-cwd>` directory name

#### Scenario: Groups are sorted by recency
- **WHEN** multiple cwd groups exist
- **THEN** the groups SHALL be ordered by the most recent `mtime` of any session inside the group, descending

#### Scenario: Sessions inside a group are sorted by recency
- **WHEN** a cwd group is expanded
- **THEN** its sessions SHALL be ordered by `mtime` descending

### Requirement: Copilot session list is grouped by working directory
The system SHALL group Copilot sessions in the left pane by their resolved working directory, with each group rendered as a collapsible section using the same visual treatment as Claude.

#### Scenario: Sessions grouped under their resolved cwd
- **WHEN** the Copilot session list renders
- **THEN** sessions SHALL appear under a collapsible group whose header shows a shortened form of the working directory in the form `{parent}/{root}` (the last two path segments)
- **AND** the `{parent}` segment SHALL be rendered with reduced opacity so that the trailing `{root}` segment reads as the dominant label
- **AND** the full working directory path SHALL be available via the header element's native tooltip (`title` attribute)

#### Scenario: cwd is resolved from workspace.yaml with documented fallbacks
- **WHEN** the Copilot adapter resolves a session's group key
- **THEN** the resolver SHALL return the first non-empty value from this ordered list: `workspace.yaml.cwd`, `workspace.yaml.git_root`, `workspace.yaml.repository`, the literal string `"(unknown)"`
- **AND** the resolver SHALL NOT read or parse `events.jsonl` to determine the group key

#### Scenario: Sessions with no resolvable cwd land in the unknown bucket
- **WHEN** a Copilot session has empty `cwd`, `git_root`, and `repository` fields in its `workspace.yaml`
- **THEN** the session SHALL appear under a single group whose key is the literal string `"(unknown)"` and whose header renders that string verbatim
- **AND** the session SHALL NOT be dropped from the listing

#### Scenario: Groups are sorted by recency
- **WHEN** multiple Copilot cwd groups exist
- **THEN** the groups SHALL be ordered by `max(created_at, updated_at)` of any session inside the group, descending

#### Scenario: Sessions inside a group are sorted by recency
- **WHEN** a Copilot cwd group is expanded
- **THEN** its sessions SHALL be ordered by `created_at` descending, matching the prior flat-list ordering

#### Scenario: Backend listing kind reflects the grouped shape
- **WHEN** the frontend calls `ListSessions("copilot")`
- **THEN** the returned `Listing.Kind` SHALL equal `"groups"` and `Listing.Groups` SHALL be populated with one `SessionGroup` per resolved cwd
- **AND** `Listing.Flat` SHALL be empty

### Requirement: Session list rows show summary metadata cheaply
The system SHALL render each session row with summary metadata that can be obtained without parsing the full session file.

#### Scenario: Claude session row contents
- **WHEN** a Claude session row renders in the list
- **THEN** the row SHALL show the session id (the file's UUID portion of `sessionId`) as the primary line, a short session id (first 8 characters of the UUID) as a tail tag, and the session file's modification time
- **AND** the row SHALL NOT repeat the encoded-cwd folder name (which is rendered on the parent group header)
- **AND** the row SHALL NOT block its render on parsing the session jsonl beyond what `os.Stat` provides

#### Scenario: Copilot session row contents
- **WHEN** a Copilot session row renders in the list
- **THEN** the row SHALL show the `workspace.yaml.summary` (or a fallback derived from `repository`, `cwd`, or short session id when `summary` is empty), the branch (when present), and the `created_at` time

### Requirement: Page header reflects the selected session
The system SHALL update the page header on the Session History page to reflect the currently selected session.

#### Scenario: No session selected
- **WHEN** no session is selected
- **THEN** the page header SHALL read "Session History"

#### Scenario: Session selected
- **WHEN** a session is selected
- **THEN** the page header SHALL read "Session History — <displayName>" where `displayName` is the resolved rich display name for that session

#### Scenario: Display name resolution for Claude
- **WHEN** the rich display name for a Claude session is resolved
- **THEN** the resolver SHALL return the first non-empty value from this ordered list: `custom-title` event's `customTitle`, `agent-name` event's `agentName`, the first non-meta `user.message` whose `message.content` is a string and does not begin with `<local-command-`, `<command-`, or `<system-reminder>`, the session's `slug`, the first 8 characters of `sessionId`

#### Scenario: Display name resolution for Copilot
- **WHEN** the rich display name for a Copilot session is resolved
- **THEN** the resolver SHALL return the first non-empty value from this ordered list: `workspace.yaml.summary`, the first `user.message` event's `data.content`, `workspace.yaml.repository`, `workspace.yaml.cwd`, the first 8 characters of the session id

### Requirement: Event timeline lists each JSONL event as a row
The system SHALL render the right-pane content viewer as an ordered list where each row corresponds to one parsed JSONL event.

#### Scenario: Event order matches file order
- **WHEN** the event timeline renders
- **THEN** events SHALL appear in the same order they appear in the source jsonl file, top to bottom

#### Scenario: Each row carries role-based styling
- **WHEN** an event row renders
- **THEN** the row SHALL apply visual styling (colour, icon, label) corresponding to its normalized event kind from this set: `user`, `assistant`, `tool_use`, `tool_result`, `system`, `attachment`, `meta`

#### Scenario: Each row shows a secondary subtype badge
- **WHEN** an event row renders
- **THEN** the row SHALL display a small badge containing the original event type string (for example `system.compaction_complete`, `attachment.diagnostics`, `assistant.reasoning`)

#### Scenario: Each row shows a short summary
- **WHEN** an event row renders in its collapsed state
- **THEN** the row SHALL display a short summary line — for `user` and `assistant` text content this is the first ~120 characters of the text; for `tool_use` it is the tool name and a one-line argument digest; for `tool_result` it is the tool name and a success/error indicator; for `system` it is the subtype; for `meta` it is the event type

### Requirement: Event rows expand to show full raw JSON
The system SHALL allow each event row to expand and reveal the complete parsed JSON payload.

#### Scenario: User expands a row
- **WHEN** the user clicks an event row's expand control
- **THEN** the row SHALL reveal a pretty-printed JSON view of the parsed event with monospace formatting and preserved nesting via an in-row collapse control (any visual implementation is acceptable as long as it preserves the surrounding row layout)

#### Scenario: Expanded raw JSON is read-only
- **WHEN** the raw JSON view is shown
- **THEN** the JSON SHALL be presented as read-only text and SHALL NOT be editable through the UI

#### Scenario: User collapses a row
- **WHEN** the user clicks the expand control of an already-expanded row
- **THEN** the row SHALL collapse back to its summary form

### Requirement: Encrypted content is replaced with a placeholder
The system SHALL replace encrypted or signature-bearing fields with a sentinel placeholder before rendering, both in the row summary and in the expanded raw JSON view.

#### Scenario: Claude thinking blocks are redacted
- **WHEN** a Claude `assistant` event includes a content block of type `thinking`
- **THEN** the block's `thinking` and `signature` fields SHALL be replaced with `{ "_redacted": "<field-name>", "_size": <byteCount> }` before reaching the frontend

#### Scenario: Copilot encrypted content is redacted
- **WHEN** a Copilot `assistant.message` event includes `data.encryptedContent` or `data.reasoningOpaque`
- **THEN** those fields SHALL be replaced with `{ "_redacted": "<field-name>", "_size": <byteCount> }` before reaching the frontend

#### Scenario: No raw-reveal toggle
- **WHEN** any event with redacted content is rendered
- **THEN** the UI SHALL NOT provide any control to reveal the original encrypted bytes

### Requirement: Session events are loaded in pages
The system SHALL deliver session events to the frontend in fixed-size pages and SHALL provide a `total` and `hasMore` indicator with each page.

#### Scenario: Initial page load
- **WHEN** the user opens a session
- **THEN** the system SHALL load the first page of events (default page size of 200 events) and render them

#### Scenario: Loading subsequent pages
- **WHEN** the user scrolls near the end of the loaded events
- **THEN** the system SHALL request the next page using the next offset and append its events to the rendered list

#### Scenario: Last page reached
- **WHEN** a returned page has `hasMore == false`
- **THEN** the frontend SHALL stop requesting further pages and MAY render an end-of-session marker

#### Scenario: Switching session cancels in-flight pagination
- **WHEN** the user selects a different session while a page request is in flight
- **THEN** the in-flight result SHALL NOT be appended to the new session's event list

### Requirement: Streaming JSONL parser tolerates large lines and corrupt lines
The system SHALL parse session JSONL files line by line with a buffer sized to accommodate single lines up to at least 4 MiB and SHALL skip lines that fail to parse without aborting the session render.

#### Scenario: Long line within buffer
- **WHEN** the parser encounters a line up to the configured buffer ceiling (at least 4 MiB)
- **THEN** the line SHALL be parsed without error

#### Scenario: Corrupt or partial line
- **WHEN** the parser encounters a line that fails `json.Unmarshal`
- **THEN** the parser SHALL skip the line, increment a per-session "skipped lines" counter, and continue parsing the rest of the file

#### Scenario: Skipped lines are reported
- **WHEN** any lines are skipped during a paginated read
- **THEN** the response SHALL include the count of skipped lines so the frontend can display a non-blocking warning

### Requirement: Session listing and event reads are read-only
The system SHALL only read session files and SHALL NOT modify, delete, rename, or write any file under `~/.claude/projects/` or `~/.copilot/session-state/`.

#### Scenario: Listing sessions performs no writes
- **WHEN** the system lists sessions for any agent
- **THEN** no file under the agent's session root SHALL be created, modified, or deleted

#### Scenario: Reading events performs no writes
- **WHEN** the system reads events for a session
- **THEN** no file under the agent's session root SHALL be created, modified, or deleted

### Requirement: Session id paths are resolved safely
The system SHALL validate session identifiers received from the frontend so that resolved paths cannot escape the agent's session root.

#### Scenario: Session id resolves inside the root
- **WHEN** a session id resolves to a path inside the configured agent root
- **THEN** the system SHALL accept the request and proceed with the read

#### Scenario: Session id escapes the root
- **WHEN** a session id resolves to a path outside the configured agent root (for example via `..` segments)
- **THEN** the system SHALL reject the request with a typed error and SHALL NOT open any file
