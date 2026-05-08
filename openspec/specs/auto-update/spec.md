# auto-update Specification

## Purpose

Provide an in-app auto-update capability for portable Windows builds of RedShell that polls a user-selected release source (GitHub or GitLab) on a fixed interval, compares semver versions, downloads the OS/arch-appropriate asset, verifies it against a publisher-controlled SHA256 sidecar, and replaces the running binary using a rename-then-spawn swap. The flow exposes lifecycle events to the Vue frontend, supports skip/defer/install actions, and coordinates with the close intercept via an in-progress flag.
## Requirements
### Requirement: Updater service polls the active release source on a fixed interval

The system SHALL run a background updater service that periodically queries the user-selected release source (GitHub or GitLab) for the latest published release and compares its semver version against the running build's version.

#### Scenario: Initial check on startup when interval has elapsed

- **WHEN** the application starts and `prefs.autoUpdate.enabled` is `true` and `now - prefs.autoUpdate.lastCheckedAt >= intervalHours`
- **THEN** the updater SHALL fire one immediate check against the active source within 5 seconds of `OnStartup` completing, and SHALL update `lastCheckedAt` after the check resolves (success or failure)

#### Scenario: Initial check on startup when interval has not elapsed

- **WHEN** the application starts and `prefs.autoUpdate.enabled` is `true` and `now - prefs.autoUpdate.lastCheckedAt < intervalHours`
- **THEN** the updater SHALL skip the immediate check and schedule the next check at `lastCheckedAt + intervalHours`

#### Scenario: Periodic ticker re-checks at the configured interval

- **WHEN** the application has been running for `intervalHours` since the last check and `prefs.autoUpdate.enabled` is `true`
- **THEN** the updater SHALL fire a check against the active source and update `lastCheckedAt`

#### Scenario: Disabled auto-update suspends polling

- **WHEN** `prefs.autoUpdate.enabled` is `false`
- **THEN** the updater SHALL NOT run any background check, and SHALL NOT consume polling budget against the active source

#### Scenario: Manual check ignores the elapsed-interval debounce

- **WHEN** the user invokes a manual "Check for updates" action from the Settings -> Updates tab or the tray menu
- **THEN** the updater SHALL fire a check immediately regardless of `lastCheckedAt`, and SHALL update `lastCheckedAt` after the check resolves

### Requirement: Updater supports two parallel sources, with the user picking one as active

The system SHALL support GitHub and GitLab as two independent release sources whose latest versions may differ. The user SHALL select exactly one source as the active source for background polling.

#### Scenario: GitHub provider returns the latest release

- **WHEN** the active source is `"github"` and the configured `githubRepo` is `"<owner>/<repo>"`
- **THEN** the updater SHALL call the GitHub Releases API at `GET https://api.github.com/repos/<owner>/<repo>/releases/latest` and parse the response into a `Release` value containing version, publish time, notes, and the URL of the OS/arch-appropriate asset and the `checksums.txt` asset

#### Scenario: GitLab provider returns the latest release

- **WHEN** the active source is `"gitlab"` and `gitlabHost` is `"https://gitlab.com"` and `gitlabProject` is `"<group>/<project>"`
- **THEN** the updater SHALL call the GitLab Releases API at `GET <gitlabHost>/api/v4/projects/<URL-encoded(gitlabProject)>/releases/permalink/latest` and parse the response into a `Release` value with the same fields as the GitHub provider

#### Scenario: Switching active source triggers an immediate check

- **WHEN** the user changes `prefs.autoUpdate.source` from one provider to the other
- **THEN** the updater SHALL stop the current ticker, fire one immediate check against the new source, update `lastCheckedAt`, and reschedule the ticker for the configured interval

#### Scenario: Settings page peeks both sources without changing active source

- **WHEN** the user opens the Settings -> Updates tab
- **THEN** the frontend SHALL call a backend method that queries BOTH the GitHub and GitLab providers in parallel and returns the latest version metadata for each, WITHOUT modifying `prefs.autoUpdate.source` or `lastCheckedAt`

#### Scenario: Source query failure surfaces an error without falling back

- **WHEN** a query against the active source fails (network error, non-2xx response, malformed JSON)
- **THEN** the updater SHALL emit a `updater:error` event with the source name and error message, and SHALL NOT silently retry against the other source

### Requirement: Update availability is determined by semver comparison and skip-version

The system SHALL compare the active source's latest release version against the running build's version using semver-aware comparison and SHALL emit an "available" event only when the latest version is strictly greater than the running version AND not equal to the persisted `skipVersion`.

#### Scenario: Newer release emits an availability event

