## Context

RedShell is a Wails v2 (v2.12.0) desktop app that stays resident in the Windows system tray. The close button can minimize-to-tray (`internal/preferences` `closeBehavior`), so the app is frequently running but window-hidden. There is currently no single-instance guard — relaunching the binary opens another window/process.

Two existing pieces of the shell constrain this design:

1. **Tray lifecycle** (`internal/tray`, `//go:build windows`): already owns `ShowWindow()` → `runtime.WindowShow(ctx)` for the tray-hidden case (`tray_windows.go:87`). The single-instance foreground-raise should reuse this path.
2. **Auto-updater relaunch** (`internal/updater/service.go`): `defaultSpawn` (line 193) does `exec.Command(exePath).Start()` to launch the new binary, then the service calls `quitApp()` to quit the old one. For a few hundred milliseconds two processes coexist. The archived auto-updater design (`openspec/changes/archive/2026-05-08-add-portable-auto-updater/design.md:170`) explicitly anticipated this and pre-wrote the mitigation: "if [a single-instance lock] is added later, gate it on a `--from-update` argv flag passed by the spawning process so the new one knows to wait for the old to release."

Wails v2.12.0 ships a built-in single-instance facility (`options.App.SingleInstanceLock`). On Windows (`internal/frontend/desktop/windows/single_instance.go`) it: creates a named mutex `wails-app-<UniqueId>sim`; on `ERROR_ALREADY_EXISTS` finds the primary's hidden message window, sends `os.Args[1:]` + working dir via `WM_COPYDATA`, then `os.Exit(0)`; the primary receives the data and invokes `OnSecondInstanceLaunch(SecondInstanceData)`.

## Goals / Non-Goals

**Goals:**
- Production builds run at most one instance; a second launch raises the existing window (even when tray-hidden or minimized) and the duplicate exits.
- Development and tooling builds (`wails dev`, plain `go build`/`go run`/`go test`/`go vet`) keep the current unrestricted behavior.
- The auto-update relaunch continues to work — exactly one instance running after a swap completes.

**Non-Goals:**
- Cross-platform enforcement. The current packaged target is Windows; the lock is wired only where Wails provides the Windows implementation. macOS/Linux fall back to no enforcement (consistent with the tray `//go:build windows` pattern).
- Passing meaningful data between instances (e.g. open-file argv handling). Only the wake-and-raise behavior is in scope.
- Any change to the close-behavior / minimize-to-tray preference semantics.

## Decisions

### D1: Use Wails' built-in `SingleInstanceLock`, not a hand-rolled mutex + IPC

`options.App.SingleInstanceLock{ UniqueId, OnSecondInstanceLaunch }` already implements the named-mutex acquisition, the hidden message-window, the argv/cwd forwarding, and the `os.Exit(0)` of the duplicate. This is precisely the "detect running instance → notify it → terminate self" requirement. Hand-rolling a mutex + named pipe would re-implement working, tested platform code for no benefit.

