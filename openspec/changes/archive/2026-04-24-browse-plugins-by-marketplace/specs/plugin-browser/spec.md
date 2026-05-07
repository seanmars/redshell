## MODIFIED Requirements

### Requirement: Browse plugins from all registered marketplaces
The system SHALL fetch all plugins from the registered marketplaces and display them on the Browse Plugins page grouped by their source marketplace.

#### Scenario: Plugin list loads on page open
- **WHEN** the user navigates to the Browse Plugins page
- **THEN** the system SHALL fetch the marketplace registry and fetch plugins from every registered marketplace, and SHALL render one section per registered marketplace

#### Scenario: Section header shows marketplace identity
- **WHEN** a marketplace section is rendered
- **THEN** the section header SHALL show the marketplace display name (falling back to its ID when no provider-specific name is available)

#### Scenario: Plugins show metadata
- **WHEN** a plugin card is displayed inside a marketplace section
- **THEN** it SHALL show: plugin name, description, author, category, and target provider

#### Scenario: Loading state is shown during fetch
- **WHEN** plugins are being fetched from remote marketplaces
- **THEN** a loading indicator SHALL be displayed

#### Scenario: No marketplaces registered
- **WHEN** the marketplace registry is empty
- **THEN** the page SHALL show a single page-level empty state whose copy explains that no marketplaces are registered and SHALL provide a link to the Marketplaces page

#### Scenario: Marketplace registered but returns no plugins
- **WHEN** a marketplace is registered and its plugin fetch succeeds with zero results
- **THEN** its section SHALL still be rendered with a "no plugins available in this marketplace" message instead of being omitted

#### Scenario: Per-marketplace fetch error is surfaced inline
- **WHEN** fetching plugins from a specific marketplace fails
- **THEN** that marketplace's section SHALL display the error message for the failing provider inline, without removing the section from the page

#### Scenario: All plugins shown regardless of provider
- **WHEN** the Browse Plugins page is rendered
- **THEN** plugins targeting every provider (e.g. Claude Code, GitHub Copilot) SHALL be displayed together within each marketplace section with no provider filter applied

## REMOVED Requirements

### Requirement: Filter plugins by provider
**Reason:** Provider-based filter tabs on Browse Plugins duplicate information already shown on each plugin card and conflict with the new marketplace-grouped layout. Users continue to select the target provider when installing via the install confirmation modal.

**Migration:** No user migration required. The install flow still asks the user to choose the target provider (`claude` or `copilot`) in the install confirmation dialog, so plugins installed before and after this change follow the same installation path. Any code or scenarios that relied on `providerFilter` state must be removed along with the tabs.
