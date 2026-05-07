# app-preferences Specification

## Purpose
TBD - created by archiving change windows-tray-icon. Update Purpose after archive.
## Requirements
### Requirement: Application preferences are persisted in `~/.redshell/preferences.json`
The system SHALL persist application-level user preferences in a JSON file located at `~/.redshell/preferences.json`, separate from agent-setup state in `~/.redshell/settings.json`.

#### Scenario: Preferences file is created on first write
- **WHEN** the system writes a preference for the first time and `~/.redshell/preferences.json` does not exist
- **THEN** the parent directory `~/.redshell/` SHALL be created if missing, and the file SHALL be written with mode `0644` containing a valid JSON object

#### Scenario: Missing preferences file yields default values
- **WHEN** the system reads preferences and `~/.redshell/preferences.json` does not exist
- **THEN** the read SHALL succeed and return the default preference values without creating the file

#### Scenario: Malformed preferences file surfaces an error
- **WHEN** the system reads preferences and `~/.redshell/preferences.json` exists but cannot be parsed as JSON
- **THEN** the read SHALL return a descriptive error rather than silently overwriting or returning defaults

### Requirement: Close-behavior preference exposes three valid states
The system SHALL store the close-button behavior preference under the key `closeBehavior` with one of three string values: `"unset"`, `"exit"`, or `"minimize-to-tray"`. The default value when the file is absent or the key is missing SHALL be `"unset"`.

#### Scenario: Default close-behavior is unset
- **WHEN** the system reads `closeBehavior` and the preferences file is absent or the key is missing
- **THEN** the returned value SHALL be `"unset"`

#### Scenario: Setting close-behavior persists immediately
- **WHEN** the system sets `closeBehavior` to `"exit"` or `"minimize-to-tray"`
- **THEN** the new value SHALL be written to `~/.redshell/preferences.json` before the call returns, and a subsequent read SHALL observe the new value

#### Scenario: Invalid close-behavior values are rejected
- **WHEN** the system attempts to set `closeBehavior` to a value other than `"unset"`, `"exit"`, or `"minimize-to-tray"`
- **THEN** the call SHALL return an error and the file SHALL NOT be modified

### Requirement: Preferences are exposed to the frontend via Wails bindings
The system SHALL expose preference read and write operations to the Vue frontend via Wails bindings on a dedicated app wrapper, so that the close-behavior modal and any future settings UI can consult and update preferences.

#### Scenario: Frontend reads close-behavior
- **WHEN** the frontend calls the bound `GetCloseBehavior` method
- **THEN** the method SHALL return the current persisted value (`"unset"`, `"exit"`, or `"minimize-to-tray"`)

#### Scenario: Frontend writes close-behavior
- **WHEN** the frontend calls the bound `SetCloseBehavior` method with a valid value
- **THEN** the method SHALL persist the value, notify in-process observers (such as the tray menu's checked state), and return success

#### Scenario: Frontend requests application exit after prompt
- **WHEN** the frontend calls the bound `RequestExit` method (used after the user picks "Exit RedShell" in the close-behavior prompt)
- **THEN** the application SHALL terminate cleanly without re-triggering the close-behavior prompt

### Requirement: Preferences service notifies in-process observers on change
The system SHALL allow in-process Go components (such as the tray manager) to subscribe to preference changes so that derived UI state (e.g. the tray menu's checkbox) can stay in sync with persisted values without polling.

#### Scenario: Observer is notified when close-behavior changes
- **WHEN** an observer is registered via `OnChange` and `SetCloseBehavior` is called with a value different from the current one
- **THEN** the observer callback SHALL be invoked with the new value before `SetCloseBehavior` returns

#### Scenario: Observer is not notified when value is unchanged
- **WHEN** an observer is registered via `OnChange` and `SetCloseBehavior` is called with the same value as the current one
- **THEN** the observer callback SHALL NOT be invoked

