## Why

The in-app updater currently only works for the Windows portable build. Installer-installed builds (NSIS, default location `C:\Program Files\RedShell\`) hit the runtime write-probe in `internal/updater/install_dir.go`, get flagged as `manual-required`, and fall back to "go to the release page and download the portable build" messaging. This forces installer users — likely the majority once installer distribution is the recommended path — into a manual download/replace flow for every release, which is the opposite of what an in-app updater is meant to deliver.

We want the installer-installed variant to also self-update from inside the app, using the same release pipeline and `checksums.txt` verification that already covers the portable variant.

## What Changes

- Add a build-kind discriminator (`portable` | `installer`) baked into the binary at build time so the running process knows which install pathway it represents, replacing "writability of exe dir" as the primary signal for which install flow to use.
- Add an installer-asset install pathway: download `RedShell-amd64-installer.exe`, verify against `checksums.txt`, then ShellExecute with the `runas` verb to elevate, pass NSIS silent flags (`/S`), quit the running app so the installer can overwrite the running binary.
- Keep the existing portable rename-trick swap (`Swap` in `rename_windows.go`) unchanged for portable builds; only the install entrypoint dispatches by `BuildKind`.
- Keep the writability probe as a defensive secondary signal: a portable build placed in a non-writable directory still degrades to `manual-required`. An installer build does NOT use the probe — it always uses the installer pathway.
- Update tray + UI manual-required gating: installer builds register the tray "Check for Updates" item, surface the Settings -> Updates tab actions, and remove the "download portable" warning. The UI may show a one-line note that the update will trigger a UAC prompt.
- Update the NSIS installer config (`build/windows/installer/project.nsi` plus the wails-generated `wails_tools.nsh` consumers) to relaunch RedShell at the end of a silent install so the user transparently lands back in the new version.
- Update `docs/updater.md` §20 (out-of-scope list) to remove "Installer 安裝版本的更新", and add a new section documenting the installer install pathway.

## Capabilities

### New Capabilities

(none — this is an enhancement to an existing capability, not a new one)

### Modified Capabilities

- `auto-update`: replace the single "non-writable install dir disables auto-update" rule with a build-kind-aware dispatch. Installer builds get a new install pathway (download installer asset + silent elevated re-install) rather than the rename swap. Manual-required is now a fallback only for portable builds in non-writable directories.

## Impact

- **Backend (Go)**:
  - `internal/updater/types.go`: add `InstallerAssetNameFor(goos, goarch)` and a `BuildKind` constant injected via `-ldflags -X`.
  - `internal/updater/service.go`: branch on `BuildKind` in `Start` (skip writability probe for installer builds) and in `install` (dispatch to installer-install vs portable swap).
  - New file `internal/updater/installer_install_windows.go` (build-tag gated): ShellExecute with `runas` verb, pass `/S` argument, return after spawn so the caller can quit the app.
  - `app/updater.go` / `main.go`: surface `BuildKind` in `State` so the frontend knows which mode is active; tray gating uses `BuildKind` instead of `ManualRequired`.
- **Frontend (Vue/TS)**:
  - `frontend/src/composables/useUpdater.ts` and `stores`: expose `buildKind` from state; conditionally hide the manual-required warning for installer builds.
  - `frontend/src/components/settings/UpdatesTab.vue`: drop the manual-required alert for installer builds, optionally show a small "Updating triggers a Windows UAC prompt" hint near the Update button.
  - `frontend/src/components/system/UpdateAvailableBanner.vue`: same gating.
- **Build / Release**:
  - `wails.json` build hooks (or `Taskfile`/`Makefile` if present) need to set `-ldflags "-X 'redshell/internal/updater.BuildKind=installer'"` for the installer-targeted build, and either default or explicit `portable` for the portable build.
  - `build/windows/installer/project.nsi`: append a `nsExec` / `Exec` instruction in the install section to relaunch RedShell at end of install when run in silent mode triggered by the updater.
- **Docs**:
  - `docs/updater.md`: §20 (out-of-scope) and add a new section describing the installer-install pathway, its UAC requirement, and the NSIS silent-install contract.
- **Tests**:
  - `internal/updater/install_dir_test.go`: stays valid; add tests for `BuildKind` dispatch.
  - New tests for installer install path: cannot fully exercise ShellExecute, but inject `installerSpawn` function and verify the service calls it with the correct path and arguments after a verified download.
- **Backwards compatibility**: portable users see no behavior change. Installer users on the previous version will need to do one manual download to reach the first version that ships this feature; subsequent updates are in-app.
- **Risks**: UAC prompt friction on every update (acceptable, documented), and the installer overwrite race with the running process (mitigated by quitting the app before the silent installer copies files; see design).
