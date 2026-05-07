## Purpose

Browse, filter, and install plugins from registered marketplaces (GitHub/GitLab) through a unified page, covering fetch of marketplace manifests, selection UX, and the install flow hand-off to each agent CLI.
## Requirements
### Requirement: Browse plugins from all registered marketplaces
The system SHALL display each registered marketplace's plugin list on the Browse Plugins page by reading the agent-specific marketplace manifest from a local clone cache of that marketplace's repository, with each marketplace rendered as a collapsible section containing a tabbed view of its claude and copilot plugins. The page itself SHALL NOT perform network I/O during rendering; remote refresh is a separate, user-triggered action.

#### Scenario: Plugin list loads on page open from local cache
- **WHEN** the user navigates to the Browse Plugins page
- **THEN** the system SHALL fetch the marketplace registry, SHALL read the claude manifest (`.claude-plugin/marketplace.json`) and the copilot manifest (`.github/plugin/marketplace.json`) from each marketplace's local cache directory under `~/.redshell/.cache/`, and SHALL render one collapsible section per registered marketplace
- **AND** the page SHALL NOT issue any git or HTTP request during this read

#### Scenario: Plugin list is driven by manifest entries
- **WHEN** a marketplace's agent manifest is read from the cache
- **THEN** the system SHALL emit one plugin entry per item in the manifest's `plugins` array, using the entry's `name`, `description`, and `source` fields, and SHALL tag the plugin with the agent whose manifest produced it

#### Scenario: Marketplace that ships only one agent's plugins
- **WHEN** a marketplace's cache contains only `.claude-plugin/marketplace.json` and no `.github/plugin/marketplace.json` (or vice versa)
- **THEN** the missing-agent read SHALL produce an error shaped `[<marketplace-id>/<agent>] cache missing; click Refresh to re-clone` for that one agent, leaving the other agent's tab populated normally

#### Scenario: Marketplace cache is missing entirely
- **WHEN** a registered marketplace has no cache directory at all (e.g. the user manually deleted it, or the directory was never created)
- **THEN** both agent reads SHALL produce the cache-missing error and the section body SHALL display the same error on each tab, prompting the user to click Refresh

#### Scenario: Section header shows marketplace identity
- **WHEN** a marketplace section is rendered
- **THEN** the section header SHALL show the marketplace display name (falling back to its ID when no agent-specific name is available) and SHALL act as a toggle that expands or collapses the section body

#### Scenario: Section default state is expanded
- **WHEN** a marketplace section is rendered for the first time after the page loads
- **THEN** the section SHALL default to the expanded state so plugins are visible without a user click

#### Scenario: Plugins show metadata
- **WHEN** a plugin card is displayed inside a marketplace section
- **THEN** it SHALL show: plugin name, description, author, category, and target agent

#### Scenario: Loading state is shown during cache read
- **WHEN** the page is reading manifests from the cache (or a Refresh is in flight)
- **THEN** a loading indicator SHALL be displayed and user input on the affected sections SHALL be visually deferred

#### Scenario: No marketplaces registered
- **WHEN** the marketplace registry is empty
- **THEN** the page SHALL show a single page-level empty state whose copy explains that no marketplaces are registered and SHALL provide a link to the Marketplaces page

#### Scenario: Marketplace registered but cache contains empty manifests
- **WHEN** a marketplace is registered, its cache exists, and both agent manifest files parse cleanly with empty `plugins` arrays
- **THEN** its section SHALL still be rendered with a "no plugins available in this marketplace" message instead of being omitted

#### Scenario: Per-marketplace cache read error is surfaced inline
- **WHEN** reading a marketplace's agent manifest from the cache fails for a reason other than file-missing (parse error, I/O error)
- **THEN** that marketplace's section SHALL display the error message scoped to the affecting agent tab, without removing the section from the page

### Requirement: Filter plugins per marketplace section by agent
The system SHALL provide an in-section tab control inside each marketplace section that filters that section's plugin list to a single agent at a time.

#### Scenario: Default agent tab
- **WHEN** a marketplace section is first rendered
- **THEN** its agent tab SHALL default to "Claude Code"

#### Scenario: User switches agent tab within a section
- **WHEN** the user clicks the "GitHub Copilot" tab inside a marketplace section
- **THEN** the section's plugin grid SHALL re-render to show only plugins whose target agent is `copilot`, and the claude tab content SHALL be hidden

#### Scenario: Tab state is independent per section
- **WHEN** the user selects a different agent tab in marketplace A and then scrolls to marketplace B
- **THEN** marketplace B's tab SHALL retain its own selection (or its default) independently of marketplace A

#### Scenario: Active tab with no plugins shows an empty message
- **WHEN** the selected agent tab has zero plugins for this marketplace and no fetch error for that agent
- **THEN** the section body SHALL show "No plugins available for this agent in this marketplace"

#### Scenario: Active tab with a fetch error shows the error
- **WHEN** the selected agent tab's manifest read failed
- **THEN** the section body SHALL show the error text for that agent and SHALL NOT show the claude fallback list

### Requirement: Refresh marketplace data on demand
The system SHALL provide a user-triggered Refresh action that updates each marketplace's local cache by performing a shallow git fetch and reset against the cached working tree, and the Browse Plugins page SHALL surface refresh failures without discarding cached data.

#### Scenario: User clicks Refresh
- **WHEN** the user clicks the Refresh button on the Browse Plugins page
- **THEN** the system SHALL run a refresh against every registered marketplace's cache (per `marketplace-management` Refresh requirement) and, on completion, SHALL re-read the cache to update the displayed plugin lists

#### Scenario: Refresh succeeds for all marketplaces
- **WHEN** every marketplace's git fetch + reset completes without error
- **THEN** the page SHALL clear any previously displayed refresh warnings and re-render plugin lists from the now-current cache

#### Scenario: Refresh fails for one marketplace but cache exists
- **WHEN** a marketplace's refresh fails (network down, auth lost, remote unreachable) and that marketplace's cache directory still contains valid manifests
- **THEN** the page SHALL render that marketplace's section using the existing (stale) cache content AND SHALL display a per-marketplace warning at the section header shaped `[<marketplace-id>] git refresh: <reason>`

#### Scenario: Refresh fails and cache is also missing
- **WHEN** a marketplace's refresh fails and no usable cache content exists
- **THEN** the section SHALL render the same cache-missing error inside both tabs and SHALL display the refresh-failure warning at the section header

#### Scenario: Refresh button is disabled while in flight
- **WHEN** a Refresh is in progress
- **THEN** the Refresh button SHALL be disabled and a loading indicator SHALL accompany it

### Requirement: Select and install multiple plugins
The system SHALL allow users to select multiple plugins and install them in a single operation.

#### Scenario: User selects plugins
- **WHEN** the user clicks a plugin card
- **THEN** the card shows a selected state with a checkmark, and the install button shows the count of selected plugins

#### Scenario: Install confirmation
- **WHEN** the user clicks the Install button with plugins selected
- **THEN** a confirmation dialog SHALL show the list of selected plugins, the target agent, and the target directory before proceeding

#### Scenario: Successful installation
- **WHEN** the user confirms the install
- **THEN** the system SHALL first ensure the plugin's marketplace is registered via `<agent> plugin marketplace add <url>` (skipping if already registered), then call `<agent> plugin install <installName>` for each selected plugin; installation log SHALL be streamed to the GUI in real time

#### Scenario: Already-installed plugin is indicated
- **WHEN** a plugin is already installed for the selected agent
- **THEN** the plugin card SHALL show an "Installed" badge and cannot be selected for re-installation
