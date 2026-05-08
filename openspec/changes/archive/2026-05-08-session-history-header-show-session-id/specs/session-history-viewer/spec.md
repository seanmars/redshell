## ADDED Requirements

### Requirement: Session-info bar exposes a Resume control
The system SHALL render a Resume control in the session-info bar that, when activated, opens a new terminal window running the agent's resume command for the selected session. The resume command pre-types so the user can interact with the running agent in a separate process from the RedShell app.

#### Scenario: Resume control appears next to the copy control
- **WHEN** a session is selected
- **THEN** the session-info bar SHALL render a Resume control immediately to the right of the Copy control, using the same icon-button shape (round, ghost variant, matching size)

#### Scenario: Resume command is constructed per agent
- **WHEN** the user activates the Resume control
- **THEN** the system SHALL launch the resume command for the active agent: `claude --resume <session-id>` for Claude and `copilot --resume <session-id>` for Copilot
- **AND** the `<session-id>` SHALL be the basename of the backend `SessionMeta.sessionID` (the same string that the Copy control writes to the clipboard)

#### Scenario: Spawned terminal starts in the session's project cwd
- **WHEN** the system launches the resume command and the selected session's `SessionMeta.cwd` is non-empty, absolute, and resolves to an existing directory
- **THEN** the spawned terminal process SHALL be created with that directory as its working directory (passed to `CreateProcessW` via `lpCurrentDirectory` / Go's `cmd.Dir`), so the agent CLI starts inside the project rather than inside the session-file directory or RedShell's own cwd
- **AND** the cwd SHALL NOT be interpolated into the shell command line, so paths containing spaces, apostrophes, or non-ASCII characters cannot affect command parsing

#### Scenario: Empty cwd inherits the spawning process's cwd
- **WHEN** the Resume request carries an empty `cwd` (for example a Copilot session whose `workspace.yaml` recorded none)
- **THEN** the system SHALL launch the terminal without setting a working-directory hint and SHALL NOT return an error; the spawned pwsh inherits RedShell's default cwd

#### Scenario: Non-existent project cwd aborts the launch with an error
- **WHEN** the Resume request carries a non-empty `cwd` that fails any of the validation checks: not absolute, does not resolve to an existing path, or resolves to a non-directory entry
- **THEN** the system SHALL return a typed `ErrProjectCwdMissing` error wrapped with the offending path string, and SHALL NOT spawn a terminal
- **AND** the frontend SHALL surface the error to the user as a transient toast that includes the path so the user knows which project directory is missing before retrying

#### Scenario: Default terminal is pwsh on Windows
- **WHEN** the system launches the resume command on Windows
- **THEN** the terminal host SHALL be `pwsh` invoked through a detaching launcher (`cmd /c start "" pwsh -NoExit -NoProfile -Command "<inner>"`) so the spawned window has its own console and is fully independent of RedShell's process tree
- **AND** the inner command SHALL be `<cli> --resume <session-id>`

#### Scenario: Resumed terminal stays open until the user closes it
- **WHEN** the agent CLI inside the resumed terminal completes (exits cleanly, errors out, or is interrupted by the user with Ctrl+C)
- **THEN** the spawned pwsh window SHALL remain open with an interactive prompt rather than auto-closing
- **AND** the window SHALL only close when the user explicitly types `exit` at the prompt or closes the window via its title-bar control
- **AND** the implementation SHALL achieve this via pwsh's `-NoExit` flag combined with `-NoProfile` so that a user pwsh profile setting (for example `$ErrorActionPreference = 'Stop'`) cannot subvert the persistent-shell guarantee

#### Scenario: Spawning launcher does not flash a visible console
- **WHEN** the system invokes the detaching launcher
- **THEN** the transient cmd.exe step SHALL NOT show a visible console window (using the `CREATE_NO_WINDOW` process-creation flag and `STARTF_USESHOWWINDOW + SW_HIDE`)
- **AND** the only user-facing window SHALL be the pwsh window opened by `start`

#### Scenario: Session id is strictly validated before interpolation
- **WHEN** the system constructs the resume command line
- **THEN** the basename session id SHALL be matched against the regular expression `^[A-Za-z0-9_-]+$` before being interpolated into the command line
- **AND** any session id that fails the match SHALL cause the request to be rejected with a typed `ErrInvalidSessionID`, and SHALL NOT result in a terminal launch

#### Scenario: Unknown agent is rejected
- **WHEN** the system receives a Resume request with an `agentID` that is not in the closed enum of supported agents
- **THEN** the request SHALL be rejected with a typed `ErrUnknownAgent`, and SHALL NOT spawn any process

#### Scenario: Unsupported platform returns a typed error
- **WHEN** the system receives a Resume request on a platform that does not implement terminal launching
- **THEN** the call SHALL return a typed `ErrTerminalUnsupported` error and SHALL NOT spawn any process

#### Scenario: Frontend surfaces success and failure
- **WHEN** the Resume control's request resolves successfully
- **THEN** the frontend SHALL display a transient toast indicating that the session is resuming in a new terminal
- **WHEN** the request fails
- **THEN** the frontend SHALL display a transient toast indicating the failure and the underlying error message

## MODIFIED Requirements

### Requirement: Page header reflects the selected session
The system SHALL render a session-info bar at the top of the Session History page's main content area when a session is selected, surfacing the canonical session id (the file's UUID portion) as the primary handle and the rich display name only when it adds information beyond the session id. The page header strip itself ("Session History" title) SHALL remain free of selection-dependent content.

