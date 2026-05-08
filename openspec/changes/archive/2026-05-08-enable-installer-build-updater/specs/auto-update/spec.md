## ADDED Requirements

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

## MODIFIED Requirements

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
