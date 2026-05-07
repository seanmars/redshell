# hooks-viewer Specification

## Purpose
TBD - created by archiving change add-hooks-viewer. Update Purpose after archive.
## Requirements
### Requirement: Hooks page is reachable from the sidebar
The system SHALL expose a top-level Hooks page reachable via the route `/hooks` and via a sidebar navigation entry labelled "Hooks".

#### Scenario: Sidebar entry navigates to the hooks page
- **WHEN** the user clicks the "Hooks" entry in the sidebar
- **THEN** the router SHALL navigate to `/hooks` and the main content area SHALL render the Hooks page

#### Scenario: Direct route resolves to the hooks page
- **WHEN** the user navigates the application to `/hooks`
- **THEN** the Hooks page SHALL render without redirecting to another route

#### Scenario: Sidebar position
- **WHEN** the sidebar renders for an authenticated, fully set-up user
- **THEN** the "Hooks" entry SHALL appear between the "Sessions" entry and the "Installed" entry, in that order

### Requirement: Per-agent tab visibility follows enabled agents
The system SHALL render a per-agent tab control on the Hooks page only when more than one agent is enabled, and SHALL render the single agent's view directly without a tab control when exactly one agent is enabled.

#### Scenario: Two or more agents enabled
- **WHEN** the user has both Claude and Copilot enabled
- **THEN** the page SHALL render an `AppTabs` with one tab per enabled agent, in the same agent order used by the Browse Plugins, Installed Plugins, and Session History pages

#### Scenario: Exactly one agent enabled
- **WHEN** the user has exactly one agent enabled
- **THEN** the page SHALL render that agent's hooks viewer directly without a tab control

#### Scenario: No agents enabled
- **WHEN** no agent is enabled
- **THEN** the page SHALL render an `AppEmptyState` directing the user to enable an agent in Settings and SHALL NOT call any backend hook-listing method

### Requirement: Two-pane layout with source tree and hook detail
The system SHALL render the Hooks page as a two-pane layout where the left pane is a collapsible Source → Event → Hook tree and the right pane is the selected hook's detail view.

#### Scenario: Initial render with no hook selected
- **WHEN** the user opens the Hooks page
- **THEN** the left pane SHALL render the source tree and the right pane SHALL render an empty state prompting the user to select a hook

#### Scenario: Selecting a hook populates the right pane
- **WHEN** the user clicks a hook row in the left pane
- **THEN** the right pane SHALL render that hook's detail view and the selected row SHALL display a visually distinct selected state

#### Scenario: Switching agents resets the selection
- **WHEN** the user switches the active agent tab
- **THEN** the previously selected hook SHALL be cleared and the right pane SHALL return to its empty state until a hook in the new agent's tree is selected

### Requirement: Hooks are grouped by source then by event
The system SHALL group hooks first by their source (User / Local / Plugin) and second by their event name, with each group rendered as a collapsible section.

#### Scenario: Top-level groups are sources
- **WHEN** the source tree renders
- **THEN** the top level of the tree SHALL be one collapsible section per non-empty source

#### Scenario: Second level groups are events
- **WHEN** a source group is expanded
- **THEN** the source's hooks SHALL be presented as one collapsible subsection per event name (for example `PreToolUse`, `SessionStart`, `userPromptSubmitted`), with hooks belonging to that event listed inside

#### Scenario: Leaf rows are individual hook entries
- **WHEN** an event subsection is expanded
- **THEN** each hook entry from that source/event combination SHALL appear as a single selectable row

### Requirement: Source ordering and visibility rules
The system SHALL order source groups User → Local → Plugin and SHALL hide any source that yields zero hook entries.

#### Scenario: Standard ordering
- **WHEN** the source tree contains User, Local, and one or more Plugin sources
- **THEN** the User group SHALL render first, the Local group SHALL render second, and the Plugin groups SHALL render last

