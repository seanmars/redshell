## ADDED Requirements

### Requirement: Open a filesystem path in the OS default handler
The system SHALL provide a Wails-bound capability that opens a filesystem path (file or directory) using the host operating system's default handler, without blocking the GUI thread.

#### Scenario: Open a directory on Windows
- **WHEN** the frontend calls `OpenPath` with a directory path on Windows
- **THEN** the system SHALL launch File Explorer focused on that directory and SHALL return without waiting for the launched process to exit

#### Scenario: Open a file on macOS
- **WHEN** the frontend calls `OpenPath` with a file path on macOS
- **THEN** the system SHALL invoke `open` with that path so the file launches in its default application

#### Scenario: Open a path on Linux
- **WHEN** the frontend calls `OpenPath` with a path on Linux
- **THEN** the system SHALL invoke `xdg-open` with that path

### Requirement: Tilde-prefixed paths SHALL be expanded
The system SHALL accept paths that start with `~` or `~/` and expand them to the user's home directory before invoking the OS handler.

#### Scenario: Expand a tilde directory path
- **WHEN** the frontend calls `OpenPath` with the value `~/.claude`
- **THEN** the system SHALL expand it to `<UserHomeDir>/.claude` before invoking the OS handler

#### Scenario: Expand a tilde file path
- **WHEN** the frontend calls `OpenPath` with the value `~/.copilot/config.json`
- **THEN** the system SHALL expand it to `<UserHomeDir>/.copilot/config.json` before invoking the OS handler

### Requirement: Surface a failure when the path does not exist
The system SHALL return an error from `OpenPath` when the target path does not exist on disk, so the caller can render user-visible feedback.

#### Scenario: Missing target returns error
- **WHEN** the frontend calls `OpenPath` with a path that fails `os.Stat`
- **THEN** the system SHALL return a non-nil error whose message identifies the missing path and SHALL NOT launch any handler
