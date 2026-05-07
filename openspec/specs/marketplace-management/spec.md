## Purpose

Manage the registry of plugin marketplaces (git repositories) that the application reads plugin manifests from, including registration, removal, ID generation, on-disk cache layout, and refresh of cached clones.

## Requirements

### Requirement: List registered marketplaces
The system SHALL read and display all marketplaces registered in `~/.redshell/marketplace.json` inside the Marketplaces tab of the Settings page.

#### Scenario: Marketplace list is displayed
- **WHEN** the user opens the Marketplaces tab on the Settings page
- **THEN** all registered marketplaces are shown as cards with name, URL, and added date

#### Scenario: Empty marketplace list
- **WHEN** no marketplaces are registered
- **THEN** the Marketplaces tab SHALL show an empty state with a prompt to add the first marketplace

### Requirement: Add a new marketplace
The system SHALL allow users to register a new marketplace by providing a git repository URL, and SHALL create a persistent local clone of that repository as part of the atomic add operation.

#### Scenario: User adds a valid marketplace URL
- **WHEN** the user enters a git repository URL and confirms
- **THEN** the system SHALL validate the URL format, generate a marketplace ID, perform a shallow (`--depth=1`) `git clone` into the marketplace's cache directory under `~/.redshell/.cache/`, read the agent display names from the on-disk manifests, and append the entry to `~/.redshell/marketplace.json`

#### Scenario: Duplicate marketplace is rejected
- **WHEN** the user submits a URL that generates an ID already present in the registry
- **THEN** the system SHALL display an error message, SHALL NOT save the duplicate, and SHALL NOT touch any existing cache directory

#### Scenario: Clone fails during add
- **WHEN** the initial `git clone` fails (network error, auth error, unreachable remote)
- **THEN** the system SHALL surface the git error to the user, SHALL NOT append the entry to the registry, and SHALL remove any partially-created cache directory

#### Scenario: Marketplace names are fetched from the cache on add
- **WHEN** a valid URL is submitted and the clone succeeds
- **THEN** the system SHALL read each agent's manifest file from the new cache directory and populate the registry entry's `name` map with any discovered display names

### Requirement: Remove a marketplace
The system SHALL allow users to remove a registered marketplace and SHALL delete the marketplace's local cache directory as part of the same operation.

#### Scenario: User removes a marketplace
- **WHEN** the user clicks Remove on a marketplace card and confirms the dialog
- **THEN** the marketplace entry SHALL be removed from `~/.redshell/marketplace.json`, the marketplace's cache directory under `~/.redshell/.cache/` SHALL be deleted, and the list updates immediately

#### Scenario: Removal is confirmed before executing
- **WHEN** the user clicks Remove
- **THEN** a confirmation dialog SHALL appear before the entry and cache are deleted

#### Scenario: Cache delete fails during remove
- **WHEN** the registry entry is removed successfully but the cache directory delete fails (e.g. a file is locked by another process)
- **THEN** the registry removal SHALL still be considered successful and the system SHALL surface a non-fatal warning about the orphaned cache directory

### Requirement: Marketplace ID generation
The system SHALL generate a unique ID from a git repository URL using the format `hostname::owner@repo`.

#### Scenario: GitHub URL generates correct ID
- **WHEN** URL is `https://github.com/owner/repo`
- **THEN** the generated ID SHALL be `github.com::owner@repo`

#### Scenario: GitLab URL generates correct ID
- **WHEN** URL is `https://gitlab.com/owner/repo`
- **THEN** the generated ID SHALL be `gitlab.com::owner@repo`

### Requirement: Marketplace cache layout
The system SHALL maintain a parallel cache tree under `~/.redshell/.cache/` where each registered marketplace has exactly one subdirectory whose name is derived deterministically from the marketplace ID.

#### Scenario: Cache directory naming
- **WHEN** a marketplace with ID `<host>::<owner>@<repo>` is registered
- **THEN** its cache directory SHALL be `~/.redshell/.cache/<sanitized-id>` where `<sanitized-id>` is the marketplace ID with each character in the set `: / \ * ? " < > |` replaced by `-`

