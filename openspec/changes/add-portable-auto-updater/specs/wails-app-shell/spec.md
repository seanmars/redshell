## MODIFIED Requirements

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

## ADDED Requirements

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