#### Scenario: No session selected
- **WHEN** no session is selected
- **THEN** the page SHALL render only the static "Session History" heading and the per-agent tab control or single-agent two-pane layout, with no session-info bar, no session id text, no copy control, and no display-name line

#### Scenario: Session id is shown as the primary line of the session-info bar
- **WHEN** a session is selected
- **THEN** the page SHALL render a session-info bar at the top of the main content area, above the per-agent tab control (or the two-pane grid when only one agent is enabled)
- **AND** the bar's primary line SHALL render the basename of the backend `SessionMeta.sessionID` field — that is, the substring after the final `/` separator when the id contains one, or the full id when it contains no separator
- **AND** the rendered id SHALL NOT be truncated, ellipsised, or styled to hide characters

#### Scenario: Path-prefixed Claude session ids render only the UUID portion
- **WHEN** the selected session's `sessionID` has the Claude path-prefixed shape `<encoded-cwd>/<uuid>`
- **THEN** the bar SHALL render only the `<uuid>` portion, not the full `<encoded-cwd>/<uuid>` string
- **AND** the encoded directory portion SHALL NOT appear anywhere in the bar

#### Scenario: Copy control copies the displayed session id
- **WHEN** a session is selected
- **THEN** a copy-to-clipboard control SHALL render immediately to the right of the rendered session id in the bar
- **AND** activating the control (click or keyboard activation) SHALL write the same string that is rendered in the bar (the basename of `SessionMeta.sessionID`) to the system clipboard
- **AND** the control SHALL give transient visual feedback that the copy succeeded (icon swap, toast, or both) and SHALL show a non-success state when the clipboard write fails

#### Scenario: Display name renders as a secondary line when it adds information
- **WHEN** a session is selected and the resolved display name is non-empty AND is neither equal to the basename id NOR a strict prefix of the basename id
- **THEN** the bar SHALL render the resolved display name on a second line directly below the session id, styled as smaller, lower-emphasis text than the session id

#### Scenario: Display name is hidden when it would duplicate the session id
- **WHEN** a session is selected and the resolved display name is empty, OR equal to the basename id, OR a strict prefix of the basename id (the documented short-id fallback case)
- **THEN** the bar SHALL NOT render any display-name line, and the only visible session-identifying content in the bar SHALL be the basename session id with its copy control

#### Scenario: Bar height is stable across display-name visibility changes
- **WHEN** the user selects different sessions in succession, some of which resolve a rich display name and some of which do not
- **THEN** the session-info bar SHALL occupy the same outer height in both states so that the content beneath it (the per-agent tab control, the session list, and the event timeline) does not shift vertically when the display-name line appears or disappears
- **AND** the bar's height SHALL be reserved up-front (for example via a fixed height utility class) rather than driven by its current content

#### Scenario: Display name resolution for Claude
- **WHEN** the rich display name for a Claude session is resolved
- **THEN** the resolver SHALL return the first non-empty value from this ordered list: `custom-title` event's `customTitle`, `agent-name` event's `agentName`, the first non-meta `user.message` whose `message.content` is a string and does not begin with `<local-command-`, `<command-`, or `<system-reminder>`, the session's `slug`, the first 8 characters of `sessionId`

#### Scenario: Display name resolution for Copilot
- **WHEN** the rich display name for a Copilot session is resolved
- **THEN** the resolver SHALL return the first non-empty value from this ordered list: `workspace.yaml.summary`, the first `user.message` event's `data.content`, `workspace.yaml.repository`, `workspace.yaml.cwd`, the first 8 characters of the session id
