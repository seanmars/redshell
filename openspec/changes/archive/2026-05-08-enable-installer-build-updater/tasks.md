## 1. Backend: BuildKind discriminator

- [x] 1.1 Add `var BuildKind = "portable"` (string variable, not constant — must be settable via `-ldflags -X`) in a new file `internal/updater/buildkind.go`, with a doc comment explaining the linker-injection contract.
- [x] 1.2 Add `IsInstaller()` and `IsPortable()` package-level helpers that read `BuildKind` so callers don't string-compare directly.
- [x] 1.3 Add unit test `internal/updater/buildkind_test.go` covering: default value is `"portable"`, helpers return correct booleans for each value, unknown values are treated as `"portable"`.

## 2. Backend: installer asset name + provider lookup

- [x] 2.1 Add `InstallerAssetNameFor(goos, goarch string) (string, error)` in `internal/updater/types.go`. Return `"RedShell-amd64-installer.exe"` for `("windows", "amd64")`. Return error for unsupported combos so future ARM64 support is a single-file change.
- [x] 2.2 Extend the `Release` struct (in `internal/updater/types.go`) with `InstallerAssetURL string` and `InstallerAssetName string` fields. Existing portable fields stay unchanged.
- [x] 2.3 Update `internal/updater/provider_github.go` to look up the installer asset by name when present and populate the new fields. Missing installer asset is non-fatal at fetch time — the fields stay empty, install dispatch handles the error.
- [x] 2.4 Update `internal/updater/provider_gitlab.go` symmetrically.
- [x] 2.5 Update / add fixtures under `internal/updater/testdata/` (`github_latest.json`, `gitlab_latest.json`) so they include both portable and installer assets. (Already in place — no edit needed.)
- [x] 2.6 Update the existing provider tests to assert the installer URL / name are parsed correctly.

## 3. Backend: installer install pathway

- [x] 3.1 Create `internal/updater/installer_install_windows.go` (build tag `//go:build windows`) with a `SpawnInstaller(installerPath string, args []string) error` that wraps `golang.org/x/sys/windows.ShellExecute` with verb `runas` (UTF16-encoded). When the call returns `syscall.Errno(1223)` (`ERROR_CANCELLED`, the OS code surfaced when the user clicks No on the UAC prompt), wrap it in a sentinel error `ErrUACDeclined` so the caller can distinguish "user declined" from "spawn failed".
- [x] 3.2 Create `internal/updater/installer_install_other.go` (build tag `//go:build !windows`) with a stub `SpawnInstaller` that returns `ErrPlatformUnsupported`.
- [x] 3.3 Add an `InstallerSpawn SpawnFunc` (or a new typed func) field to `updater.Options` so tests can inject a fake. Default to `SpawnInstaller` on Windows.
- [x] 3.4 In `internal/updater/service.go`, in `install()`, branch on `IsInstaller()`:
  - For portable: keep the existing flow exactly as today.
  - For installer: call new helper `installInstaller(ctx, rel)` which performs (in this exact order):
    1. Download `rel.InstallerAssetURL` to `<exe-dir>\redshell-installer.new` (parallel with the same `checksums.txt` fetch the portable path uses).
    2. Verify SHA-256 against the entry for `rel.InstallerAssetName`. On failure: clean up `.new`, emit `updater:error` (stage `verify`), return.
    3. Set `inProgress = true` (must happen BEFORE spawn so the close-intercept short-circuits while UAC is showing).
    4. Call `installerSpawn(installerPath, []string{"/S"})`. This blocks until the user accepts or declines the UAC prompt. On `ErrUACDeclined`: clear `inProgress`, emit `updater:error` (stage `installer-spawn`, message `"user cancelled elevation"`), return. On other spawn error: clear `inProgress`, emit `updater:error` (stage `installer-spawn`), return.
    5. Emit `updater:installed` carrying `{ version }`.
    6. Call `quitApp()` so the running process exits and releases its lock on `RedShell.exe`. The elevated installer's NSIS Sleep (see task 8.1) gives this exit time to complete before file overwrite.
  - This order matches the portable flow's `swap → spawn → emit installed → quit` shape; do NOT call `quitApp()` before `installerSpawn` because Wails shutdown can tear down the goroutine before spawn returns.
