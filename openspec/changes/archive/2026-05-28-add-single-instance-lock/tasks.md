## 1. Parent-PID wait helper

- [x] 1.1 Add a `waitForParentExit(pid int, timeout time.Duration)` helper in a Windows-tagged file (`//go:build windows`) using `windows.OpenProcess(SYNCHRONIZE, false, pid)` + `WaitForSingleObject` with a millisecond timeout; close the handle on return.
- [x] 1.2 Add a no-op stub of the same signature in a `//go:build !windows` file so non-Windows builds compile.
- [x] 1.3 Add a small argv parser that extracts `--wait-parent-pid=<n>` from `os.Args` (returns pid and ok); unit-test it for present / absent / malformed values.
- [x] 1.4 Unit-test the wait helper's timeout path (a PID that never exits returns at the timeout, not indefinitely) on Windows; keep it fast (sub-second timeout in the test).

## 2. Single-instance lock (production-gated)

- [x] 2.1 Add `singleinstance_prod.go` (`//go:build production`) exposing `newSingleInstanceLock(onSecond func(options.SecondInstanceData)) *options.SingleInstanceLock` that returns a configured lock with the stable `UniqueId` constant.
- [x] 2.2 Add `singleinstance_other.go` (`//go:build !production`) returning `nil` from the same `newSingleInstanceLock` signature.
- [x] 2.3 Define the stable `UniqueId` constant (`"com.seanmars.redshell"`) in `main.go` next to the wails.json product-info helpers.

## 3. Wire into main.go startup

- [x] 3.1 At the top of `main()`, before `wails.Run`, parse `--wait-parent-pid`; if present, call `waitForParentExit(pid, ~10s)` so the post-update relaunch waits for the old process to release the lock.
- [x] 3.2 Add a package-level `var appCtx context.Context`; set it in the `OnStartup` hook so the second-instance callback can use a captured context.
- [x] 3.3 Set `options.App.SingleInstanceLock = newSingleInstanceLock(onSecondInstance)` where `onSecondInstance` unminimizes + shows the window via the existing tray `ShowWindow()` path (falling back to `runtime.WindowUnminimise` + `runtime.WindowShow` on `appCtx`).
- [x] 3.4 During implementation, verify whether `WindowShow` raises to foreground; if not, add an explicit foreground call (e.g. brief `WindowSetAlwaysOnTop` toggle) inside the callback.

## 4. Updater relaunch handshake

- [x] 4.1 In `internal/updater/service.go` `defaultSpawn`, append `--wait-parent-pid=<os.Getpid()>` to the relaunched binary's args so the new process waits for the old one to exit.
- [x] 4.2 Update or add an updater test asserting the spawned command line includes `--wait-parent-pid` with the current PID (use the existing `Spawn` injection point in `Options`).

## 5. Verification

- [x] 5.1 Run `go fmt` and `go vet` on all changed Go files; confirm `go test ./...` passes with no build tag (no-op variants compile).
- [ ] 5.2 Production build (`wails build`): launch the app, launch it a second time, confirm the first window comes to the foreground and the second process exits with no new window.
- [ ] 5.3 Repeat 5.2 with the first instance minimized to tray and again minimized to the taskbar; confirm it unhides/unminimizes and comes forward.
- [ ] 5.4 Dev mode (`wails dev` or a `dev`-tagged build): confirm two instances can run simultaneously.
- [ ] 5.5 End-to-end update: trigger an update so the updater swaps and relaunches; confirm exactly one instance is running afterward (the relaunch did not self-terminate).
