## Context

`internal/updater` was designed v1 with an explicit non-goal of supporting installer builds (`docs/updater.md` §20). The only knob it has today for "is this an installable build?" is a runtime write probe (`IsWritable(filepath.Dir(s.exePath))` in `install_dir.go`). When the probe fails — which is what happens for any binary running from `C:\Program Files\RedShell\` because NSIS installs require admin elevation — the service refuses to start the run loop and emits `updater:manual-required`. The frontend then renders a banner that tells the user to download the portable build manually.

That design was correct for v1 because there was no decided strategy for how to overwrite a running binary in a directory the process can't write to. NSIS shipping is now mature in this repo (the installer macros in `build/windows/installer/wails_tools.nsh` are wired and a release-time `RedShell-amd64-installer.exe` asset is already promised by the release workflow in `docs/updater.md` §19, alongside a unified `checksums.txt`). With the publishing side ready, we can teach the updater to take a different install pathway when it knows it is an installer build: download the same release's installer asset, verify its checksum exactly like the portable asset, then spawn it with the NSIS silent flag (`/S`) under a `runas` shell verb so Windows UAC elevates it. The installer overwrites files (with admin rights), then re-launches RedShell at the end of its install section.

Constraints / current state to respect:

- The `Service` struct, its `runLoop`, and `Start` flow are already fairly tightly tested. We want to add a branch, not restructure.
- `Swap` (rename trick) is platform-conditional via build tags (`rename_windows.go` vs `rename_other.go`). The new installer path is also Windows-only and must follow the same build-tag pattern.
- The release artifact pipeline already emits the installer binary with a stable name and entry in `checksums.txt`. We do not need to extend the release process — only the consumer.
- The frontend's `manualRequired` flag is wired into multiple places (banner, settings tab, tray gating). Replacing it with a build-kind concept needs a single source of truth in `State`.

## Goals / Non-Goals

**Goals:**

- Installer-installed RedShell on Windows can perform an in-app update without the user manually downloading a new binary.
- Reuse the existing release artifacts (`RedShell-amd64-installer.exe`, `checksums.txt`) and verification logic — no new endpoints, no new asset format.
- Maintain a single, unambiguous signal at runtime for which install pathway to use, set at build time so it is not subject to environment variance.
- Keep the portable rename-trick code path unchanged for portable builds. Zero behavior change for portable users.
- Surface the active install pathway in `State` so the frontend renders the correct UI (no manual-required banner for installer builds; optional UAC-prompt hint).

**Non-Goals:**

- Differential / patch updates of installer files. The installer always does a full install.
- Per-machine vs per-user installer modes. The current installer requests admin (`RequestExecutionLevel "admin"`) and stays that way.
- Auto-rollback if the installer fails. NSIS keeps the previous install intact until success; if the silent install fails, the user still has the previous version.
- Suppressing the UAC prompt (impossible without an admin-installed updater service, which is overkill here).
- macOS / Linux installer flow. v1 of those platforms is still out of scope per `rename_other.go`.

## Decisions

### Decision 1: Build-kind is a linker-injected constant, not a runtime detector

**Choice:** Add `var BuildKind = "portable"` in `internal/updater/types.go` (or a new `buildkind.go`). The installer-targeted `wails build` invocation passes `-ldflags "-X 'redshell/internal/updater.BuildKind=installer'"`. At runtime, every install dispatch reads this constant directly — no probing, no registry lookup, no env var.

**Rationale:** Three alternatives were considered:

1. **Writability probe (current approach extended)**: distinguishes "writable" from "not writable", but conflates two different things — a portable build dropped into a non-writable folder is a user mistake, not the installer pathway. Conflating them caused the original limitation.
2. **NSIS uninstall registry key lookup**: read `HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\seanmarsRedShell` and check whether `UninstallString`'s parent folder matches the running exe's parent. Works, but it's a runtime side-channel that depends on registry state we don't control deterministically.
3. **Linker-injected constant (chosen)**: deterministic, baked into the binary, impossible to mistake. Costs a one-time wiring change in the build command. Symmetrical with the way `runningVersion` is currently sourced from the embedded `wails.json`.

The writability probe stays as a **fallback** for the portable pathway only: if `BuildKind == "portable"` and the dir is not writable, the existing `updater:manual-required` flow still triggers. The probe is no longer consulted for installer builds.

### Decision 2: Installer pathway = download the same release's installer asset, verify with `checksums.txt`, ShellExecute with `runas` + `/S`

**Choice:** When `BuildKind == "installer"` and the user invokes `InstallAvailable()`, the service:

1. Computes the installer asset name via a new `InstallerAssetNameFor(goos, goarch)` (Windows AMD64 → `RedShell-amd64-installer.exe`).
2. Looks up that filename in the release's assets (the same `Release.AssetURL` resolution provider already does, just with a different filename — easiest to extend each provider's `LatestRelease` to also resolve the installer asset URL when present, or to add a sibling lookup method `InstallerAssetURL`).
3. Downloads it to `<exe-dir>\redshell-installer.new`, streaming SHA-256, in parallel with the same `checksums.txt` already required for verification.
4. Looks up the installer filename in the parsed checksums map; aborts with `updater:error` (stage=`verify`) on mismatch / missing entry. (Same code path as portable verify.)
5. Calls a new `internal/updater/installer_install_windows.go` `SpawnInstaller(installerPath, args)` that uses Windows `ShellExecuteW` (via `golang.org/x/sys/windows`) with verb `runas` and the parameter string `/S` to trigger UAC elevation.
6. Sets `inProgress = true` immediately before the spawn (so the close-intercept skips the prompt) and calls `quitApp()` after spawn returns success. The running RedShell exits; the elevated installer overwrites files; the installer's `Section` ends with `Exec "$INSTDIR\${PRODUCT_EXECUTABLE}"` to relaunch.

**Rationale:** Three alternatives were considered:

- **Bundle a tiny elevated helper exe with the app**: would let us reuse the existing rename swap with admin rights. But it requires shipping a second signed binary, the helper itself can't update itself, and we'd duplicate verification logic. NSIS already does all of the "copy files into Program Files with elevation" work.
- **Use `wusa.exe`-style MSI delivery**: switch the build to MSI. Larger refactor, doesn't reuse the NSIS work already in place.
- **Silent NSIS reinstall (chosen)**: zero new binary to ship, leverages existing installer, single UAC prompt, NSIS handles the file lock by waiting / retry. The cost is one UAC prompt per update — acceptable for a developer tool installed by power users.

`ShellExecuteW` rather than `exec.Command`: `exec.Command` cannot raise a UAC elevation prompt; only `ShellExecuteEx` with verb `runas` can. We confine this to a Windows-tagged file so non-Windows builds never compile it.

### Decision 3: Installer does NOT auto-relaunch RedShell in v1; UI tells the user to reopen

**Choice:** The installer-install pathway intentionally does NOT relaunch RedShell after the silent install completes. The user reopens RedShell from the existing Start menu / desktop shortcut. The UI surfaces a one-line note before the install starts: "After the installer finishes, reopen RedShell from your Start menu."

**Rationale:** A previous draft of this design proposed `Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'` at the end of the install section. That has a real correctness regression: `wails_tools.nsh` already specifies `RequestExecutionLevel "admin"`, so the installer always runs elevated; an `Exec` from inside the install Section uses `CreateProcess` with the parent's primary token, meaning the relaunched RedShell would inherit the admin/elevated token. WebView2 sandbox semantics change under elevation, file writes from the app land in admin-only locations, and the user has no visible cue that the app is running elevated. That is unacceptable for v1.