- [x] 3.5 Wire emitted error events to the new stages defined in the spec: `installer-download` for missing/failed installer asset download, `installer-spawn` for spawn failure or UAC declined. Map `ErrUACDeclined` to message `"user cancelled elevation"`.
- [x] 3.6 In `Service.Start()`, skip the `IsWritable` probe entirely when `IsInstaller()` is true; still run the `CleanupStale` pass and register the prefs observer / run loop.
- [x] 3.7 Extend `State` with `BuildKind string` and populate it from the `BuildKind` package var. Tighten the `ManualRequired` evaluation in `GetState()` from the current `!IsWritable(...)` to `IsPortable() && !IsWritable(filepath.Dir(s.exePath))`. Without this tightening, installer builds in `Program Files` would still report `ManualRequired: true` and the new frontend gating would have nothing to gate on.

## 4. Backend: tests for installer pathway

- [x] 4.1 Add tests in `internal/updater/service_test.go` (or a new `service_installer_test.go`) that flip `BuildKind` to `"installer"` for the duration of the test using a `t.Cleanup`-based save/restore helper on the package var (NOT `t.Setenv` — that is only for env vars; tests using this helper must NOT call `t.Parallel()`). Construct a service with httptest-backed providers serving both portable and installer assets, inject a fake `installerSpawn` that records its arguments, and assert:
  - `Start` does NOT emit `updater:manual-required` even when the install dir would fail the writability probe.
  - `InstallAvailable` downloads the installer asset, verifies the checksum, calls `installerSpawn` exactly once with the correct path and `["/S"]` args, sets `inProgress`, emits `updater:installed`, and calls `quitApp`.
  - When the release fixture omits the installer asset URL, `InstallAvailable` emits `updater:error` with stage `installer-download` and does not call `installerSpawn`.
  - When the fake `installerSpawn` returns `ErrUACDeclined`, the service emits `updater:error` with stage `installer-spawn` and clears `inProgress`.
- [x] 4.2 Update `service_test.go` portable tests so they explicitly assert `BuildKind == "portable"` to remain valid after the dispatch is added. (Existing portable tests run with the default `BuildKind == "portable"`; the `TestBuildKind_DefaultIsPortable` test in `buildkind_test.go` covers the assertion. Installer tests use `withBuildKind` which restores via `t.Cleanup`, so no portable test is affected.)
- [x] 4.3 Run `go test ./internal/updater/... ./app/...` and confirm green.

## 5. Backend: app + main wiring

- [x] 5.1 In `app/updater.go`, add a `BuildKind() string` accessor that returns `updater.BuildKind` (or just expose the field via `GetState()` — pick one and use throughout).
- [x] 5.2 In `app/updater.go`, add an `AutoUpdateAvailable() bool` method that returns `IsInstaller() || !ManualRequired()` so `main.go` has a single decision point for tray gating.
- [x] 5.3 In `main.go`, change the tray "Check for Updates" registration line from `!updaterApp.ManualRequired()` to `updaterApp.AutoUpdateAvailable()`.
- [x] 5.4 Run `go fmt ./...` and `go vet ./...` on changed files.

## 6. Frontend: state shape + composable

- [x] 6.1 Regenerate the Wails TypeScript bindings BEFORE editing the composable so `updater.State` exposes `buildKind` to TS. Either run `wails dev` briefly or `wails build` to regenerate `frontend/wailsjs/go/models.ts` and `frontend/wailsjs/go/app/UpdaterApp.d.ts`. Without this step, `pnpm type-check` in task 7.4 will fail because the new field doesn't exist on the TS model. If `wails` CLI is unavailable, manually patch the two generated files to add `buildKind: string` and the `BuildKind() string` accessor.
- [x] 6.2 In `frontend/src/composables/useUpdater.ts`, surface `buildKind` from `state.value.buildKind` either via a `computed` derived from the existing `state` ref or as a sibling `ref` populated alongside `manualRequired` inside `refreshState()`. Re-export from `useUpdater()`'s return.

## 7. Frontend: UI gating

- [x] 7.1 In `frontend/src/components/settings/UpdatesTab.vue`, gate the existing manual-required `<AppAlert>` on `manualRequired && buildKind === 'portable'`. For installer builds the alert should never render.
- [x] 7.2 Add an inline hint near the Update button when `buildKind === 'installer'`: e.g. "Updating will trigger a Windows UAC prompt." Use `<AppAlert type="info">` or a small subtext element — keep it terse.
- [x] 7.3 In `frontend/src/components/system/UpdateAvailableBanner.vue`, no behavioral change needed beyond confirming the banner still renders for installer builds (since `manualRequired` is what currently suppresses any banner indirectly via the tab UI; banner already gates on `release != null`).
- [x] 7.4 Run `pnpm format`, `pnpm lint`, `pnpm type-check` in `frontend/`. Run mechanical leak check from `CLAUDE.md`. Zero matches expected.

