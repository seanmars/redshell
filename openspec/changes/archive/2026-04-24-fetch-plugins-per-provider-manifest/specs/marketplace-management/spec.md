## MODIFIED Requirements

### Requirement: Add a new marketplace
The system SHALL allow users to register a new marketplace by providing a git repository URL, and SHALL create a persistent local clone of that repository as part of the atomic add operation.

#### Scenario: User adds a valid marketplace URL
- **WHEN** the user enters a git repository URL and confirms
- **THEN** the system SHALL validate the URL format, generate a marketplace ID, perform a shallow (`--depth=1`) `git clone` into the marketplace's cache directory under `~/.redshell/.cache/`, read the provider display names from the on-disk manifests, and append the entry to `~/.redshell/marketplace.json`

#### Scenario: Duplicate marketplace is rejected
- **WHEN** the user submits a URL that generates an ID already present in the registry
- **THEN** the system SHALL display an error message, SHALL NOT save the duplicate, and SHALL NOT touch any existing cache directory

#### Scenario: Clone fails during add
- **WHEN** the initial `git clone` fails (network error, auth error, unreachable remote)
- **THEN** the system SHALL surface the git error to the user, SHALL NOT append the entry to the registry, and SHALL remove any partially-created cache directory

#### Scenario: Marketplace names are fetched from the cache on add
- **WHEN** a valid URL is submitted and the clone succeeds
- **THEN** the system SHALL read each provider's manifest file from the new cache directory and populate the registry entry's `name` map with any discovered display names

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

## ADDED Requirements

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