Three alternatives to drop-relaunch were considered:

- **Spawn a non-elevated launcher from the running app before quitting** (small Go helper or `cmd.exe /c "timeout 2 && start ... RedShell.exe"`): launcher inherits the medium-integrity token of the still-running RedShell, polls for both PIDs and the elevated installer to exit, then launches RedShell at medium integrity. Works, but introduces a third process and timing logic. Defer to v2 if user demand justifies it.
- **NSIS UAC plugin / `ShellExecAsUser` to de-elevate inside NSIS**: needs a plugin dependency and is fiddly; deferred.
- **Drop auto-relaunch (chosen)**: zero new moving parts, no integrity-token bug. The cost is a single user action — clicking the Start menu shortcut — which is acceptable for an in-app updater that already requires a UAC click.

### Decision 3b: NSIS install Section sleeps briefly to let the running app release its file lock

**Choice:** Add `Sleep 2000` at the very top of the install `Section` in `build/windows/installer/project.nsi`, before any `SetOutPath` / `wails.files` write. This gives the just-quitting RedShell process up to 2 seconds to release its lock on `RedShell.exe` before the installer attempts to overwrite it.

**Rationale:** When the updater calls `quitApp()` (Wails `runtime.Quit`), the running process needs a small wall-clock window to actually exit and release the OS-level executable lock. NSIS's `File` directive does not retry on lock failure — if the elevated installer reaches the write before our process is gone, the install Section aborts mid-flight. Three approaches were considered:

- **Pass our PID via an installer arg and use `${WaitForSingleObject}` from `WinAPI.nsh`**: most precise, but the wails-generated `wails_tools.nsh` doesn't include `WinAPI.nsh` and adding it requires careful template management.
- **Bundle the `nsProcess` NSIS plugin** to find/kill `RedShell.exe`: works, requires plugin shipping in the build pipeline.
- **`Sleep 2000` at top of install Section (chosen)**: zero plugin dependency, single line of NSIS, well within typical Wails OnShutdown duration (sub-second). If the lock is still held after 2s, the install fails fast and the user sees the failure when they reopen RedShell and find the version unchanged. Documented as a known limitation.

The `Sleep` only fires for installer-driven invocations because interactive installs already happen with the app closed (the user clicks the .exe themselves). It is therefore harmless for the existing interactive install UX.

