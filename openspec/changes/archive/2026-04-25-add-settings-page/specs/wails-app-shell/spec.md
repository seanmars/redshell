## MODIFIED Requirements

### Requirement: Wails app initializes and displays main window
The system SHALL initialize a Wails v2 application with a Vue 3 frontend and display a main window on startup.

#### Scenario: App starts successfully
- **WHEN** the user launches the application binary
- **THEN** a desktop window opens with minimum size 1024x700 and title "RedShell"

#### Scenario: App window has navigation sidebar
- **WHEN** the main window is displayed
- **THEN** a left sidebar SHALL show top-level navigation items: Browse Plugins, Installed Plugins

#### Scenario: Sidebar has a footer region with a settings button
- **WHEN** the main window is displayed
- **THEN** the left sidebar SHALL contain a footer region at the bottom, distinct from the main navigation list, that contains a settings icon button linking to `/settings`

#### Scenario: App navigates between pages
- **WHEN** the user clicks a navigation item in the sidebar or the settings button in the sidebar footer
- **THEN** the main content area updates to show the corresponding page without page reload

#### Scenario: Default landing route
- **WHEN** the app first renders at the root path `/`
- **THEN** the router SHALL redirect to `/browse` so the user lands on the primary plugin-browsing flow rather than a configuration screen
