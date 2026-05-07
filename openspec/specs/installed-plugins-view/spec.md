## ADDED Requirements

### Requirement: Display installed plugins per agent
The system SHALL read and display all installed plugins for each AI agent.

#### Scenario: Installed plugins list shows Claude Code plugins
- **WHEN** the user navigates to the Installed Plugins page
- **THEN** plugins listed in `~/.claude/plugins/installed_plugins.json` SHALL be displayed under the Claude Code section

#### Scenario: Installed plugins list shows GitHub Copilot plugins
- **WHEN** the user navigates to the Installed Plugins page
- **THEN** plugins listed in `~/.copilot/config.json` SHALL be displayed under the GitHub Copilot section

#### Scenario: Empty installed plugins state
- **WHEN** no plugins are installed for an agent
- **THEN** that agent's section SHALL show an empty state message with a link to Browse Plugins

### Requirement: View installed plugin details
The system SHALL display metadata for each installed plugin.

#### Scenario: Plugin card shows details
- **WHEN** an installed plugin is displayed
- **THEN** the card SHALL show: plugin name, version (if available), installed date, and source marketplace

### Requirement: Uninstall a plugin
The system SHALL allow users to uninstall a plugin from an agent.

#### Scenario: User uninstalls a plugin
- **WHEN** the user clicks Uninstall on a plugin card and confirms the dialog
- **THEN** the system SHALL call `<agent> plugin uninstall <pluginID>` to perform the removal; the plugin files and registry entry are managed by the agent CLI

#### Scenario: Uninstall confirmation dialog
- **WHEN** the user clicks Uninstall
- **THEN** a confirmation dialog SHALL show the plugin name and agent before proceeding

#### Scenario: Uninstall updates the list immediately
- **WHEN** uninstall completes successfully
- **THEN** the plugin card SHALL be removed from the list without requiring a page refresh

### Requirement: Agent tab navigation for installed plugins
The system SHALL organize installed plugins by agent using a tab or segmented control.

#### Scenario: Tab switches between agents
- **WHEN** the user clicks the Claude Code or GitHub Copilot tab
- **THEN** only that agent's installed plugins are shown in the main area
