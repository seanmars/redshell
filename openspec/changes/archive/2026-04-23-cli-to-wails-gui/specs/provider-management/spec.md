## ADDED Requirements

### Requirement: Display supported AI providers and their status
The system SHALL display a list of supported AI providers with their configuration status.

#### Scenario: Providers page shows all supported providers
- **WHEN** the user navigates to the Providers page
- **THEN** the page SHALL list Claude Code and GitHub Copilot with their config directory paths

#### Scenario: Provider shows configured status
- **WHEN** the provider's config directory exists on the filesystem
- **THEN** the provider card SHALL show a "Configured" badge in green

#### Scenario: Provider shows not-configured status
- **WHEN** the provider's config directory does not exist
- **THEN** the provider card SHALL show a "Not Configured" badge in gray with a hint message

### Requirement: Display provider configuration paths
The system SHALL display the filesystem paths associated with each provider.

#### Scenario: Claude Code paths are displayed
- **WHEN** the user views the Claude Code provider card
- **THEN** the card SHALL show: config dir `~/.claude`, commands dir `~/.claude/commands`, skills dir `~/.claude/skills`

#### Scenario: GitHub Copilot paths are displayed
- **WHEN** the user views the GitHub Copilot provider card
- **THEN** the card SHALL show: config dir `~/.copilot`, commands dir `~/.github/prompts`, skills dir `~/.github/skills`

### Requirement: API token configuration
The system SHALL allow users to set API tokens for GitHub and GitLab to avoid rate limiting when fetching marketplace data.

#### Scenario: User sets GitHub token
- **WHEN** the user enters a GitHub token in the settings field and saves
- **THEN** the token is stored in the app's config and used for subsequent GitHub API calls

#### Scenario: Environment variable token fallback
- **WHEN** no token is set in app config but GITHUB_TOKEN environment variable exists
- **THEN** the system SHALL use the environment variable value for GitHub API calls
