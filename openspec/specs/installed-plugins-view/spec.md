## Purpose

The Installed Plugins view lets users see and manage plugins already installed against each enabled AI agent (Claude Code, GitHub Copilot). It reads each agent's own installed-plugins file as the source of truth, organizes results by agent tabs, and exposes per-card actions (uninstall, update) that shell out to the corresponding agent CLI.
## Requirements
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

### Requirement: Update an installed plugin
The system SHALL allow users to update an installed plugin to the latest version published by its source marketplace by invoking the owning agent's `plugin update` subcommand.

#### Scenario: Update button appears on every installed plugin card
- **WHEN** the Installed Plugins page renders a plugin card for any enabled agent
- **THEN** the card SHALL show an **Update** button immediately to the left of the existing **Uninstall** button

#### Scenario: User updates a Claude Code plugin
- **WHEN** the user clicks **Update** on a plugin card whose agent is `claude`
- **THEN** the system SHALL run `claude plugin update <pluginName>@<marketplaceName>` using the same `name@marketplace` identifier the card already uses for uninstall

#### Scenario: User updates a GitHub Copilot plugin
- **WHEN** the user clicks **Update** on a plugin card whose agent is `copilot`
- **THEN** the system SHALL run `copilot plugin update <pluginName>@<marketplaceName>` using the same `name@marketplace` identifier the card already uses for uninstall

#### Scenario: Update streams CLI output through the install-log channel
- **WHEN** an update is in progress
- **THEN** stdout lines from the agent CLI SHALL be emitted on the `plugin:install-log` event so the existing log overlay can display them

#### Scenario: Update success refreshes the installed list
- **WHEN** an update completes without error
- **THEN** the system SHALL re-read the agent's installed-plugins file and update the card's metadata (e.g. version) without requiring a page refresh
- **AND** a success toast SHALL be shown referencing the plugin's `name@marketplace` identifier

#### Scenario: Update failure surfaces error to user
- **WHEN** the agent CLI exits with a non-zero status during update
- **THEN** an error toast SHALL be shown containing the CLI's stderr message
- **AND** the installed list SHALL remain unchanged

#### Scenario: Update button disabled while a request is in flight
- **WHEN** an update request for a given plugin has been dispatched and has not yet resolved
- **THEN** that card's **Update** and **Uninstall** buttons SHALL be disabled until the request resolves, to prevent overlapping CLI invocations against the same plugin

#### Scenario: Update is blocked when the owning agent is disabled
- **WHEN** the user attempts to update a plugin for an agent that is not enabled in settings
- **THEN** the operation SHALL fail with an "agent is disabled" error and SHALL NOT invoke the agent CLI

