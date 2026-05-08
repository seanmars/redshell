## Why

RedShell ships portable Windows builds via GitHub and GitLab release pages, but users have no way to discover or apply new versions other than manually checking the release page and re-downloading. Without an in-app update path, users stay on stale versions, miss bug fixes (e.g. the recent error fix in `676db04`), and the project's polish suffers despite frequent releases. Adding a portable auto-updater closes that gap with a low-risk rename-trick replacement that does not require admin rights, code signing, or a separate updater binary.

## What Changes

- Add a new background updater service that polls a user-selected source (GitHub or GitLab) on a configurable interval and surfaces available updates to the frontend.
- Add SHA256 verification of downloaded binaries against a `checksums.txt` sidecar published next to each release asset (non-negotiable since the project does not sign binaries).
- Add a Windows-specific portable replacement flow using the rename trick: download to `redshell.exe.partial` -> verify -> rename current to `redshell.exe.old` -> rename new to `redshell.exe` -> spawn new process -> quit; clean up `.old` on next start.
- Add `AutoUpdate` preference fields (`enabled`, `intervalHours`, `source`, `githubRepo`, `gitlabHost`, `gitlabProject`, `skipVersion`, `lastCheckedAt`) under the existing `~/.redshell/preferences.json` file.
- Add a Settings -> Updates tab letting users enable / disable auto-update, pick the active source (with side-by-side latest-version peek of both GH and GL), choose check interval (1 / 6 / 12 / 24 / 168 hours), trigger a manual check, and view last-check status.
- Add an in-app toast / dialog flow with `Update Now` / `Skip This Version` / `Later` actions when the active source returns a newer version than the running build.
- Add a "Check for Updates" item to the Windows system tray context menu that opens the Updates settings tab.
- Modify `OnBeforeClose` to bypass the close-behavior prompt when an update is in progress, so the app can quit cleanly during the rename swap.
- Modify the release process to emit a `checksums.txt` file alongside `redshell-windows-amd64.exe` (and the existing installer) with sha256 lines in `sha256sum`-compatible format.
- Non-portable installer-based users are explicitly out of scope for v1; if the running binary lives in a non-writable directory (e.g. Program Files), the updater SHALL detect this and surface a "manual update required" message rather than attempt the rename trick.

## Capabilities

### New Capabilities
- `auto-update`: Background version polling, source provider abstraction (GitHub / GitLab parallel sources), portable Windows binary replacement via rename trick, SHA256 integrity verification, and the in-app notify / install user flow.

### Modified Capabilities
- `app-preferences`: New `AutoUpdate` block of preference keys persisted in `~/.redshell/preferences.json` with defined defaults, validation, and observer notifications.
- `system-tray`: New "Check for Updates" item in the right-click context menu that opens the Updates settings tab.
- `wails-app-shell`: `OnBeforeClose` SHALL bypass the close-behavior preference and the prompt event when an update is mid-flight, so the rename swap completes without user interaction.

## Impact

- **Code (new)**: `internal/updater/` (service, providers, rename, cleanup, semver, tests), `app/updater.go`, `frontend/src/components/settings/UpdatesTab.vue`, `frontend/src/composables/useUpdater.ts`.
- **Code (modified)**: `main.go` (wire updater service into `OnStartup` / `OnBeforeClose`, run startup `.old` cleanup), `internal/preferences/service.go` (extend schema with `AutoUpdate` block + defaults + validation), `internal/tray/tray_windows.go` (add menu item), `frontend/src/views/SettingsView.vue` (register tab), `frontend/src/stores/preferences.ts` (expose new preference fields), `app/system.go` (no change, but `GetAppVersion()` is consumed by updater).
- **Dependencies**: Add `golang.org/x/mod/semver` for prerelease-safe version comparison. No frontend dep additions.
- **Release workflow**: New required step to publish `checksums.txt` (or per-asset `.sha256` files in the same format) alongside binaries on every GH / GL release. Documented in `CONTRIBUTING.md` (or equivalent) so manual releases stay consistent.
- **Telemetry / external traffic**: Periodic outbound HTTPS requests to `api.github.com` and the configured GitLab host (default `gitlab.com`); no analytics or back-channel beacons.
- **Security**: Without code signing, SHA256 verification against a publisher-controlled checksum file is the only integrity layer. A signed `checksums.txt.asc` is left as a future enhancement, not part of this change.
- **Out of scope**: macOS / Linux portable updates (the build-tagged rename file SHALL stub them as no-ops); installer-driven updates; differential / binary-patch updates; release-channel concept (beta / nightly).
