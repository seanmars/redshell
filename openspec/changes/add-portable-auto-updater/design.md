## Context

RedShell currently distributes Windows builds two ways: a portable single-binary `redshell.exe` and an NSIS installer `RedShell-amd64-installer.exe`, both published to the project's GitHub and GitLab release pages. Releases happen by hand: build locally with `wails build`, then upload to both forges. There is no in-app awareness of newer versions; users upgrade by manually re-downloading.

The codebase already exposes `GetAppVersion()` from `app/system.go` (sourced from `wails.json`'s `productVersion`), so the "current version" half of an updater is solved. The Windows tray manager (`internal/tray/`) and preferences service (`internal/preferences/`) already model the patterns this design needs to extend: build-tagged OS-specific code, observer notifications on preference change, JSON-backed persistence with default fallback.

The change targets portable users only. NSIS-installed users typically run from `C:\Program Files\seanmars\RedShell\`, where the user-mode process cannot rename or write files; this design SHALL detect that case and surface a "manual update required" message instead of attempting the rename trick.

The publisher (the project maintainer) is the only person who controls release artifacts. There is no plan to introduce code signing in the foreseeable future, so cryptographic integrity rests entirely on a SHA256 sidecar checksum file shipped alongside binaries.

## Goals / Non-Goals

**Goals:**
- Background polling against a user-selected source (GitHub or GitLab) at a sensible interval, with manual "Check now" available.
- Two parallel sources whose latest versions may diverge; the user picks one as active. The Settings UI peeks at both for comparison.
- Portable Windows replacement that survives a running `.exe` lock by using rename-then-spawn, with no admin requirement and no separate updater binary.
- SHA256 verification of every downloaded binary against a publisher-controlled `checksums.txt` sidecar, since the project is unsigned.
- Optional updates: user can `Skip` a specific version, defer with `Later`, or install with `Update Now`. No forced updates.
- Clean integration with existing tray, preferences, and close-behavior systems so the update flow does not collide with `OnBeforeClose` prompting.

**Non-Goals:**
- macOS / Linux portable replacement (build-tagged stub returns a "platform not supported" error; spec coverage Windows-only for v1).
- Installer-driven updates (NSIS users out of scope; detected and reported, not actioned).
- Differential / binary-patch updates (full asset re-download every time).
- Release channels beyond `latest` (no beta / nightly / canary).
- Auto-rollback on failed update (failure leaves both `.old` and original `.exe`; documented recovery is to delete `.old`).
- Telemetry beyond the necessary HTTP requests to the configured release host.

## Decisions

### Decision: Portable rename trick over self-replace-in-place

Windows allows renaming a running `.exe` (since Vista), but not deletion or overwrite. The flow is:

1. Download asset to `redshell.exe.partial` next to the running binary.
2. Stream-hash with SHA-256 during download; verify against `checksums.txt` line for the asset filename.
3. Rename `.partial` -> `.new` (atomic; commits the verified payload).
4. Rename current `redshell.exe` -> `redshell.exe.old` (works while running).
5. Rename `redshell.exe.new` -> `redshell.exe`.
6. `exec.Command(newExePath).Start()` to spawn the replacement process.
7. Set the `inUpdate` flag and call `runtime.Quit(ctx)` so `OnBeforeClose` short-circuits.
8. New process at startup runs `updater.CleanupStale()` which deletes any `*.old` next to the running exe.

**Why not self-replace via `os.Rename` of a single in-place download?** Same target name as the running exe -> rename fails. The `.new` intermediate is required.

**Why not write a launcher.bat / launcher.exe?** Spawning a child process before quitting is simpler, leaves no temp script on disk, and avoids cross-platform shell-quoting headaches. The new process briefly co-exists with the old one (a few hundred milliseconds) which is harmless because there is no single-instance lock today.

**Alternative considered: copy + replace via `MoveFileEx(MOVEFILE_REPLACE_EXISTING|MOVEFILE_DELAY_UNTIL_REBOOT)`**. Rejected because it forces the user to reboot to actually pick up the new version, which contradicts the "Update Now" UX. Also: not testable without a reboot.

### Decision: Active-source model with parallel GH / GL providers

Users pick exactly one source as active. The background ticker polls only that source. The Settings -> Updates tab makes a one-shot peek at both sources when opened so the user can see which is fresher before switching.

```
type Provider interface {
    Name() string                                   // "github" | "gitlab"
    LatestRelease(ctx context.Context) (Release, error)
}

type Release struct {
    Version     string    // semver, e.g. "v0.5.0" (kept with leading "v" to match git tags)
    PublishedAt time.Time
    Notes       string    // raw markdown, frontend renders
    AssetURL    string    // https URL to redshell-windows-amd64.exe
    AssetName   string    // "redshell-windows-amd64.exe"
    AssetSize   int64
    ChecksumsURL string   // https URL to checksums.txt
}
```

**Why not "merge both, take the newer"?** Confusing: a publisher push to GH-only would silently move every user to a version that never landed on GL. Active-source keeps the source-of-truth explicit and the user in control.

**Why not auto-fallback to the other source on error?** Silently switching sources hides connectivity problems and could mask a publisher mistake. Errors surface to the UI; the user decides whether to switch.

### Decision: SHA256 verification via `checksums.txt` sidecar

Each release publishes `checksums.txt` next to the binary assets, in `sha256sum`-compatible format:

```
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  redshell-windows-amd64.exe
da39a3ee5e6b4b0d3255bfef95601890afd80709  RedShell-amd64-installer.exe
```

The updater downloads `checksums.txt` after resolving the release, parses it line-by-line (split on any whitespace, take `parts[0]` as hash and `parts[1]` as filename), finds the entry for the OS/arch-appropriate asset, and verifies the streamed download against that hash.

**Why this format?** Compatible with `sha256sum -c checksums.txt` for hand verification by power users, easy to produce in CI (`sha256sum redshell-windows-amd64.exe > checksums.txt`), and trivial to parse in Go without a markdown library.

**Why not embed hashes in release notes?** Requires a markdown table parser and is fragile to format changes. The sidecar file is a separate, machine-checkable contract.

**Why not a `.sig` file?** That implies signing, which the project explicitly does not plan to do. A SHA256 sidecar is the strongest integrity layer available without keys.

**Failure mode:** If `checksums.txt` is missing, the download is rejected and an error surfaces to the UI. There is no "best-effort, install anyway" fallback.

### Decision: Background ticker with cached `lastCheckedAt`, debounced on startup

On `OnStartup`, the updater service:
1. Reads `prefs.AutoUpdate.lastCheckedAt`.
2. If `now - lastCheckedAt >= intervalHours`, fires an immediate check.
3. Schedules a `time.Ticker` at `intervalHours` for the rest of the session.

`intervalHours` is constrained to the preset values `[1, 6, 12, 24, 168]` to keep load on GH (60 req/hr unauthenticated) and GL predictable. Default `6`.

When the user changes `source` or `intervalHours`, the service:
- Stops the current ticker.
- Triggers an immediate check on the new source.
- Reschedules with the new interval.

**Why presets, not a free-form spinner?** A user could set `0` and DDoS GitHub. Presets bound the worst case.

**Why no exponential backoff on errors?** A simple stop-on-error and retry-on-next-tick is enough. The intervals (>= 1 hour) already provide natural backoff.

### Decision: `golang.org/x/mod/semver` for version comparison

Versions are git tags like `v0.5.0` or `v0.5.0-rc1`. `semver.Compare` handles prerelease ordering correctly (e.g. `v0.5.0-rc1 < v0.5.0`).

**Why not roll a comparator?** Prerelease ordering is the kind of thing you get wrong once and never want to debug. The dependency is already transitively pulled in via the Go toolchain ecosystem.

### Decision: OS-specific code via build tags, mirroring `internal/tray/` pattern

```
internal/updater/
  rename_windows.go    // build constraint: //go:build windows
  rename_other.go      // build constraint: //go:build !windows
```

`rename_other.go` returns a sentinel error `ErrPlatformUnsupported`. The service layer turns that into a UI-visible "Auto-update is only available on Windows" message.

**Why not skip non-Windows compilation entirely?** Because Go tests run on the developer's machine, which may be macOS (the maintainer might run `go test ./...` from a Mac). Stub-on-other-platforms keeps the package compilable everywhere.

### Decision: Detect installer-installed mode, refuse to update

At service start, the updater checks whether the directory holding the running exe is writable by the current user (write a zero-byte temp file, delete it, or use `os.Stat` + ACL probe). If write fails:
- Updater does not register the ticker.
- A one-time UI banner explains "This is an installed build; use the installer to update or download the portable build instead."
- The user can dismiss; the banner respects `skipVersion` semantics (versioned dismiss).

**Why not detect via NSIS uninstall registry key?** Reading `HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\seanmarsRedShell` would work but couples the updater to NSIS layout. A write-probe is generic and survives any future installer.

### Decision: `inUpdate` flag bypasses `OnBeforeClose` prompt

The current `OnBeforeClose` either applies `closeBehavior` (`exit`, `minimize-to-tray`) or emits the `unset` prompt event. During an update swap, neither of these is correct: the new process must be allowed to start and the old one must terminate without UI.

The updater service exposes `InProgress() bool`. `OnBeforeClose` checks it first and, when true, returns `false` (allow close) without consulting preferences or emitting events. This mirrors the existing "explicit quit" flag pattern used by the tray Quit menu item (see `system-tray` spec).

### Decision: Preferences shape under existing `AutoUpdate` key

```jsonc
{
  "closeBehavior": "minimize-to-tray",
  "autoUpdate": {
    "enabled": true,
    "intervalHours": 6,
    "source": "github",
    "githubRepo": "seanmars/redshell",
    "gitlabHost": "https://gitlab.com",
    "gitlabProject": "seanmars/redshell",
    "skipVersion": "",
    "lastCheckedAt": "2026-05-07T00:00:00Z"
  }
}
```

`githubRepo`, `gitlabHost`, `gitlabProject` are settable so a fork or self-hosted GitLab user can point the updater at their own release page without rebuilding. They have sensible defaults and SHALL NOT be exposed in the v1 Settings UI (advanced users edit the JSON directly).

`skipVersion` stores a single semver string. If `release.Version == prefs.AutoUpdate.skipVersion`, the available toast is suppressed but the "Latest version" indicator in Settings still renders. Skipping a version persists across sessions and across source switches (because publisher A and publisher B may both publish the same skip-worthy version).

## Risks / Trade-offs

- **No code signing -> SmartScreen / AV warnings**: First-run after replacement may trigger Defender SmartScreen "Unrecognized app" or third-party AV quarantine of `redshell.exe.new`. Mitigation: download to `.partial` and only rename to `.new` after SHA256 passes, narrowing the AV scan window. Document the risk in README; long-term answer is signing, deferred.
- **Two-process race during swap**: For a few hundred ms after `exec.Command(newExePath).Start()`, two RedShell processes exist. Mitigation: rely on absence of single-instance lock; if one is added later, gate it on a `--from-update` argv flag passed by the spawning process so the new one knows to wait for the old to release.
- **Antivirus blocks rename**: A small minority of AV products lock newly written executables. Mitigation: catch rename errors, leave `.partial` and original in place, surface a clear error UI ("Antivirus may have quarantined the download"). User can retry.
- **Publisher publishes to one source but not the other**: User on the lagging source sees stale "latest" forever. Mitigation: Settings tab side-by-side peek of both sources surfaces the discrepancy; user can switch. Acceptable because parallel sources is an explicit decision.
- **MITM / hijacked release**: HTTPS plus SHA256 against a publisher-controlled `checksums.txt` is the only defense. If the attacker compromises the GH/GL account they can replace both the binary and the checksum file. Mitigation: scoped-token publishing, branch protection on the repo. Not solvable by the updater itself.
- **GitHub rate limit (60 req/hr unauthenticated, per IP)**: At interval >= 1h and one user per IP, well within budget. Multiple users behind shared NAT could collide. Mitigation: send `If-Modified-Since` / `ETag` headers; treat 304 as "no change" without consuming the polling budget materially.
- **`exec.Command(newExePath).Start()` argv inheritance**: Spawned child must NOT inherit the parent's stdin/stdout/stderr if those are pipes (rare for desktop GUI apps but possible under wrappers). Mitigation: explicitly null those `os/exec.Cmd` fields.
- **Partial download retained on crash**: If the app crashes mid-download, `.partial` lingers. Mitigation: cleanup deletes both `*.old` and `*.partial` siblings on startup.
- **Corrupted `checksums.txt`**: If the parser cannot find the asset filename in the file, treat as a failed download. Do not fall through to "install without verification".
- **Disk full mid-download**: Standard `io.Copy` error path; surface to UI, leave `.partial` for the cleanup step on next start.

## Migration Plan

This change is purely additive. There is no migration of existing user state.

- **Preferences forward-compat**: Existing `~/.redshell/preferences.json` files without an `autoUpdate` block SHALL load successfully, with all `AutoUpdate` fields defaulting to the spec-defined defaults (`enabled: true`, `intervalHours: 6`, `source: "github"`, etc.). The first preference write back to disk SHALL include the populated block.
- **Existing builds**: Users on pre-update-feature versions have no way to learn about the new feature in-app on their current install. They learn through normal release-notes channels (GitHub release page, GitLab release page). After they upgrade once manually, all future upgrades go through the new flow.
- **Release process change**: Starting with the first release that ships this feature, every release SHALL include a `checksums.txt` file. The release-process documentation update is part of this change (`tasks.md` step 10).
- **Rollback**: If a release ships with a broken updater, users on stable older versions are unaffected (their old code does not poll). Users who upgraded to the broken release can disable auto-update in Settings or delete `~/.redshell/preferences.json` to fall back to defaults; manual re-download remains the escape hatch.

## Open Questions

- **`checksums.txt` filename collision with NSIS installer**: The same `checksums.txt` covers both `redshell-windows-amd64.exe` and `RedShell-amd64-installer.exe`. The updater only uses the portable line; the installer line is informational. Confirmed acceptable.
- **GitLab API auth**: gitlab.com public projects do not require auth for `releases` API, but a self-hosted private GL would. v1 supports unauthenticated access only; a `gitlabToken` preference field is a future enhancement.
- **Tray notification on update available**: should the tray icon paint a small overlay badge when an update is available? Out of scope for v1 (toast + tray menu item is enough); revisit if users request it.