- **Alternative considered — manual `windows.CreateMutex` + custom IPC:** more code, duplicates Wails internals, and the only thing it would buy (finer control over the duplicate's exit) is not needed for the normal case and is handled separately for the updater case (see D3).

### D2: Gate the lock on the `production` build tag via `//go:build !production`

Wails ships `app_dev.go` (`//go:build dev`) and `app_production.go` (`//go:build production`); exactly one is compiled, so `wails build` sets `-tags production` and `wails dev` sets `-tags dev`. We exploit this:

- `singleinstance_prod.go` (`//go:build production`) returns a configured `*options.SingleInstanceLock`.
- `singleinstance_other.go` (`//go:build !production`) returns `nil`.

`main.go` sets `SingleInstanceLock: newSingleInstanceLock(...)`; a `nil` value disables enforcement. Using `!production` (rather than `dev`) for the no-op variant is deliberate: plain `go build`, `go test ./...`, and `go vet ./...` (which carry no Wails tag, and which project rules require running on changed Go files) must still compile and must NOT enforce the lock. `!production` covers dev + test + vet + plain builds; `production` covers only the shipped binary.

- **Alternative considered — runtime `runtime.Environment(ctx).BuildType` check:** rejected because `SingleInstanceLock` is configured in the `options.App` struct passed to `wails.Run`, before any context exists. Build-time gating is the only clean fit.

### D3: Survive the updater swap with a `--wait-parent-pid=<oldpid>` handshake

Without coordination, the updater's relaunched binary would start while the old process is still alive, hit the mutex, forward its argv to the dying old instance, and `os.Exit(0)` — ending with zero instances. Fix:

- `internal/updater` `defaultSpawn` passes `--wait-parent-pid=<current-pid>` to the spawned binary.
- `main.go`, **before** `wails.Run`, parses argv for `--wait-parent-pid`. If present, it waits (bounded, ~10s) for that PID to exit, then proceeds. By the time `wails.Run` calls `SetupSingleInstance`, the old mutex is released and the new process cleanly becomes the primary.
- Wait mechanism is PID-based on Windows: `windows.OpenProcess(SYNCHRONIZE, ...)` + `WaitForSingleObject(handle, timeoutMs)`. Lives in a `//go:build windows` helper with a no-op stub elsewhere. The flag is a generic relaunch primitive (not named `--from-update`) so it composes if a similar handshake is ever needed again.

- **Alternative considered — fixed `time.Sleep` before `wails.Run`:** fragile; if old-process shutdown stalls the sleep is either too short (race persists) or needlessly slow.
- **Alternative considered — re-derive and poll Wails' internal mutex name (`wails-app-<id>sim`):** couples us to a private naming convention that can change between Wails versions. PID-wait is decoupled from Wails internals.

### D4: Foreground-raise in `OnSecondInstanceLaunch`

The callback (`func(SecondInstanceData)`) has no context parameter, so it closes over a Wails context captured in `OnStartup` (the existing `*App` wrappers already capture ctx this way; `main.go` will hold a package-level `var appCtx context.Context` set in the `OnStartup` hook). The callback:
1. `runtime.WindowUnminimise(appCtx)` then `runtime.WindowShow(appCtx)` — reuse `trayMgr.ShowWindow()` where it already performs the tray-hidden show so behavior matches the tray "Show RedShell" item.
2. If `WindowShow` alone does not steal focus from a foreground app, add an explicit foreground call (`runtime.WindowSetAlwaysOnTop(true)` toggled off, or a platform foreground call). This is verified empirically during implementation; the spec only requires the window become visible and frontmost.

### D5: `UniqueId` is a stable, app-scoped string

Use `"com.seanmars.redshell"` (Wails namespaces it to mutex `wails-app-com.seanmars.redshellsim`). Defined as a constant in `main.go` next to the `GetAppVersion()` / wails.json product info so it is discoverable and stable across releases (the mutex name must not change between versions or two different versions could co-run).

## Risks / Trade-offs

- **Updater swap leaves zero instances** → Mitigated by D3 (`--wait-parent-pid` handshake). This is the single highest-risk interaction and is in-scope, not deferred. Verified by the manual update-flow test in tasks.
- **Foreground raise may not steal focus** (Windows foreground-lock rules) → Mitigated by reusing the tray show path and adding an explicit foreground call if `WindowShow` proves insufficient during verification. Worst case the taskbar button flashes — acceptable, still better than a second window.
- **Stale mutex if a process is killed hard** → The OS releases the named mutex when the owning process handle closes, even on hard kill, so a fresh launch acquires cleanly. No manual cleanup needed.
- **`--wait-parent-pid` PID reuse** → Bounded timeout caps the wait; if the PID was reused by an unrelated process the wait simply returns at timeout and startup proceeds. Low impact (worst case a ~10s delay on one relaunch).
- **Cannot unit-test the OS lock** → The lock and foreground-raise are OS-level and require two real processes. Mitigation: unit-test the argv parsing and the wait-helper timeout logic; cover the end-to-end behavior with explicit manual verification steps (see tasks).
- **Non-Windows builds get no enforcement** → Accepted non-goal, consistent with the existing tray stub pattern.

## Migration Plan

Additive and build-time-gated; no persisted state or schema changes. Rollback is reverting the diff. No user data migration. The `--wait-parent-pid` flag is internal (set only by the updater) and ignored by Wails production argv handling.

## Open Questions

- Whether `runtime.WindowShow` + `WindowUnminimise` is sufficient to raise to foreground on Windows, or an explicit foreground call is required — **resolved by Wails source inspection**: the Windows `ShowWindow()` implementation (`internal/frontend/desktop/windows/frontend.go:987-988`) ends with `SetForegroundWindow` + `SetFocus`, and `WindowUnminimise` calls `Restore()`, so `WindowUnminimise` + `WindowShow` covers both the tray-hidden and taskbar-minimized cases and brings the window to the front. No extra `WindowSetAlwaysOnTop` toggle is needed. Final GUI confirmation rolls into manual verification tasks 5.2/5.3.