### Decision 4: New error stages `installer-download` and `installer-spawn` in the event payload

**Choice:** Extend the documented `stage` enum in `updater:error` events to include `installer-download` (when the installer asset can't be downloaded or its URL can't be resolved) and `installer-spawn` (when `ShellExecuteW` returns failure or the user declines the UAC prompt).

**Rationale:** Frontends can already render unknown stages as opaque error strings, but adding explicit stages lets the UI offer better remediation copy ("UAC was declined — try again and click Yes"). Naming the stages distinctly from the portable equivalents (`download`, `spawn`) means analytics and logs can separate the two pathways without parsing message strings. We deliberately do NOT introduce new top-level event names — this keeps the `useUpdater` composable's status state machine unchanged.

### Decision 5: `State.BuildKind` is exposed; `State.ManualRequired` is tightened to apply only to portable builds

**Choice:** Add `BuildKind string` to `updater.State`. Tighten `ManualRequired` evaluation: it is now true only when `IsPortable() && !IsWritable(filepath.Dir(s.exePath))`. Today's evaluation is just `!IsWritable(...)`, which would incorrectly flag installer builds in `Program Files` as manual-required and defeat the purpose of this change. Frontend gates the manual-required banner on `manualRequired` (unchanged signal name, tightened condition) and gates the UAC hint on `buildKind === "installer"` (new).

**Rationale:** Keeping the `ManualRequired` field name preserves the public Wails-binding shape. Changing only the *condition* — explicitly OR-ing in `IsPortable()` — means the frontend `manualRequired.value` reactive ref keeps its current name and reactivity wiring; only its truthiness pattern changes. Tray gating in `main.go` switches from `!updaterApp.ManualRequired()` to `updaterApp.AutoUpdateAvailable()` (a new accessor that returns `IsInstaller() || !ManualRequired()`). The OR-with-`IsPortable()` change must also be reflected in `Service.Start()` — the `IsWritable` probe must be skipped for installer builds, so the `updater:manual-required` event never fires for them.

## Risks / Trade-offs

- **[UAC declined by user]** → Surface `updater:error` with stage `installer-spawn` and message "User cancelled elevation"; UI offers a Retry button. The previous version is left fully intact.
- **[Installer overwrite races with the still-running app]** → Mitigated by always setting `inProgress` and calling `quitApp()` *before* `SpawnInstaller` returns to its caller. We also rely on NSIS's standard file-locking retry (NSIS `OutPath` writes wait briefly for locks). If we observe install failures from this race in practice, fallback is to add a small `time.Sleep(500ms)` between quit and ShellExecute, which we explicitly do not pre-optimize.
- **[Older installer builds don't auto-relaunch]** → A user updating from a build that predates this change to a build that includes both the in-app updater AND the relaunch line will still need to manually relaunch the first time, then in-app update for every subsequent release. Document this in `docs/updater.md`.
- **[Antivirus / SmartScreen flags newly downloaded installer]** → The installer asset is already a published GitHub/GitLab release artifact; its reputation is the same as a manually downloaded install. SmartScreen may show a warning the first time, same as today.
- **[ldflag misconfigured at build time]** → If the build command forgets the `-X` flag, an installer build will run with `BuildKind="portable"` and degrade to manual-required (current behavior). This is safe — the failure mode is exactly today's behavior, not a destructive action. We add an explicit Make/Task target so the flag is hard to forget, and a smoke test in CI that asserts `BuildKind == "installer"` after the installer build.
- **[Tests cannot exercise ShellExecuteW]** → We inject `installerSpawn func(installerPath string, args []string) error` via `Options`, defaulting to the real `ShellExecuteW` wrapper on Windows. Tests assert the function is called with the expected path and args after a verified download, without actually spawning anything.

## Migration Plan

1. Backend changes first (BuildKind, installer pathway, tests). Until the build command is updated, all builds default to `portable` — zero behavior change.
2. Frontend changes (state shape, gating). Same: until something flips `BuildKind`, the UI is identical.
3. NSIS `project.nsi` relaunch line. Safe to ship before the updater feature lands.
4. Build pipeline change: pass the ldflag for the installer-targeted build. After this point, the next installer release will self-update.
5. Document the one-time manual download requirement for users on the previous installer version (a release note item).

Rollback: revert the build pipeline change so installer builds ship without the `installer` BuildKind. They degrade to manual-required exactly as today.

## Open Questions

- Do we want to surface the "Install will require Windows UAC permission" hint inline in `UpdatesTab` and `UpdateAvailableBanner`, or only in a tooltip on the Update button? The proposal leans toward inline; happy to revisit during UI implementation.
- Should we extend `InstallerAssetNameFor` to also handle ARM64 (`RedShell-arm64-installer.exe`) now, or wait until ARM64 release builds exist? Lean toward "add the function signature now, return error for unsupported arches, ship with AMD64 only".
- Is there a value in keeping the `manual-required` event for portable builds that fail the writability probe AFTER this change ships? Yes — preserved as-is. No deprecation in this proposal.
