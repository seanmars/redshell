## Why

Today RedShell allows any number of copies of the binary to run at once. In a packaged production build this is confusing: a user who double-clicks the icon while the app is already minimized to the system tray gets a second window (or a second tray icon) instead of the running app coming forward. Enforcing a single instance — and surfacing the already-running window — makes the desktop behavior match user expectations for a resident tray app. Development must stay exempt so `wails dev` and ad-hoc `go run` debugging can launch freely.

## What Changes

- Production builds SHALL enforce a single running instance. When a second launch is attempted, the already-running instance SHALL be brought to the foreground (including when it is hidden in the tray or minimized) and the second process SHALL terminate itself.
- Development builds (`wails dev`, plain `go build`/`go run`/`go test`) SHALL NOT enforce the lock, so multiple instances remain allowed.
- The auto-updater relaunch path SHALL be made compatible with the lock: the updater spawns the new binary and then quits the old one, so the new binary must wait for the old process to exit before acquiring the lock — otherwise the post-update relaunch would detect the still-alive old instance and self-terminate, leaving zero instances running.

## Capabilities

### New Capabilities
<!-- none — this is shell-level startup behavior layered onto the existing app shell -->

### Modified Capabilities
- `wails-app-shell`: add single-instance enforcement to app startup (production only), with foreground-raise of the existing instance and self-termination of the duplicate; extend the updater-relaunch coordination so the swap survives the lock.

## Impact

- `main.go` — set `options.App.SingleInstanceLock` (production only, via build-tag-gated helper) with an `OnSecondInstanceLaunch` callback that raises the existing window; add a pre-`wails.Run` guard that waits for the parent process to exit when relaunched by the updater.
- New build-tag-gated helper files (e.g. `singleinstance_prod.go` with `//go:build production` and `singleinstance_other.go` with `//go:build !production`) that return the configured lock or `nil`.
- `internal/updater/service.go` — `defaultSpawn` passes a `--wait-parent-pid=<oldpid>` argv to the relaunched binary so it can wait for the old process to release the lock.
- A small Windows helper to wait on a parent PID (`OpenProcess(SYNCHRONIZE)` + `WaitForSingleObject`) with a bounded timeout, plus a no-op stub elsewhere.
- Window-raise reuses the existing tray `ShowWindow` path (`internal/tray`) for consistency with the tray-hidden case.
- Dependencies: no new modules — relies on Wails v2.12.0 `options.SingleInstanceLock` (already vendored) and `golang.org/x/sys/windows` (already in the module graph via the tray/updater code).
