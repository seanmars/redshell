## ADDED Requirements

### Requirement: Browse plugins from all registered marketplaces
The system SHALL fetch and display all available plugins from registered marketplaces.

#### Scenario: Plugin list loads on page open
- **WHEN** the user navigates to the Browse Plugins page
- **THEN** the system SHALL fetch plugins from all registered marketplaces and display them as cards

#### Scenario: Plugins show metadata
- **WHEN** a plugin card is displayed
- **THEN** it SHALL show: plugin name, description, author, category, and source marketplace name

#### Scenario: Loading state is shown during fetch
- **WHEN** plugins are being fetched from remote marketplaces
- **THEN** a loading indicator SHALL be displayed

#### Scenario: No marketplaces registered
- **WHEN** the marketplace registry is empty
- **THEN** the page SHALL show an empty state with a link to the Marketplaces page

### Requirement: Filter plugins by provider
The system SHALL allow users to filter the plugin list by AI provider.

#### Scenario: User selects a provider filter
- **WHEN** the user selects "Claude Code" or "GitHub Copilot" from the provider filter
- **THEN** only plugins compatible with that provider are displayed

#### Scenario: All providers shown by default
- **WHEN** no filter is applied
- **THEN** all plugins from all providers are displayed

### Requirement: Select and install multiple plugins
The system SHALL allow users to select multiple plugins and install them in a single operation.

#### Scenario: User selects plugins
- **WHEN** the user clicks a plugin card
- **THEN** the card shows a selected state with a checkmark, and the install button shows the count of selected plugins

#### Scenario: Install confirmation
- **WHEN** the user clicks the Install button with plugins selected
- **THEN** a confirmation dialog SHALL show the list of selected plugins, the target provider, and the target directory before proceeding

#### Scenario: Successful installation
- **WHEN** the user confirms the install
- **THEN** the system SHALL first ensure the plugin's marketplace is registered via `<provider> plugin marketplace add <url>` (skipping if already registered), then call `<provider> plugin install <installName>` for each selected plugin; installation log SHALL be streamed to the GUI in real time

#### Scenario: Already-installed plugin is indicated
- **WHEN** a plugin is already installed for the selected provider
- **THEN** the plugin card SHALL show an "Installed" badge and cannot be selected for re-installation

### Requirement: GitHub and GitLab API integration for plugin fetching
The system SHALL fetch plugin metadata from GitHub and GitLab repositories via their REST APIs.

#### Scenario: Fetch from GitHub repository
- **WHEN** a marketplace URL points to a GitHub repository
- **THEN** the system SHALL use the GitHub REST API to fetch the marketplace.json and plugin file tree

#### Scenario: Authenticated fetch with token
- **WHEN** a GitHub or GitLab token is configured
- **THEN** API requests SHALL include the Authorization header to avoid rate limiting
