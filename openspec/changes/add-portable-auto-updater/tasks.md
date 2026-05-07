## 1. Preferences schema extension

- [x] 1.1 Add `AutoUpdate` struct (with `Enabled`, `IntervalHours`, `Source`, `GithubRepo`, `GitlabHost`, `GitlabProject`, `SkipVersion`, `LastCheckedAt` fields and JSON tags) to `internal/preferences/service.go`, defaulting in `defaultPreferences()`
- [x] 1.2 Implement validation helpers for `IntervalHours` (allowed set `{1,6,12,24,168}`) and `Source` (`"github"` or `"gitlab"`); reject invalid values with descriptive errors
- [x] 1.3 Add field-level setters: `SetAutoUpdateEnabled`, `SetAutoUpdateInterval`, `SetAutoUpdateSource`, `SetAutoUpdateGithubRepo`, `SetAutoUpdateGitlabHost`, `SetAutoUpdateGitlabProject`, `SetAutoUpdateSkipVersion`, `SetAutoUpdateLastCheckedAt` and a bulk `SetAutoUpdate` for atomic replacement
- [x] 1.4 Wire observer notifications so changes to `Enabled`, `IntervalHours`, `Source`, `SkipVersion` invoke registered observers; `LastCheckedAt` writes do NOT notify
- [x] 1.5 Extend `internal/preferences/service_test.go` covering: missing block defaulting, partial block defaulting, interval validation, source validation, observer fire/no-fire matrix
- [x] 1.6 Run `go fmt`, `go vet`, `go test ./internal/preferences/...`

## 2. Updater core (provider abstraction + GitHub provider)

- [x] 2.1 Add `golang.org/x/mod/semver` to `go.mod` (`go get golang.org/x/mod/semver`)
- [x] 2.2 Create `internal/updater/types.go` with `Release` struct (Version, PublishedAt, Notes, AssetURL, AssetName, AssetSize, ChecksumsURL) and `Provider` interface (`Name()`, `LatestRelease(ctx)`)
- [x] 2.3 Create `internal/updater/provider_github.go` calling `GET https://api.github.com/repos/{owner}/{repo}/releases/latest`, parsing JSON, locating the `redshell-windows-amd64.exe` and `checksums.txt` asset URLs from the `assets` array
- [x] 2.4 Add `If-None-Match` / `ETag` support to GitHub provider so 304 responses do not consume rate-limit budget; cache the last `ETag` per provider instance in memory
- [x] 2.5 Add `internal/updater/testdata/github_latest.json` fixture (real GH response shape with two assets) and write `provider_github_test.go` using `httptest.Server` to verify URL construction, JSON parsing, asset selection, ETag flow, and error cases (404, malformed JSON, missing assets)
- [x] 2.6 Run `go fmt`, `go vet`, `go test ./internal/updater/...`

## 3. Updater core (GitLab provider)

- [x] 3.1 Create `internal/updater/provider_gitlab.go` calling `GET {host}/api/v4/projects/{urlEncoded(project)}/releases/permalink/latest`, parsing JSON, locating the `redshell-windows-amd64.exe` and `checksums.txt` links from the `assets.links` array
- [x] 3.2 Add `internal/updater/testdata/gitlab_latest.json` fixture and write `provider_gitlab_test.go` matching the GitHub test surface (URL construction with URL encoding, JSON parsing, link selection, error cases)
- [x] 3.3 Run `go fmt`, `go vet`, `go test ./internal/updater/...`

## 4. Updater core (semver, checksum, download)

- [x] 4.1 Create `internal/updater/version.go` with `Compare(a, b string) int` wrapping `semver.Compare` (handle leading-`v` normalization both ways)
- [x] 4.2 Create `internal/updater/checksum.go` with `ParseChecksums(io.Reader) (map[string]string, error)` parsing `<hex>  <filename>` lines into a name->hex map; reject empty / malformed input
- [x] 4.3 Add `Download(ctx, url, destPath, expectedSize int64) (sha256 string, err error)` that streams to a temp `.partial` next to `destPath`, computes SHA-256 during the copy, and returns the lowercase hex string; emits progress callbacks at most every 250 ms
- [x] 4.4 Add unit tests for `Compare` (table-driven over normal versions and prereleases), `ParseChecksums` (valid / extra-whitespace / malformed lines / empty file), and `Download` (httptest serving a known body, verify hash + atomic rename + progress callback)
- [x] 4.5 Run `go fmt`, `go vet`, `go test ./internal/updater/...`

## 5. Updater core (Windows rename trick + cleanup, OS stubs)

