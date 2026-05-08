# system-tray Specification

## Purpose
TBD - created by archiving change windows-tray-icon. Update Purpose after archive.
## Requirements
### Requirement: System tray icon is present while the app is running on Windows
The system SHALL display a system tray icon on Windows for the duration of the application's lifetime, from successful startup until the process exits.

#### Scenario: Tray icon appears on startup
- **WHEN** the user launches the RedShell application on Windows
- **THEN** a tray icon SHALL appear in the Windows notification area within a few seconds of the main window opening, displaying the RedShell logo

#### Scenario: Tray icon is removed on exit
- **WHEN** the application process exits (via tray "Quit", via Windows shutdown, or after the user chose `exit` from the close prompt)
- **THEN** the tray icon SHALL be removed from the notification area before the process terminates

#### Scenario: Tray icon is absent on non-Windows platforms
- **WHEN** the application is running on macOS or Linux
- **THEN** no tray icon is created and all tray-related code paths SHALL be no-ops, with the application behaving as if the tray feature were not present

### Requirement: Tray icon left-click toggles the main window
The system SHALL toggle the main window's visibility when the user left-clicks (or double-clicks, depending on platform convention) the tray icon.

#### Scenario: Showing a hidden window from the tray
- **WHEN** the main window is hidden (because the user previously minimized it to the tray) and the user left-clicks the tray icon
- **THEN** the main window SHALL be shown and brought to the foreground with focus

#### Scenario: Hiding a visible window via the tray
- **WHEN** the main window is visible and the user left-clicks the tray icon
- **THEN** the main window SHALL be hidden to the tray (equivalent to choosing `minimize-to-tray`), without exiting the application

### Requirement: Tray icon right-click menu exposes show / toggle / quit
The system SHALL display a context menu when the user right-clicks the tray icon, containing at least the items "Show RedShell", "Close button minimizes to tray" (checkable), and "Quit RedShell".

#### Scenario: Show menu item restores the window
- **WHEN** the user selects the "Show RedShell" item from the tray context menu
- **THEN** the main window SHALL be shown and focused, regardless of its previous visibility state

#### Scenario: Close-to-tray toggle reflects current preference
- **WHEN** the tray context menu is opened
- **THEN** the "Close button minimizes to tray" item SHALL display a checked state if and only if the persisted close-behavior preference is `minimize-to-tray`

#### Scenario: Toggling close-to-tray persists the change
- **WHEN** the user clicks the "Close button minimizes to tray" item while it is unchecked
- **THEN** the persisted close-behavior preference SHALL be updated to `minimize-to-tray` and the menu item SHALL display a checked state on next open

#### Scenario: Toggling close-to-tray off persists exit behavior
- **WHEN** the user clicks the "Close button minimizes to tray" item while it is checked
- **THEN** the persisted close-behavior preference SHALL be updated to `exit` and the menu item SHALL display an unchecked state on next open

#### Scenario: Quit menu item exits regardless of preference
- **WHEN** the user selects the "Quit RedShell" item from the tray context menu
- **THEN** the application SHALL exit cleanly without showing the first-run close-behavior prompt and without applying the `minimize-to-tray` preference

### Requirement: Window close button respects the persisted close-behavior preference
The system SHALL intercept attempts to close the main window via the OS close button (X) or equivalent (Alt+F4, taskbar context "Close") and route the action through the persisted close-behavior preference.

#### Scenario: Close with `minimize-to-tray` preference hides the window
- **WHEN** the user clicks the OS close button on the main window and the persisted preference is `minimize-to-tray`
- **THEN** the main window SHALL be hidden to the tray and the process SHALL remain running

#### Scenario: Close with `exit` preference exits the app
- **WHEN** the user clicks the OS close button on the main window and the persisted preference is `exit`
- **THEN** the application SHALL terminate cleanly, the tray icon SHALL be removed, and no prompt SHALL be shown

#### Scenario: Close with `unset` preference triggers the first-run prompt
- **WHEN** the user clicks the OS close button on the main window and the persisted preference is `unset`
- **THEN** the application SHALL cancel the close, remain running with the window visible, and emit a request to the frontend to display the close-behavior prompt modal

### Requirement: First-run close-behavior prompt collects and persists the user's choice
The system SHALL show a modal dialog the first time the user attempts to close the main window with no recorded close-behavior preference, offering "Exit RedShell" and "Minimize to tray" as the only choices, and SHALL persist the selection so the prompt does not appear again.

#### Scenario: Modal offers exactly two action choices
- **WHEN** the close-behavior prompt modal is displayed
- **THEN** it SHALL present two action buttons labelled "Exit RedShell" and "Minimize to tray", and SHALL NOT be dismissable via Esc, the modal backdrop, or any other UI element

#### Scenario: Choosing exit persists and closes
- **WHEN** the user clicks "Exit RedShell" in the prompt modal
- **THEN** the persisted close-behavior preference SHALL be set to `exit` and the application SHALL terminate cleanly

#### Scenario: Choosing minimize persists and hides
- **WHEN** the user clicks "Minimize to tray" in the prompt modal
- **THEN** the persisted close-behavior preference SHALL be set to `minimize-to-tray` and the main window SHALL be hidden to the tray

#### Scenario: Prompt does not reappear after a choice has been recorded
- **WHEN** the user clicks the OS close button on a subsequent launch and the persisted preference is `exit` or `minimize-to-tray`
- **THEN** no prompt SHALL be shown; the close action SHALL be executed directly per the preference

#### Scenario: Repeated close attempts while prompt is open re-focus the modal
- **WHEN** the prompt modal is already open and the user attempts to close the window again (e.g. Alt+F4)
- **THEN** the application SHALL re-emit the prompt event so the modal regains focus, and SHALL NOT open a second modal or terminate the process

### Requirement: Tray context menu includes a "Check for Updates" item

The system SHALL include a "Check for Updates" item in the right-click context menu of the Windows tray icon, positioned above the "Quit RedShell" item, that opens the Settings -> Updates tab in the main window.

#### Scenario: Menu item is present when auto-update is supported on this install

- **WHEN** the user right-clicks the tray icon on a portable build (writable install directory)
- **THEN** the context menu SHALL contain an item labelled "Check for Updates" between the existing "Close button minimizes to tray" item and the "Quit RedShell" item

#### Scenario: Menu item opens the Settings Updates tab and triggers a check

- **WHEN** the user selects "Check for Updates" from the tray context menu
- **THEN** the main window SHALL be shown and focused, navigation SHALL move to `/settings` with the Updates tab active, and the updater SHALL fire a manual check against the active source

#### Scenario: Menu item is hidden on installed (non-writable) builds

- **WHEN** the user right-clicks the tray icon on an installed build (non-writable install directory, detected at startup)
- **THEN** the "Check for Updates" item SHALL NOT appear in the context menu, since the auto-update flow cannot proceed in that environment

