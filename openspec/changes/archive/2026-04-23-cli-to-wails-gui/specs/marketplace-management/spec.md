## ADDED Requirements

### Requirement: List registered marketplaces
The system SHALL read and display all marketplaces registered in `~/.redshell/marketplace.json`.

#### Scenario: Marketplace list is displayed
- **WHEN** the user navigates to the Marketplaces page
- **THEN** all registered marketplaces are shown as cards with name, URL, and added date

#### Scenario: Empty marketplace list
- **WHEN** no marketplaces are registered
- **THEN** the page SHALL show an empty state with a prompt to add the first marketplace

### Requirement: Add a new marketplace
The system SHALL allow users to register a new marketplace by providing a git repository URL.

#### Scenario: User adds a valid marketplace URL
- **WHEN** the user enters a git repository URL and confirms
- **THEN** the system SHALL validate the URL format, generate a marketplace ID, and append the entry to `~/.redshell/marketplace.json`

#### Scenario: Duplicate marketplace is rejected
- **WHEN** the user submits a URL that generates an ID already present in the registry
- **THEN** the system SHALL display an error message and not save the duplicate

#### Scenario: Marketplace names are fetched on add
- **WHEN** a valid URL is submitted
- **THEN** the system SHALL attempt to fetch the marketplace's display names for each provider from the remote repository

### Requirement: Remove a marketplace
The system SHALL allow users to remove a registered marketplace.

#### Scenario: User removes a marketplace
- **WHEN** the user clicks Remove on a marketplace card and confirms the dialog
- **THEN** the marketplace entry SHALL be removed from `~/.redshell/marketplace.json` and the list updates immediately

#### Scenario: Removal is confirmed before executing
- **WHEN** the user clicks Remove
- **THEN** a confirmation dialog SHALL appear before the entry is deleted

### Requirement: Marketplace ID generation
The system SHALL generate a unique ID from a git repository URL using the format `hostname::owner@repo`.

#### Scenario: GitHub URL generates correct ID
- **WHEN** URL is `https://github.com/owner/repo`
- **THEN** the generated ID SHALL be `github.com::owner@repo`

#### Scenario: GitLab URL generates correct ID
- **WHEN** URL is `https://gitlab.com/owner/repo`
- **THEN** the generated ID SHALL be `gitlab.com::owner@repo`