- [x] 5.1 Create `internal/updater/rename_windows.go` (`//go:build windows`) implementing `Swap(currentPath, newPath string) error`: rename current to `.old`, rename new to current
- [x] 5.2 Create `internal/updater/rename_other.go` (`//go:build !windows`) returning `ErrPlatformUnsupported` from `Swap`
- [x] 5.3 Create `internal/updater/cleanup.go` with `CleanupStale(exePath string) error` deleting any `<exePath>.old` and `<exePath>.partial` siblings; ignore `os.ErrNotExist`
- [x] 5.4 Create `internal/updater/install_dir.go` with `IsWritable(dir string) bool` that probes by writing and deleting a `redshell-update-probe-*` temp file
- [x] 5.5 Add `rename_windows_test.go` (build-tagged) using `os.CreateTemp` + temp-dir to create three files and assert the swap result; `cleanup_test.go` covering both stale artifacts and missing-file cases; `install_dir_test.go` for writable / non-writable simulations using a read-only temp dir on Windows
- [x] 5.6 Run `go fmt`, `go vet`, `go test ./internal/updater/...`

## 6. Updater service (orchestration)

- [x] 6.1 Create `internal/updater/service.go` defining `Service` struct holding `prefs *preferences.Service`, `runningVersion string`, `exePath string`, `httpClient *http.Client`, `providers map[string]Provider`, `inProgress atomic.Bool`, `lastResult *Release`, `eventEmitter func(name string, data any)`
- [x] 6.2 Implement `NewService(...)` and a test-friendly `NewServiceWithDeps(prefs, exePath, httpClient, providers, emitter)` constructor; default constructor builds providers from prefs and uses `os.Executable()` for `exePath`
- [x] 6.3 Implement `Start(ctx context.Context)`: run `CleanupStale`, check `IsWritable(dir(exePath))`, if not writable emit `updater:manual-required` and return without registering ticker; otherwise schedule the first check based on `lastCheckedAt` and start the ticker goroutine
- [x] 6.4 Implement `runCheck(trigger string)`: emit `updater:check-started`, call `activeProvider().LatestRelease(ctx)`, write `lastCheckedAt`, compare with `Compare(latest, runningVersion)`, emit `updater:available` or `updater:up-to-date` (honoring `skipVersion`), or `updater:error` on failure
- [x] 6.5 Implement `CheckNow()` (manual trigger), `PeekBothSources(ctx)` (returns map[source]Release without touching active source state), `InstallAvailable(release Release)`, `SkipVersion(version string)`, `Unskip()`, `GetState()` (current snapshot for UI)
- [x] 6.6 Implement install pipeline in `InstallAvailable`: download asset to `.partial` in parallel with downloading checksums into memory; verify; rename `.partial` -> `.new`; call `Swap`; spawn child via `exec.Command(exePath).Start()` with `Stdin/Stdout/Stderr = nil`; set `inProgress.Store(true)`; emit `updater:installed`; call `runtime.Quit(ctx)`
- [x] 6.7 Wire preference observer so `Enabled`, `IntervalHours`, `Source`, `SkipVersion` changes restart the ticker accordingly
- [x] 6.8 Add `service_test.go` exercising: skip-startup-check when `lastCheckedAt` recent, fire-startup-check when stale, manual check honors no debounce, source switch triggers immediate check, skip-version suppresses event, install flow happy path against `httptest.Server`-served binary + checksums, install rejects on SHA mismatch, install rejects on missing checksums file, non-writable directory disables ticker
- [x] 6.9 Run `go fmt`, `go vet`, `go test ./internal/updater/...`

## 7. Wails app wrapper + binding

- [x] 7.1 Create `app/updater.go` defining `UpdaterApp` with `Startup(ctx)` capturing the Wails ctx and forwarding it to `Service.Start`
- [x] 7.2 Expose bound methods: `CheckNow() error`, `PeekBothSources() (PeekResult, error)`, `InstallAvailable() error` (uses last cached available release), `SkipVersion(version string) error`, `Unskip() error`, `GetState() State`
- [x] 7.3 Wire `eventEmitter` to `runtime.EventsEmit(ctx, name, data)` so all `updater:*` events reach the frontend
- [x] 7.4 Update `main.go`: instantiate `updater.NewService(...)`, instantiate `app.NewUpdaterApp(...)`, append to `Bind`, call `updaterApp.Startup(ctx)` from `OnStartup`, modify `OnBeforeClose` to short-circuit when `updaterSvc.InProgress()` returns true
- [x] 7.5 Run `go fmt`, `go vet`, `go test ./...`

