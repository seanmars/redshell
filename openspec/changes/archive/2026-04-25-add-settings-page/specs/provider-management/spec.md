## MODIFIED Requirements

### Requirement: Display supported AI providers and their status
The system SHALL display a list of supported AI providers with their configuration status inside the Providers tab of the Settings page.

#### Scenario: Providers tab shows all supported providers
- **WHEN** the user opens the Providers tab on the Settings page
- **THEN** the tab content SHALL list Claude Code and GitHub Copilot with their config directory paths

#### Scenario: Provider shows configured status
- **WHEN** the provider's config directory exists on the filesystem
- **THEN** the provider card SHALL show a "Configured" badge in green

#### Scenario: Provider shows not-configured status
- **WHEN** the provider's config directory does not exist
- **THEN** the provider card SHALL show a "Not Configured" badge in gray with a hint message
