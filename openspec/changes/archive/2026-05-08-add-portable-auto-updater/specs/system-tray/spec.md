## ADDED Requirements

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