- **WHEN** a check resolves with a release whose version compares greater than `GetAppVersion()` and the version does not equal `prefs.autoUpdate.skipVersion`
- **THEN** the updater SHALL emit a `updater:available` runtime event carrying the full `Release` payload

#### Scenario: Equal-or-older release emits no event

- **WHEN** a check resolves with a release whose version compares less than or equal to `GetAppVersion()`
- **THEN** the updater SHALL NOT emit a `updater:available` event, and the Settings UI SHALL display a "You are on the latest version" indicator

#### Scenario: Skipped version emits no event

- **WHEN** a check resolves with a release whose version equals `prefs.autoUpdate.skipVersion`
- **THEN** the updater SHALL NOT emit a `updater:available` event, but the Settings UI SHALL still display the latest version (with a "Skipped" badge) so the user can revoke the skip

#### Scenario: Prerelease comparison uses semver ordering

- **WHEN** the running version is `v0.5.0` and the latest release is `v0.5.0-rc1`
- **THEN** the updater SHALL determine that no update is available (because `v0.5.0-rc1 < v0.5.0` under semver ordering)

### Requirement: Downloads are verified against a publisher-controlled SHA256 sidecar

The system SHALL download the OS/arch-appropriate binary asset and the `checksums.txt` sidecar from the same release, parse the sidecar in `sha256sum`-compatible format, and verify the downloaded binary's SHA-256 hash against the line whose filename matches the asset.

#### Scenario: Successful verification proceeds with installation

- **WHEN** the streamed SHA-256 of the downloaded asset equals the hash in `checksums.txt` for the asset filename
- **THEN** the updater SHALL atomically rename the verified payload from `redshell.exe.partial` to `redshell.exe.new` and proceed with the swap flow

#### Scenario: Verification mismatch aborts the install

- **WHEN** the streamed SHA-256 of the downloaded asset does not equal the hash in `checksums.txt`
- **THEN** the updater SHALL delete the `.partial` file, emit a `updater:error` event indicating "checksum mismatch", and SHALL NOT swap the running binary

#### Scenario: Missing sidecar aborts the install

- **WHEN** the release's `checksums.txt` cannot be downloaded (404, network error) OR cannot be parsed (no whitespace-separated hash + filename pairs)
- **THEN** the updater SHALL emit a `updater:error` event indicating "checksum file unavailable", and SHALL NOT install without verification

#### Scenario: Sidecar without an entry for the asset aborts the install

- **WHEN** `checksums.txt` is well-formed but contains no line whose second field matches the asset filename
- **THEN** the updater SHALL emit a `updater:error` event indicating "asset not listed in checksums", and SHALL NOT install

### Requirement: Portable Windows replacement uses the rename trick

On Windows, when the running binary is a portable-kind build, the system SHALL replace the running binary using a rename-then-spawn sequence that survives the OS file lock on the running executable, and SHALL clean up the previous-version artifact on the next process start. This requirement does NOT apply to installer-kind builds, which use the elevated silent-installer pathway instead.

#### Scenario: Successful rename swap and respawn

- **WHEN** the user accepts an available update on a portable-kind build and `redshell.exe.new` has been created and verified
- **THEN** the updater SHALL rename the running `redshell.exe` to `redshell.exe.old`, rename `redshell.exe.new` to `redshell.exe`, spawn the new process via `exec.Command(newExePath).Start()` with detached stdio, set the in-progress flag, and call the Wails runtime quit

#### Scenario: Stale `.old` cleanup on next start

- **WHEN** the application starts and a `redshell.exe.old` (or any `*.old` matching the running exe basename) exists in the same directory
- **THEN** the updater SHALL attempt to delete it on startup, and SHALL silently ignore failures (the file is no longer locked once the previous process has exited)

#### Scenario: Stale `.partial` cleanup on next start

- **WHEN** the application starts and a `redshell.exe.partial` exists in the same directory
- **THEN** the updater SHALL delete it on startup so a previously interrupted download does not consume disk indefinitely

#### Scenario: Rename failure preserves the original binary

- **WHEN** any of the three rename operations fails (e.g. AV lock, permission denied) on a portable-kind build
- **THEN** the updater SHALL NOT have replaced the running `redshell.exe`, SHALL surface the OS error to the UI, and SHALL leave the `.new` file in place so the user can manually swap if desired

#### Scenario: Non-writable install directory on a portable build disables auto-update

- **WHEN** `BuildKind == "portable"` AND the directory containing the running exe is not writable by the current user (e.g. a portable binary placed under `Program Files`)
- **THEN** the updater SHALL NOT register the background ticker, and SHALL emit a one-time `updater:manual-required` event so the UI can display a "This is an installed build; download the portable build to enable auto-updates" message

