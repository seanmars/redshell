## ADDED Requirements

### Requirement: Auto-update preferences are persisted under the `autoUpdate` key

The system SHALL persist auto-update settings under the `autoUpdate` key of `~/.redshell/preferences.json` with the following schema and defaults:

| Field | Type | Default | Validation |
|---|---|---|---|
| `enabled` | `bool` | `true` | none |
| `intervalHours` | `int` | `6` | MUST be one of `1, 6, 12, 24, 168` |
| `source` | `string` | `"github"` | MUST be `"github"` or `"gitlab"` |
| `githubRepo` | `string` | `"seanmars/redshell"` | MUST match `<owner>/<repo>` |
| `gitlabHost` | `string` | `"https://gitlab.com"` | MUST be a valid `https://` URL |
| `gitlabProject` | `string` | `"seanmars/redshell"` | MUST match `<group>/<project>` (path slashes allowed) |
| `skipVersion` | `string` | `""` | empty OR a valid semver string |
| `lastCheckedAt` | `string` (ISO 8601 / RFC 3339) | `""` | empty OR a parseable timestamp |

#### Scenario: Default auto-update block when key is missing

- **WHEN** the system reads preferences and the `autoUpdate` key is absent from `~/.redshell/preferences.json`
- **THEN** the in-memory representation SHALL contain the full default block as listed above, and a subsequent write SHALL persist the populated block to disk

#### Scenario: Partial auto-update block fills missing fields with defaults

- **WHEN** the system reads preferences and the `autoUpdate` block is present but missing one or more fields (e.g. only `enabled` is set)
- **THEN** missing fields SHALL be populated with their defaults in memory, and a subsequent write SHALL persist the complete block

#### Scenario: Invalid `intervalHours` is rejected on write

- **WHEN** the system attempts to set `autoUpdate.intervalHours` to a value not in `{1, 6, 12, 24, 168}`
- **THEN** the call SHALL return an error and the file SHALL NOT be modified

#### Scenario: Invalid `source` is rejected on write

- **WHEN** the system attempts to set `autoUpdate.source` to a value other than `"github"` or `"gitlab"`
- **THEN** the call SHALL return an error and the file SHALL NOT be modified

#### Scenario: Empty `skipVersion` is the only way to clear a skipped version

- **WHEN** the system sets `autoUpdate.skipVersion` to `""`
- **THEN** the value SHALL be persisted as empty string and subsequent skip-version checks against the empty value SHALL never match a real release

### Requirement: Auto-update preferences are exposed to the frontend via Wails bindings

The system SHALL expose read and write operations for the `autoUpdate` block over Wails bindings so the Settings -> Updates tab can render and modify the preference.

#### Scenario: Frontend reads the auto-update block

- **WHEN** the frontend calls the bound `GetAutoUpdate` method
- **THEN** the method SHALL return the full persisted `AutoUpdate` struct (with defaults applied for missing fields)

#### Scenario: Frontend writes individual auto-update fields

- **WHEN** the frontend calls a bound setter such as `SetAutoUpdateEnabled(bool)`, `SetAutoUpdateInterval(hours int)`, `SetAutoUpdateSource(source string)`, `SetAutoUpdateSkipVersion(version string)`
- **THEN** the method SHALL validate the value (per the schema table), persist the new value, and notify in-process observers of the change

#### Scenario: Frontend updates `lastCheckedAt`

- **WHEN** the updater service finishes a check and calls `SetAutoUpdateLastCheckedAt(time.Time)`
- **THEN** the value SHALL be persisted in RFC 3339 form and the change SHALL NOT trigger an observer callback (to avoid noisy churn from routine ticks)

### Requirement: Preferences observer notifications include auto-update field changes

The system SHALL invoke registered observers when an auto-update field that affects runtime behavior (`enabled`, `intervalHours`, `source`, `skipVersion`) changes from one value to a different value.

#### Scenario: Observer fires on enable/disable toggle

- **WHEN** an observer is registered and `SetAutoUpdateEnabled` is called with a value different from the current one
- **THEN** the observer SHALL be invoked with the new boolean value before the setter returns

#### Scenario: Observer fires on interval change

- **WHEN** an observer is registered and `SetAutoUpdateInterval` is called with a value different from the current one
- **THEN** the observer SHALL be invoked with the new interval before the setter returns

#### Scenario: Observer fires on source change

- **WHEN** an observer is registered and `SetAutoUpdateSource` is called with a value different from the current one
- **THEN** the observer SHALL be invoked with the new source before the setter returns

#### Scenario: Observer does not fire on `lastCheckedAt` updates

- **WHEN** the updater service calls `SetAutoUpdateLastCheckedAt`
- **THEN** observers SHALL NOT be invoked, since this field is purely informational and does not affect runtime behavior
