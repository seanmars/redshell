## ADDED Requirements

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
