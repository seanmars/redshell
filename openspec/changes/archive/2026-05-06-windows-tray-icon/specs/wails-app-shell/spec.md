## ADDED Requirements

### Requirement: App shell registers a close intercept and tray lifecycle
The system SHALL register an `OnBeforeClose` hook on the Wails application that consults the persisted close-behavior preference, and SHALL start a system tray manager during startup and stop it during shutdown when running on a supported platform (Windows).

#### Scenario: Tray manager starts during app startup on Windows
- **WHEN** the Wails `OnStartup` hook fires on Windows
- **THEN** the system SHALL start the tray manager in a dedicated goroutine and SHALL pass it the Wails context so menu callbacks can show, hide, and quit the main window

#### Scenario: Tray manager stops during app shutdown
- **WHEN** the Wails `OnShutdown` hook fires
- **THEN** the system SHALL stop the tray manager so the tray icon is removed before the process terminates

#### Scenario: Close hook delegates to the persisted preference
- **WHEN** the Wails `OnBeforeClose` hook fires
- **THEN** the hook SHALL read the persisted close-behavior preference and apply the matching behavior: hide the window for `minimize-to-tray`, allow the close for `exit`, or emit the close-behavior prompt event for `unset`

#### Scenario: Tray-driven quit bypasses the close intercept
- **WHEN** the user selects "Quit RedShell" from the tray context menu
- **THEN** the application SHALL set an internal "explicit quit" flag and call the Wails runtime quit so that the `OnBeforeClose` hook (if invoked) returns `false` and lets the process terminate without emitting the prompt or applying `minimize-to-tray`

### Requirement: App shell mounts the close-behavior prompt modal
The system SHALL mount a single instance of the close-behavior prompt modal at the app root so that the modal is available to receive the prompt event from any route.

#### Scenario: Prompt modal is reachable from any route
- **WHEN** the user is on any route (`/browse`, `/installed`, `/settings`, etc.) and the backend emits the close-behavior prompt event
- **THEN** the modal SHALL render over the current view without requiring navigation, and SHALL not be duplicated by route-level mounts
