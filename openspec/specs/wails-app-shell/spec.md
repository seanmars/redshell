## Purpose

Define the Wails v2 desktop application shell (window, sidebar navigation, frontend build pipeline, and Go-to-frontend binding layer) that hosts the plugin marketplace experience.
## Requirements
### Requirement: Wails app initializes and displays main window
The system SHALL initialize a Wails v2 application with a Vue 3 frontend and display a main window on startup.

#### Scenario: App starts successfully
- **WHEN** the user launches the application binary
- **THEN** a desktop window opens with minimum size 1024x700 and title "RedShell"

#### Scenario: App window has navigation sidebar
- **WHEN** the main window is displayed
- **THEN** a left sidebar SHALL show top-level navigation items in order: Browse Plugins, Sessions, Installed Plugins

#### Scenario: Sidebar has a footer region with a settings button
- **WHEN** the main window is displayed
- **THEN** the left sidebar SHALL contain a footer region at the bottom, distinct from the main navigation list, that contains a settings icon button linking to `/settings`

#### Scenario: App navigates between pages
- **WHEN** the user clicks a navigation item in the sidebar or the settings button in the sidebar footer
- **THEN** the main content area updates to show the corresponding page without page reload

#### Scenario: Default landing route
- **WHEN** the app first renders at the root path `/`
- **THEN** the router SHALL redirect to `/browse` so the user lands on the primary plugin-browsing flow rather than a configuration screen

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

### Requirement: App shell registers a close intercept and tray lifecycle
The system SHALL register an `OnBeforeClose` hook on the Wails application that consults the persisted close-behavior preference, and SHALL start a system tray manager during startup and stop it during shutdown when running on a supported platform (Windows). The `OnBeforeClose` hook SHALL also coordinate with the updater service so an in-progress update can quit the app without triggering the close-behavior prompt or the `minimize-to-tray` behavior.

#### Scenario: Tray manager starts during app startup on Windows
- **WHEN** the Wails `OnStartup` hook fires on Windows
- **THEN** the system SHALL start the tray manager in a dedicated goroutine and SHALL pass it the Wails context so menu callbacks can show, hide, and quit the main window

#### Scenario: Tray manager stops during app shutdown
- **WHEN** the Wails `OnShutdown` hook fires
- **THEN** the system SHALL stop the tray manager so the tray icon is removed before the process terminates

#### Scenario: Close hook delegates to the persisted preference
- **WHEN** the Wails `OnBeforeClose` hook fires AND the updater service reports `InProgress() == false`
- **THEN** the hook SHALL read the persisted close-behavior preference and apply the matching behavior: hide the window for `minimize-to-tray`, allow the close for `exit`, or emit the close-behavior prompt event for `unset`

#### Scenario: Tray-driven quit bypasses the close intercept
- **WHEN** the user selects "Quit RedShell" from the tray context menu
- **THEN** the application SHALL set an internal "explicit quit" flag and call the Wails runtime quit so that the `OnBeforeClose` hook (if invoked) returns `false` and lets the process terminate without emitting the prompt or applying `minimize-to-tray`

#### Scenario: In-progress update bypasses the close intercept
- **WHEN** the Wails `OnBeforeClose` hook fires AND the updater service reports `InProgress() == true`
- **THEN** the hook SHALL return `false` immediately (allowing the close), SHALL NOT consult the persisted close-behavior preference, and SHALL NOT emit the close-behavior prompt event, so the rename swap and child-process spawn can complete cleanly

### Requirement: App shell mounts the close-behavior prompt modal
The system SHALL mount a single instance of the close-behavior prompt modal at the app root so that the modal is available to receive the prompt event from any route.

#### Scenario: Prompt modal is reachable from any route
- **WHEN** the user is on any route (`/browse`, `/installed`, `/settings`, etc.) and the backend emits the close-behavior prompt event
- **THEN** the modal SHALL render over the current view without requiring navigation, and SHALL not be duplicated by route-level mounts

### Requirement: App shell wires the updater service into startup and shutdown lifecycles

The system SHALL initialize the updater service during `OnStartup` (after preferences and tray are ready), run startup cleanup of stale `.old` and `.partial` files, and bind the updater app wrapper so the frontend can read and act on update state.

