## MODIFIED Requirements

### Requirement: Display supported AI providers and their status
The system SHALL display a list of supported AI providers with their installed CLI version inside the Providers tab of the Settings page.

#### Scenario: Providers tab shows all supported providers
- **WHEN** the user opens the Providers tab on the Settings page
- **THEN** the tab content SHALL list Claude Code and GitHub Copilot, each with a Directory row, a Configuration row, and a status badge

#### Scenario: Provider shows installed CLI version
- **WHEN** the provider's CLI binary is on the user's PATH and `<cli> --version` returns a string containing a semver triple
- **THEN** the provider card SHALL show that triple (e.g. `2.1.119`) inside the status badge

#### Scenario: Provider shows not-installed icon
- **WHEN** the provider's CLI binary is not on the user's PATH, the version probe times out, or the output contains no semver triple
- **THEN** the provider card SHALL show a warning icon in the status badge position with an accessible label of `Not installed`

#### Scenario: Provider shows hint when dotfile dir is missing
- **WHEN** the provider's home dotfile directory (`~/.claude` or `~/.copilot`) does not exist on the filesystem
- **THEN** the provider card SHALL show the secondary hint message `Install <Label> to enable this provider.`

### Requirement: Display provider configuration paths
The system SHALL display the directory and primary settings file associated with each provider, and each path SHALL be openable in the OS default handler from the card.

#### Scenario: Claude Code paths are displayed
- **WHEN** the user views the Claude Code provider card
- **THEN** the card SHALL show a Directory row labelled `~/.claude` and a Configuration row labelled `~/.claude/settings.json`

#### Scenario: GitHub Copilot paths are displayed
- **WHEN** the user views the GitHub Copilot provider card
- **THEN** the card SHALL show a Directory row labelled `~/.copilot` and a Configuration row labelled `~/.copilot/config.json`

#### Scenario: Clicking the Directory row opens the folder
- **WHEN** the user clicks the Directory row for a provider
- **THEN** the system SHALL invoke the OS default file-manager handler for that directory using the OpenPath capability

#### Scenario: Clicking the Configuration row opens the settings file
- **WHEN** the user clicks the Configuration row for a provider
- **THEN** the system SHALL invoke the OS default handler for that settings file using the OpenPath capability

#### Scenario: Clicking a path that does not exist surfaces an error
- **WHEN** the user clicks a Directory or Configuration row whose target path does not exist on disk
- **THEN** the system SHALL display a toast notification reporting the failure and SHALL NOT crash the UI

## ADDED Requirements

### Requirement: Provider exposes installed CLI version to the frontend
The system SHALL include an installed-CLI version field on each provider returned by the Wails binding.

#### Scenario: Version field present when CLI installed
- **WHEN** `ListProviders` returns and the provider's CLI is installed and prints a parseable version
- **THEN** the returned `Provider` object SHALL have a non-empty `version` field of the form `<major>.<minor>.<patch>`

#### Scenario: Version field empty when CLI not installed
- **WHEN** `ListProviders` returns and the provider's CLI cannot be invoked or prints no parseable version
- **THEN** the returned `Provider` object SHALL have an empty `version` field
