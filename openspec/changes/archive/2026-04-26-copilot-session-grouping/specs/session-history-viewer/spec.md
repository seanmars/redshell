## REMOVED Requirements

### Requirement: Copilot session list is a flat list
**Reason**: Copilot listing is being aligned with Claude's grouped layout to give the Session History viewer a single, consistent per-agent UX. The flat ordering provided no project context and forced the user to scan summaries to locate sessions in a known repo.

**Migration**: The grouping requirement below replaces it. The wire shape (`Listing.Kind`) flips from `"flat"` to `"groups"` for Copilot; no on-disk migration is needed because session files are read-only and untouched. Frontend code already supports the `groups` shape; the `flat` arm of `SessionList.vue` remains as inert generic infrastructure and may be reused by a future agent.

## ADDED Requirements

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