## 8. System tray menu integration

- [x] 8.1 Extend `internal/tray/tray_windows.go` to accept an updater handle (or a callback) so the menu can call into it
- [x] 8.2 Add a "Check for Updates" menu item between "Close button minimizes to tray" and "Quit RedShell"; on click: show + focus main window, navigate to `/settings?tab=updates`, fire `CheckNow`
- [x] 8.3 If the updater reported `manual-required` at startup, omit the "Check for Updates" item entirely
- [x] 8.4 Update `internal/tray/tray_other.go` stub signature to match
- [ ] 8.5 Manual smoke test on Windows: tray right-click shows the new item; clicking it opens Settings with the Updates tab active

## 9. Frontend Updates tab + composable

- [x] 9.1 Create `frontend/src/composables/useUpdater.ts` exposing reactive state (`status`, `runningVersion`, `latestVersion`, `lastCheckedAt`, `progress`, `error`, `peekGithub`, `peekGitlab`) and methods (`checkNow`, `installAvailable`, `skip`, `unskip`, `peekBoth`); subscribe to `updater:check-started`, `updater:available`, `updater:up-to-date`, `updater:download-progress`, `updater:installed`, `updater:error`, `updater:manual-required` via `EventsOn`
- [x] 9.2 Create `frontend/src/components/settings/UpdatesTab.vue` with: Enable/disable toggle (`AppCheckbox`), interval `<select>` (1/6/12/24/168 hours preset), source radio group (GitHub / GitLab) showing each source's latest version peek with "Use this" buttons, manual "Check now" button, last-checked timestamp, current/latest version display, "Skip this version" / "Unskip" affordance
- [x] 9.3 Update `frontend/src/views/SettingsView.vue` to register the new `UpdatesTab` in the existing `AppTabs`
- [x] 9.4 Wire route query param `?tab=updates` so the tray menu deep-link lands on the right tab
- [x] 9.5 Add a top-level toast or banner in the default layout (consume `useUpdater()`) that surfaces `updater:available` with three buttons: Update Now, Skip, Later; reuse `useToast` if appropriate or use `AppModal` for the install-progress dialog
- [x] 9.6 Add a vitest unit test for `useUpdater.ts` mocking `wailsjs/runtime/runtime` and the bound `UpdaterApp` methods, covering: state transitions on each event, skip persists via store action, install triggers backend method
- [x] 9.7 Run `pnpm format`, `pnpm lint`, `pnpm type-check`, `pnpm test:unit`

## 10. Frontend mechanical leak check + state store

- [x] 10.1 Extend `frontend/src/stores/preferences.ts` (or add a new `stores/updater.ts` if absent) to expose the `AutoUpdate` block bindings used by `UpdatesTab.vue`
- [x] 10.2 Run the daisyUI leak grep from CLAUDE.md against `frontend/src/components/settings` to confirm `UpdatesTab.vue` introduces zero new daisyUI leak; refactor any inline `btn` / `card` etc. into existing `App*` primitives
- [x] 10.3 Run `pnpm lint --fix`, `pnpm format`

## 11. Release workflow documentation

- [x] 11.1 Add a "Cutting a release" section to `README.md` (or `CONTRIBUTING.md` if it exists) describing: tag with `vX.Y.Z`, run `wails build --target windows/amd64 --nsis`, rename portable to `redshell-windows-amd64.exe`, generate `checksums.txt` with `sha256sum redshell-windows-amd64.exe RedShell-amd64-installer.exe > checksums.txt`, upload the three files to BOTH GitHub and GitLab releases under the same tag
- [ ] 11.2 Confirm the documented steps work end-to-end by performing a `0.5.0-rc1` test release (or equivalent) and triggering an in-app check from a build that reports `0.4.0` to verify the full flow against real GH and GL endpoints

## 12. Final verification

- [x] 12.1 Run `go fmt ./...`, `go vet ./...`, `go test ./...` from the repo root; all green
- [x] 12.2 Run `pnpm install`, `pnpm format`, `pnpm lint`, `pnpm type-check`, `pnpm test:unit` from `frontend/`; all green
- [ ] 12.3 Run `wails build --target windows/amd64`; confirm `build/bin/redshell.exe` runs, the Updates tab renders, the tray menu has the new item, and the close prompt still works for the existing `closeBehavior` flow when no update is in flight
- [x] 12.4 Update `wails.json` `productVersion` to the next release number (e.g. `0.5.0`) so the test release in 11.2 can demonstrate a real version bump
- [x] 12.5 Run `openspec validate add-portable-auto-updater --strict` to confirm the change is archive-ready