#### Scenario: Cache directory is a complete git clone
- **WHEN** a marketplace's cache directory is populated
- **THEN** it SHALL contain a `.git/` subdirectory produced by `git clone --depth=1` and SHALL contain the checked-out working tree for the default branch

### Requirement: Refresh marketplace caches
The system SHALL provide a Refresh action that updates one or all marketplace caches without unregistering them, using shallow git operations against each cache's existing remote.

#### Scenario: Refresh a single marketplace
- **WHEN** `Refresh(marketplaceID)` is invoked and the marketplace is registered
- **THEN** the system SHALL acquire the per-marketplace cache lock, perform `git fetch --depth=1 origin` followed by `git reset --hard FETCH_HEAD` in that cache directory, and return any git error verbatim

#### Scenario: Refresh all marketplaces
- **WHEN** `RefreshAll()` is invoked
- **THEN** the system SHALL iterate every registered marketplace, invoke Refresh on each, and return the aggregated set of refreshed IDs plus any per-marketplace errors; a failure on one marketplace SHALL NOT abort the loop

#### Scenario: Refresh of a missing cache auto-reclones
- **WHEN** Refresh is invoked on a marketplace whose cache directory does not exist or lacks a `.git/` subdirectory
- **THEN** the system SHALL perform `git clone --depth=1` into the cache directory instead of `git fetch`, and SHALL return success if the clone succeeds

#### Scenario: Concurrent refreshes on the same marketplace are serialized
- **WHEN** two Refresh invocations target the same marketplace ID at overlapping times
- **THEN** the per-marketplace cache lock SHALL serialize them so at most one git operation runs against the cache directory at a time

### Requirement: Update agent-side marketplace registries
The system SHALL provide a single user-triggered action that, for every enabled agent, asks the agent's own CLI to refresh its registered marketplaces by invoking `<agentID> plugin marketplace update`. This is distinct from the existing RedShell-side cache `Refresh` and SHALL NOT modify `~/.redshell/.cache/`.

#### Scenario: User triggers Update from the Marketplaces tab
- **WHEN** the user clicks the "Update" button on the Marketplaces tab
- **THEN** the system SHALL invoke the backend `UpdateAgentMarketplaces` action exactly once and SHALL disable the button until the action resolves

#### Scenario: Action fans out to every enabled agent
- **WHEN** `UpdateAgentMarketplaces` runs and the user has both `claude` and `copilot` enabled
- **THEN** the system SHALL run `claude plugin marketplace update` and `copilot plugin marketplace update`, and SHALL include one outcome entry per agent in the result

#### Scenario: Action skips disabled agents
- **WHEN** `UpdateAgentMarketplaces` runs and an agent is disabled in agent settings
- **THEN** the system SHALL NOT shell out to that agent's CLI and SHALL NOT include an outcome entry for it

#### Scenario: One failing agent does not abort the others
- **WHEN** the CLI invocation for one enabled agent fails (non-zero exit, agent CLI missing, or network error)
- **THEN** the system SHALL continue invoking the remaining enabled agents and SHALL return a result whose outcome list contains a failure entry for the failing agent and success entries for the others

#### Scenario: Live CLI output reaches the frontend
- **WHEN** an agent CLI emits stdout while `UpdateAgentMarketplaces` runs
- **THEN** the system SHALL forward each line, prefixed with the agent ID (e.g. `[claude] ...`), through the `plugin:install-log` Wails event so the frontend can render progress in real time

#### Scenario: Per-agent failure carries a usable error message
- **WHEN** an agent's CLI invocation fails
- **THEN** the corresponding outcome SHALL have `OK` set to false and `Error` set to a non-empty message that includes the agent ID and the CLI's stderr output (or a friendly "agent CLI '<id>' is not installed" message when the binary is absent on `PATH`)

#### Scenario: Update does not touch RedShell's clone cache
- **WHEN** `UpdateAgentMarketplaces` runs to completion
- **THEN** the contents of `~/.redshell/.cache/` SHALL be unchanged and `~/.redshell/marketplace.json` SHALL be unchanged
