## ADDED Requirements

### Requirement: Settings page is reachable from the sidebar footer
The system SHALL expose a Settings page at `/settings` that is reached via an icon button in the left sidebar's footer region, not via a top-level sidebar navigation entry.

#### Scenario: User opens settings from sidebar footer
- **WHEN** the user clicks the settings icon button in the sidebar footer
- **THEN** the application SHALL navigate to `/settings` and the main content area SHALL render the Settings page without a page reload

#### Scenario: Settings is not present in the top-level sidebar navigation list
- **WHEN** the user inspects the top-level navigation list in the sidebar
- **THEN** the list SHALL contain entries for the primary workflow pages (Browse, Installed) and SHALL NOT contain a Settings entry at the same visual level

#### Scenario: Settings icon button has an accessible label
- **WHEN** the user focuses or hovers over the sidebar footer settings button
- **THEN** a tooltip or accessible name identifying the control as "Settings" SHALL be presented

### Requirement: Settings page renders Providers and Marketplaces as tabs
The system SHALL render the Settings page as a tabbed layout containing at least a Providers tab and a Marketplaces tab, with the active tab rendering the corresponding configuration surface inside the Settings page.

#### Scenario: Marketplaces tab is active by default
- **WHEN** the user navigates to `/settings` without a `tab` query parameter
- **THEN** the Marketplaces tab SHALL be marked active and its content SHALL be rendered

#### Scenario: Providers tab shows the provider configuration surface
- **WHEN** the Providers tab is active
- **THEN** the tab content area SHALL display the list of supported AI providers with their configured status, equivalent to what the standalone Providers page previously displayed

#### Scenario: Marketplaces tab shows the marketplace configuration surface
- **WHEN** the Marketplaces tab is active
- **THEN** the tab content area SHALL display the list of registered marketplaces with add, remove, and refresh controls, equivalent to what the standalone Marketplaces page previously displayed

#### Scenario: Switching tabs does not reload the page
- **WHEN** the user clicks a tab that is not currently active
- **THEN** the Settings page SHALL switch the active tab and render the new tab's content in place, without a full page reload

### Requirement: Tab selection is reflected in the URL and supports deep linking
The system SHALL synchronize the active Settings tab with a `tab` query parameter on the `/settings` URL so that tab selection is shareable via URL and survives browser back/forward navigation.

#### Scenario: Selecting a tab updates the URL
- **WHEN** the user selects the Marketplaces tab while on `/settings`
- **THEN** the URL SHALL be updated to `/settings?tab=marketplaces` using router push or replace semantics

#### Scenario: Deep link opens the requested tab
- **WHEN** the user navigates directly to `/settings?tab=marketplaces`
- **THEN** the Marketplaces tab SHALL be active on first render without the user needing to click a tab

#### Scenario: Unknown tab value falls back to default
- **WHEN** the user navigates to `/settings?tab=unknown`
- **THEN** the Marketplaces tab SHALL be activated as the fallback and the URL MAY be normalized to `/settings` or `/settings?tab=marketplaces`

### Requirement: Legacy routes redirect into the Settings page
The system SHALL preserve the previously exposed `/providers` and `/marketplaces` routes as redirects into the Settings page so that existing bookmarks and references continue to work.

#### Scenario: Legacy providers route redirects
- **WHEN** the user navigates to `/providers`
- **THEN** the router SHALL redirect to `/settings?tab=providers` and the Providers tab SHALL be active

#### Scenario: Legacy marketplaces route redirects
- **WHEN** the user navigates to `/marketplaces`
- **THEN** the router SHALL redirect to `/settings?tab=marketplaces` and the Marketplaces tab SHALL be active