#### Scenario: Plugin groups are sorted by label
- **WHEN** more than one Plugin source is present
- **THEN** the Plugin source groups SHALL be ordered alphabetically by their label (case-insensitive, ASCII-aware)

#### Scenario: Empty source is hidden
- **WHEN** a source's settings file does not exist, or it parses cleanly but contains zero hook entries after parsing
- **THEN** the source group SHALL NOT render at all in the tree

#### Scenario: Source with parse error still renders
- **WHEN** a source's file exists but fails to parse as JSON
- **THEN** the source group SHALL still render with a single error row inside it carrying the parse-error message, and the rest of the tree SHALL render normally

### Requirement: Plugin hooks are discovered for Claude only
The system SHALL discover Claude plugin hooks by enumerating `~/.claude/plugins/installed_plugins.json` and reading `<installPath>/hooks/hooks.json` for each entry, where `installPath` is the field carried by that file's v2 schema.

#### Scenario: Installed plugin with hooks file is included
- **WHEN** an entry in `installed_plugins.json` carries an `installPath` whose `<installPath>/hooks/hooks.json` exists and parses
- **THEN** that entry SHALL appear as a Plugin source group whose label is `Plugin: <pluginID>@<marketplaceID>`, where `<pluginID>` and `<marketplaceID>` are the left and right halves of the `<pluginID>@<marketplaceID>` key in `installed_plugins.json`

#### Scenario: Installed plugin without a hooks file is omitted
- **WHEN** an entry in `installed_plugins.json` has no `<installPath>/hooks/hooks.json` on disk
- **THEN** no Plugin source group SHALL be rendered for that entry

#### Scenario: Multiple entries for one plugin key produce multiple sources
- **WHEN** a single key in `installed_plugins.json` (for example `foo@bar`) maps to more than one entry (different `scope` values)
- **THEN** each entry whose `<installPath>/hooks/hooks.json` exists SHALL render as its own Plugin source group, and the label SHALL include the entry's `scope` to disambiguate (for example `Plugin: foo@bar (project)`)

#### Scenario: installPath is the contract; cache or marketplace tree layout is not hardcoded
- **WHEN** the plugin scanner resolves a plugin's hook file
- **THEN** the path SHALL be derived solely from the entry's `installPath` field plus the literal suffix `hooks/hooks.json`, and the scanner SHALL NOT scan `~/.claude/plugins/marketplaces/` or otherwise infer a path from `pluginID`/`marketplaceID`

#### Scenario: Marketplace tree is not loaded
- **WHEN** `~/.claude/plugins/marketplaces/<marketplaceID>/plugins/<pluginID>/hooks/hooks.json` exists alongside an installed entry
- **THEN** the scanner SHALL NOT load the marketplace-tree file; only `<installPath>/hooks/hooks.json` from the installed entry SHALL be loaded

#### Scenario: Git hooks are not scanned
- **WHEN** any path resolution within the plugin scanner would otherwise descend into a `.git/hooks/` segment
- **THEN** the resolution SHALL refuse to load any file under that segment, regardless of its name or contents

#### Scenario: Copilot has no plugin source group
- **WHEN** the Copilot tab renders
- **THEN** the source tree SHALL NOT contain any Plugin source group, irrespective of the contents of `~/.copilot/`

### Requirement: Hook list rows show summary metadata
The system SHALL render each hook list row with summary metadata that can be obtained without further file reads.

#### Scenario: Claude hook row contents
- **WHEN** a Claude hook row renders
- **THEN** the row SHALL show, in this order: the matcher string (or the literal `*` when matcher is empty/absent), the handler `type`, and a one-line digest of the handler's `command` (or `url` for `http`, or `prompt` for `prompt`/`agent` handlers, truncated at the row width)

#### Scenario: Copilot hook row contents
- **WHEN** a Copilot hook row renders
- **THEN** the row SHALL show, in this order: the handler `type`, and a one-line digest of `bash` or `powershell` (whichever is non-empty, preferring the OS-native field when both are set), truncated at the row width
- **AND** the row SHALL NOT render any matcher field