### Requirement: Update flow exposes lifecycle events to the frontend

The system SHALL emit Wails runtime events for each meaningful step of the check-and-install flow so the Vue frontend can render progress, errors, and completion without polling.

#### Scenario: Check started

- **WHEN** the updater begins a check against the active source
- **THEN** it SHALL emit `updater:check-started` carrying `{ source: "github" | "gitlab", trigger: "startup" | "ticker" | "manual" }`

#### Scenario: Available update

- **WHEN** the updater determines a newer non-skipped version is available
- **THEN** it SHALL emit `updater:available` carrying the full `Release` payload

#### Scenario: Up-to-date

- **WHEN** the updater determines no newer version is available
- **THEN** it SHALL emit `updater:up-to-date` carrying `{ source, latestVersion, runningVersion }`

#### Scenario: Download progress

- **WHEN** a download is in progress (portable asset OR installer asset)
- **THEN** the updater SHALL emit `updater:download-progress` carrying `{ bytesDownloaded, totalBytes }` at most every 250ms

#### Scenario: Install complete

- **WHEN** the updater has successfully renamed and spawned the new process (portable) OR has successfully spawned the elevated silent installer (installer) and is about to quit
- **THEN** it SHALL emit `updater:installed` carrying the new version

#### Scenario: Error

- **WHEN** any step fails (network, parse, checksum, rename, spawn, installer download, installer spawn, UAC declined)
- **THEN** the updater SHALL emit `updater:error` carrying `{ stage, message }` where `stage` is one of `"check"`, `"download"`, `"verify"`, `"rename"`, `"spawn"`, `"installer-download"`, `"installer-spawn"`

### Requirement: User can skip a specific version, defer, or install immediately

The system SHALL provide three actions when an update is available: install now, skip this version (persisted), and defer (no persistence; available again on the next check).

#### Scenario: Install now begins the download flow

- **WHEN** the user invokes "Update Now" from the available-update UI
- **THEN** the updater SHALL start downloading the asset and the checksums file in parallel and proceed through verify -> rename -> spawn -> quit

#### Scenario: Skip this version persists the version string

- **WHEN** the user invokes "Skip this version" for a release with version `vX.Y.Z`
- **THEN** the updater SHALL set `prefs.autoUpdate.skipVersion = "vX.Y.Z"` and dismiss the toast/dialog; subsequent checks resolving the same version SHALL NOT re-emit the available event

#### Scenario: Later defers without persistence

- **WHEN** the user invokes "Later" from the available-update UI
- **THEN** the updater SHALL dismiss the toast/dialog without modifying any preference; the next ticker tick SHALL re-emit the available event

#### Scenario: Skipping a version does not hide it from Settings

- **WHEN** `prefs.autoUpdate.skipVersion` equals the latest version returned by the active source
- **THEN** the Settings -> Updates tab SHALL still display that version with a "Skipped" indicator and an "Unskip" action

### Requirement: Updater service exposes an in-progress flag for close-intercept coordination

The system SHALL expose a thread-safe `InProgress() bool` accessor that returns `true` from the moment the rename swap begins until the application process exits.

#### Scenario: Flag is true during the rename-and-spawn window

- **WHEN** the updater has begun the rename of `redshell.exe` to `redshell.exe.old`
- **THEN** `InProgress()` SHALL return `true` for any caller until the process exits

#### Scenario: Flag is false during normal operation

- **WHEN** no install is in flight (no download started, or download finished but install not yet attempted)
- **THEN** `InProgress()` SHALL return `false`

### Requirement: Provider abstraction is testable without network access

The system SHALL allow tests to substitute the GitHub and GitLab providers with fake implementations or `httptest.Server` instances so unit tests do not depend on real release endpoints.

#### Scenario: Service constructor accepts injected provider and base directory

- **WHEN** a test constructs the updater service via a constructor that accepts a `Provider` and an explicit working directory
- **THEN** the service SHALL use those dependencies for all I/O, and SHALL NOT fall back to real GitHub/GitLab endpoints or the user's exe directory

#### Scenario: Provider tests use httptest

- **WHEN** the GitHub or GitLab provider tests run
- **THEN** they SHALL stand up an `httptest.Server` returning fixture JSON responses and assert the parsed `Release` matches the expected shape, without contacting `api.github.com` or `gitlab.com`

### Requirement: Build kind discriminates portable vs installer install pathway

The system SHALL carry a build-time `BuildKind` discriminator with the values `"portable"` (default) or `"installer"`, baked into the binary at build time via linker flag, and SHALL use this discriminator to choose the install pathway taken by `InstallAvailable()`.

#### Scenario: Portable build uses the rename swap pathway