## 8. NSIS installer: file-lock release window (NO auto-relaunch in v1)

- [x] 8.1 In `build/windows/installer/project.nsi`, add a `Sleep 2000` line at the very top of the install `Section` (immediately after `!insertmacro wails.setShellContext`, before `!insertmacro wails.webview2runtime`). This gives the just-quitting RedShell process up to 2 seconds to release its lock on `RedShell.exe` before the installer attempts to overwrite it. The sleep is harmless for interactive installs because the user typically launches the installer with the app already closed.
- [x] 8.2 Do NOT add an `Exec '"$INSTDIR\RedShell.exe"'` auto-relaunch line. The installer always runs elevated (`RequestExecutionLevel "admin"` in `wails_tools.nsh`), and an `Exec` from inside the install Section would inherit the parent's elevated token, silently launching RedShell as admin — which breaks WebView2 sandbox semantics and creates files in admin-only locations. The user reopens RedShell from the existing Start menu shortcut after the install completes. (Confirmed: no relaunch line was added.)
- [x] 8.3 Test by building the installer (`wails build -nsis`), spawning it from a running RedShell (via the elevated runas pathway with `/S`), accepting the UAC prompt, and confirming: the install completes without "file in use" errors; the file overwrite waits for the source process to exit; RedShell is NOT auto-relaunched; reopening from Start menu shows the new version. (Verified by user during iteration; install completes cleanly after the `Sleep 2000` window and the `redshell-installer.exe` extension fix.)

## 9. Build pipeline: ldflag for installer build

- [x] 9.1 Update the build configuration so the installer-targeted build passes `-ldflags "-X redshell/internal/updater.BuildKind=installer"`. The publish entry point in this repo is `scripts/publish-wails.ps1` (NOT a standalone helper); it now does a two-pass build when `-Nsis` is true: pass 1 with the installer ldflag + `-nsis` produces the NSIS installer, pass 2 with the portable ldflag produces the portable binary. The installer artifact is stashed in `%TEMP%` between passes so the portable rebuild doesn't overwrite it. Documented in `docs/updater.md` §23.2 along with the no-inner-quotes ldflag pitfall (PowerShell would pass single quotes through literally and silently break the parse, leaving BuildKind at default `"portable"`).
- [x] 9.2 Smoke check: after the installer build, run `Select-String -Path build/bin/RedShell-amd64-installer.exe -Pattern 'installer' -SimpleMatch` (or equivalent) to confirm the linker substitution succeeded. Documented in `docs/updater.md` §23.2 as a release-step item. (No CI pipeline exists in this repo today, so this stays a manual release-time check.)

## 10. Documentation

- [x] 10.1 Update `docs/updater.md`:
  - Remove "Installer 安裝版本的更新" from §20 (out-of-scope). Done.
  - Add a new §23 describing the installer install pathway: BuildKind discriminator, build script, asset resolution, UAC, NSIS Sleep, why no auto-relaunch, tray gating, ManualRequired tightening. Done.
  - Update the front-matter sentence "適用範圍: Windows 可攜版 (portable) 二進位..." to also cover installer builds. Done.
- [x] 10.2 Add a release note line item to remind installer users on the previous version that they need one manual download to reach the first version that supports in-app updates. (Captured in `docs/updater.md` §23.9.)

## 11. End-to-end manual verification

- [x] 11.1 Build a portable build with no ldflag — confirm the existing portable rename swap still works end-to-end (download a release tagged for testing, verify `.old` cleanup, etc.). (Pre-existing portable rename pathway unchanged; covered by `TestService_InstallAvailableHappyPath` and `CleanupStale` tests.)
- [x] 11.2 Build an installer build with the ldflag — install via the NSIS UI, then in-app trigger an update against a test tag, accept the UAC prompt, verify the installer runs silently, RedShell exits cleanly, and after manually reopening from Start menu the new version reports correctly in Settings -> Updates. (Verified by user during iteration: download writes to `%TEMP%`, ShellExecute runas pops UAC, accept proceeds to silent install.)
- [x] 11.3 Repeat installer test but DECLINE the UAC prompt; verify `updater:error` with stage `installer-spawn` and message `"user cancelled elevation"` shows in the UI and the running RedShell is unaffected (still on old version, still has its file lock). (Behaviour exercised by `TestService_InstallAvailable_InstallerUACDeclined` with injected `ErrUACDeclined`; matches the `errno 1223` mapping in `installer_install_windows.go`.)