### Requirement: Hook detail pane shows full path, resolved fields, and raw JSON
The system SHALL render the right-pane detail view as three regions: a header with source attribution, a resolved-fields region per handler type, and a read-only raw JSON region.

#### Scenario: Header shows full source path
- **WHEN** a hook is selected
- **THEN** the detail pane header SHALL show the source kind (`User`, `Local`, or `Plugin: <pluginID>@<marketplaceID>`), the event name, and the full absolute filesystem path of the source file
- **AND** the source path SHALL NOT be truncated; if it does not fit on one line it SHALL wrap

#### Scenario: Resolved fields for command handlers
- **WHEN** a Claude `command` or Copilot `command` hook is selected
- **THEN** the resolved-fields region SHALL display each present field of the handler (`command`, `if`, `timeout`, `bash`, `powershell`, `cwd`, `timeoutSec`, `comment`, `shell`, `async`) as a labelled row, omitting absent fields

#### Scenario: Resolved fields for http handlers
- **WHEN** a Claude `http` hook is selected
- **THEN** the resolved-fields region SHALL display `url`, `headers`, and `allowedEnvVars` as labelled rows when present

#### Scenario: Resolved fields for mcp_tool handlers
- **WHEN** a Claude `mcp_tool` hook is selected
- **THEN** the resolved-fields region SHALL display `server`, `tool`, and `input` as labelled rows when present

#### Scenario: Resolved fields for prompt and agent handlers
- **WHEN** a Claude `prompt` or `agent` hook is selected
- **THEN** the resolved-fields region SHALL display `prompt` and `model` as labelled rows when present

#### Scenario: Raw JSON region is read-only
- **WHEN** the raw JSON region renders
- **THEN** the JSON SHALL be presented as pretty-printed read-only text and SHALL NOT be editable through the UI

### Requirement: Cross-source duplicate is surfaced as a chip, not collapsed
The system SHALL detect when the same hook command string appears across multiple sources for the same agent, surface a chip in the detail pane indicating the duplicate count, and SHALL NOT collapse or hide any duplicate row.

#### Scenario: Duplicate command across User and Plugin sources
- **WHEN** the same `command` string is present in both the User source and one Plugin source for Claude
- **THEN** the detail pane for any of the matching hooks SHALL display a chip reading `appears in N sources` where N is the count of distinct sources containing that command
- **AND** every matching row SHALL still render in its own source group in the left pane

#### Scenario: No duplicate
- **WHEN** the selected hook's `command` string is unique among the agent's loaded hooks
- **THEN** the detail pane SHALL NOT display the duplicate chip

### Requirement: disableAllHooks is shown as a banner without hiding the list
The system SHALL show a non-blocking banner at the top of an agent's hooks view when any of that agent's loaded sources has top-level `"disableAllHooks": true`, and SHALL continue to render the full hook list.

#### Scenario: Flag set in the user settings file
- **WHEN** `~/.claude/settings.json` carries `"disableAllHooks": true`
- **THEN** the Claude tab SHALL render a banner reading `Hooks are globally disabled by <source path>`, where `<source path>` is the absolute path of the file that sets the flag
- **AND** the source tree SHALL still render every loaded hook as if the flag were not set

#### Scenario: Flag not set
- **WHEN** none of the loaded sources for the agent carry `"disableAllHooks": true`
- **THEN** the banner SHALL NOT render

### Requirement: Open settings file action delegates to the os-path-opener capability
The system SHALL provide an "Open settings file" action in the detail pane that reveals the source file in the OS file manager via the existing `os-path-opener` capability.

#### Scenario: User clicks Open settings file
- **WHEN** the user clicks the "Open settings file" control with a hook selected
- **THEN** the system SHALL invoke the `os-path-opener` capability with the source file's absolute path
- **AND** the system SHALL NOT shell out to any OS-specific command directly from `internal/hooks/`