#### Scenario: Updater service starts after preferences are loaded

- **WHEN** the Wails `OnStartup` hook fires
- **THEN** the updater service SHALL be started after the preferences service is available, SHALL receive the Wails context for `runtime.EventsEmit`, and SHALL register itself as an observer of the preferences `autoUpdate` block so source / interval / enabled changes take effect without restart

#### Scenario: Stale artifact cleanup runs before service start

- **WHEN** the application starts
- **THEN** the updater package's `CleanupStale()` function SHALL run before the ticker is registered, deleting any `*.old` or `*.partial` files in the running exe's directory whose basename matches the running exe (with `.exe.old` / `.exe.partial` suffixes)

#### Scenario: Updater app wrapper is bound to Wails

- **WHEN** the Wails options struct is constructed
- **THEN** the `Bind` slice SHALL include an `*app.UpdaterApp` instance exposing methods for `CheckNow`, `PeekBothSources`, `InstallAvailable`, `SkipVersion`, `Unskip`, and `GetState`

#### Scenario: `OnShutdown` does not need to stop the updater explicitly

- **WHEN** the Wails `OnShutdown` hook fires during normal exit (not during an in-progress update)
- **THEN** the updater service SHALL be eligible for goroutine cleanup via context cancellation propagated from the Wails context, and SHALL NOT require a separate explicit `Stop()` call from `OnShutdown`

### Requirement: Production builds enforce a single running instance

In production builds the system SHALL allow at most one running instance of the application. When a second launch is attempted while an instance is already running, the system SHALL bring the existing instance's window to the foreground — including when that window is hidden in the system tray or minimized — and the second process SHALL terminate itself without opening a new window. This enforcement SHALL be gated on the production build tag so that development and tooling builds (`wails dev`, plain `go build`/`go run`/`go test`/`go vet`) impose no instance limit.

#### Scenario: Second launch in production raises the existing window and exits

- **WHEN** the production binary is launched while another instance is already running
- **THEN** the already-running instance SHALL bring its main window to the foreground and the newly launched process SHALL exit without creating a second window or tray icon

#### Scenario: Second launch reveals a tray-hidden or minimized instance

- **WHEN** the running instance has been minimized to the tray or minimized to the taskbar AND the production binary is launched again
- **THEN** the running instance SHALL unhide and unminimize its window and bring it to the foreground, reusing the same window-show path as the tray "Show RedShell" action

#### Scenario: Development builds allow multiple instances

- **WHEN** the application is run via `wails dev` or any non-production build (no `production` build tag)
- **THEN** the single-instance lock SHALL NOT be configured and launching additional instances SHALL be permitted

#### Scenario: Single-instance identity is stable across releases

- **WHEN** the single-instance lock is configured
- **THEN** it SHALL use a fixed application-scoped unique identifier that does not change between releases, so two different installed versions cannot run simultaneously

### Requirement: Updater relaunch waits for the previous instance before acquiring the lock

The system SHALL keep the auto-update relaunch compatible with single-instance enforcement. The updater SHALL launch the replacement binary with a `--wait-parent-pid=<pid>` argument identifying the outgoing process. Before initializing the Wails runtime (and therefore before acquiring the single-instance lock), the relaunched binary SHALL wait, up to a bounded timeout, for the identified parent process to exit, then proceed to start normally. A binary launched without this argument SHALL start immediately without waiting.

#### Scenario: Post-update relaunch waits for the old process then becomes the sole instance

- **WHEN** the updater spawns the replacement binary with `--wait-parent-pid=<old-pid>` and then quits the outgoing process
- **THEN** the replacement binary SHALL wait for the outgoing process to exit (up to the bounded timeout) before initializing the Wails runtime, and SHALL then acquire the single-instance lock cleanly so exactly one instance is running after the swap completes

#### Scenario: Bounded wait does not hang startup

- **WHEN** the relaunched binary is told to wait for a parent PID that does not exit within the timeout
- **THEN** the binary SHALL stop waiting at the timeout and continue startup rather than blocking indefinitely

#### Scenario: Normal launch does not wait

- **WHEN** the binary is launched without a `--wait-parent-pid` argument
- **THEN** it SHALL start immediately without performing any parent-process wait