- **WHEN** the running binary was built without the installer linker flag (`BuildKind == "portable"`) and `InstallAvailable()` is invoked
- **THEN** the updater SHALL download the portable asset (e.g. `redshell-windows-amd64.exe`), verify it against `checksums.txt`, and proceed through the rename-then-spawn swap as defined in the Portable Windows replacement requirement

#### Scenario: Installer build uses the silent-installer pathway

- **WHEN** the running binary was built with the installer linker flag (`BuildKind == "installer"`) and `InstallAvailable()` is invoked
- **THEN** the updater SHALL download the installer asset (e.g. `RedShell-amd64-installer.exe`), verify it against `checksums.txt`, and proceed through the elevated silent-install pathway as defined in the Installer Windows replacement requirement

#### Scenario: BuildKind is exposed in the State snapshot

- **WHEN** the frontend calls `GetState()`
- **THEN** the returned `State` SHALL include a `buildKind` field equal to the running binary's `BuildKind` so the UI can render mode-specific copy

#### Scenario: BuildKind is exposed to the tray gating in main.go

- **WHEN** `main.go` decides whether to register the tray "Check for Updates" item
- **THEN** the decision SHALL be true when `BuildKind == "installer"` OR (`BuildKind == "portable"` AND the install directory is writable), and SHALL be false otherwise

### Requirement: Installer Windows replacement re-runs the silent NSIS installer with UAC elevation

When the running binary is an installer-kind build on Windows, the system SHALL replace itself by downloading the installer release asset, verifying it against `checksums.txt`, and spawning it under a `runas` shell verb with the NSIS silent flag so a single UAC prompt elevates the install. The installer SHALL be responsible for relaunching the application after the install completes.

#### Scenario: Successful elevated silent install

- **WHEN** the user accepts an available update on an installer-kind build, the verified installer asset has been written to `<exe-dir>\redshell-installer.new`, and the spawn helper invokes Windows `ShellExecuteW` with verb `runas` and parameters `/S`
- **THEN** the updater SHALL set the in-progress flag to true BEFORE the spawn call so the close intercept short-circuits while the UAC dialog is showing, and AFTER `installerSpawn` returns success SHALL emit `updater:installed` carrying the new version and call `quitApp()` so the running process exits and the elevated installer can overwrite files

#### Scenario: User declines the UAC elevation prompt

- **WHEN** the spawn helper returns the Windows error code corresponding to "operation cancelled by user" (`syscall.Errno(1223)` / `ERROR_CANCELLED`)
- **THEN** the updater SHALL clear the in-progress flag, leave the running binary in place, and emit a `updater:error` event with `{ stage: "installer-spawn", message: "user cancelled elevation" }`

#### Scenario: Installer asset missing from the release

- **WHEN** the active source's release does not contain an asset whose name matches `InstallerAssetNameFor(GOOS, GOARCH)` (e.g. `RedShell-amd64-installer.exe`)
- **THEN** the updater SHALL emit a `updater:error` event with `{ stage: "installer-download", message: "installer asset not found" }` and SHALL NOT attempt to fall back to the portable swap pathway

#### Scenario: Installer checksum mismatch aborts the install

- **WHEN** the streamed SHA-256 of the downloaded installer asset does not equal the hash in `checksums.txt` for the installer filename
- **THEN** the updater SHALL delete the downloaded installer file, emit a `updater:error` event with `{ stage: "verify", message: "checksum mismatch ..." }`, and SHALL NOT spawn the installer

#### Scenario: NSIS installer waits briefly for the running app to release its file lock

- **WHEN** the elevated NSIS installer begins its install Section after being spawned by the in-app updater
- **THEN** the installer SHALL include a brief `Sleep 2000` instruction at the top of the install Section, before any file overwrite, so the just-quitting RedShell process has time to release its OS lock on the running executable

#### Scenario: NSIS installer does NOT auto-relaunch RedShell in silent mode

- **WHEN** the silent installer (`/S`) completes its install section successfully on Windows
- **THEN** the installer SHALL NOT execute the relaunched RedShell from inside the install Section. The reason is that `wails_tools.nsh` declares `RequestExecutionLevel "admin"`, so an `Exec` from inside the install Section would inherit the parent's elevated token and silently launch RedShell with admin rights. The user reopens RedShell from the existing Start menu / desktop shortcut.

#### Scenario: Installer install pathway does not consult the writability probe

- **WHEN** `BuildKind == "installer"` and `Start()` is called
- **THEN** the updater SHALL skip the `IsWritable(filepath.Dir(exePath))` probe, SHALL register the run loop and the preferences observer, and SHALL NOT emit `updater:manual-required`