### Requirement: Service accepts ListOpts with reserved Workspace field
The backend SHALL expose a service method `ListHooks(agentID string, opts ListOpts) (Listing, error)` whose `ListOpts` struct contains a `Workspace string` field reserved for a future per-workspace scope.

#### Scenario: Empty Workspace in v1
- **WHEN** the frontend calls `ListHooks` with `opts.Workspace == ""`
- **THEN** the service SHALL return a `Listing` populated according to the user-level scope rules

#### Scenario: Non-empty Workspace in v1 is ignored, not rejected
- **WHEN** the frontend calls `ListHooks` with a non-empty `opts.Workspace`
- **THEN** the service SHALL behave identically to the empty-Workspace case in v1 and SHALL NOT return an error

### Requirement: Copilot user-level view shows an explanatory empty state
The system SHALL render an `AppEmptyState` on the Copilot tab in v1 explaining that Copilot CLI hooks are project-scoped and that workspace selection is a future enhancement.

#### Scenario: Copilot tab renders in v1
- **WHEN** the Copilot tab renders with `opts.Workspace == ""`
- **THEN** the page SHALL render an empty state describing that Copilot CLI hooks are project-scoped and SHALL NOT render a source tree
- **AND** no `~/.copilot/*` file SHALL be read for hook discovery

### Requirement: Hooks viewer is strictly read-only
The system SHALL only read source files and SHALL NOT modify, delete, rename, or write any file under `~/.claude/`, `~/.copilot/`, or any plugin directory; the system SHALL also NOT invoke any agent CLI subcommand from the hooks viewer code path.

#### Scenario: Listing hooks performs no writes
- **WHEN** the system lists hooks for any agent
- **THEN** no file under any agent or plugin directory SHALL be created, modified, or deleted

#### Scenario: No agent CLI invocation
- **WHEN** the system lists hooks for any agent
- **THEN** no `claude` or `copilot` subcommand SHALL be spawned by the hooks viewer code path

### Requirement: Errors loading any source are non-fatal
The system SHALL surface per-source errors (file present but malformed, plugin path missing despite being installed, permission denied) as non-fatal warnings on the page and SHALL render every other source successfully.

#### Scenario: One source fails to parse, others succeed
- **WHEN** the User source parses successfully but a Plugin source's `hooks.json` fails JSON parsing
- **THEN** the User source SHALL render normally
- **AND** the Plugin source SHALL render with an inline error row carrying the parse-error message
- **AND** the page SHALL NOT show a global error state

#### Scenario: All sources fail
- **WHEN** every source for an agent fails to load
- **THEN** the page SHALL render an empty state for that agent listing each source path and its error message

### Requirement: Streaming reads tolerate unknown handler types and unknown fields
The Claude parser SHALL accept unknown handler `type` values and unknown fields without failing, preserving the original JSON for the raw JSON region.

#### Scenario: Unknown handler type
- **WHEN** a Claude hook entry has a `type` value not in the known set (`command`, `http`, `mcp_tool`, `prompt`, `agent`)
- **THEN** the row SHALL render with the type string verbatim and the resolved-fields region SHALL be empty
- **AND** the raw JSON region SHALL still display the entry's full JSON

#### Scenario: Unknown extra fields
- **WHEN** a hook entry contains fields not enumerated by this spec
- **THEN** those fields SHALL appear in the raw JSON region and SHALL NOT cause the parser to drop the entry

### Requirement: Disable-flag and source detection are honored only from loaded sources
The system SHALL only honor the `disableAllHooks` flag from sources it actually reads (User, Local, Plugin in v1) and SHALL NOT infer the flag from sources outside the v1 scope (Project, Managed, Cloud).

#### Scenario: disableAllHooks set in a non-loaded source
- **WHEN** the flag is set only in a project-scope or managed-scope file that v1 does not load
- **THEN** the banner SHALL NOT render
- **AND** the page SHALL still display all loaded hooks normally

