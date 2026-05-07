## ADDED Requirements

### Requirement: Wails app initializes and displays main window
The system SHALL initialize a Wails v2 application with a React frontend and display a main window on startup.

#### Scenario: App starts successfully
- **WHEN** the user launches the application binary
- **THEN** a desktop window opens with minimum size 1024x700 and title "RedShell"

#### Scenario: App window has navigation sidebar
- **WHEN** the main window is displayed
- **THEN** a left sidebar SHALL show navigation items: Providers, Marketplaces, Browse Plugins, Installed Plugins

#### Scenario: App navigates between pages
- **WHEN** the user clicks a navigation item in the sidebar
- **THEN** the main content area updates to show the corresponding page without page reload

### Requirement: Frontend build pipeline integrates with Wails
The system SHALL use Vite + Vue 3 as the frontend build tool, integrated with the Wails build system.

#### Scenario: Development mode hot-reload
- **WHEN** the developer runs `wails dev`
- **THEN** the app launches with hot-reload enabled for frontend changes

#### Scenario: Production build
- **WHEN** the developer runs `wails build`
- **THEN** a single self-contained binary is produced in `build/bin/`

### Requirement: Go backend services are bound to frontend
The system SHALL expose Go service methods to the Vue frontend via Wails bindings.

#### Scenario: TypeScript bindings are auto-generated
- **WHEN** `wails dev` or `wails build` runs
- **THEN** TypeScript type definitions for all bound Go methods are generated in `frontend/wailsjs/go/`

#### Scenario: Vue views use Pinia stores as binding layer
- **WHEN** a Vue view needs to call a Go method
- **THEN** it MUST call via a Pinia store action (not call wailsjs bindings directly from the template)
