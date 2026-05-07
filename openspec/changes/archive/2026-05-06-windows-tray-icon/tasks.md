## 1. Preferences service

- [x] 1.1 Create `internal/preferences/service.go` with a `Service` struct that owns `~/.redshell/preferences.json`, plus `NewService()` (resolves home dir) and `NewServiceWithPath(path string)` (test seam mirroring `agent.NewSettingsServiceWithPaths`)
- [x] 1.2 Define exported types: `Preferences struct { CloseBehavior string \`json:"closeBehavior"\` }` and string constants `CloseBehaviorUnset = "unset"`, `CloseBehaviorExit = "exit"`, `CloseBehaviorMinimizeToTray = "minimize-to-tray"`
- [x] 1.3 Implement `Get() (Preferences, error)`: read file, return defaults (`CloseBehavior: "unset"`) when missing, return descriptive error on malformed JSON
- [x] 1.4 Implement `GetCloseBehavior() (string, error)` and `SetCloseBehavior(value string) error`; reject values outside the three constants with a typed error
- [x] 1.5 Implement `OnChange(cb func(Preferences))` observer registration and fire callbacks from inside `SetCloseBehavior` only when the value actually changes
- [x] 1.6 Add `internal/preferences/service_test.go` covering: missing file → defaults; round-trip set/get; malformed JSON → error; observer fires on change; observer does not fire on no-op set; invalid value → error and file untouched

## 2. Tray manager (Windows-first, with cross-platform stubs)

- [x] 2.1 Add `github.com/getlantern/systray` to `go.mod` (run `go get` and let `go mod tidy` resolve indirect deps); commit the updated `go.sum`
- [x] 2.2 Create the `internal/tray/` package with a public `Manager` interface: `Start(ctx context.Context, prefs *preferences.Service) error`, `Stop()`, `Available() bool`, `RequestExit()`, `ShowWindow()`, `HideWindow()`
- [x] 2.3 Add `internal/tray/tray_windows.go` with `//go:build windows` implementing `Manager` via `systray.Run`. The constructor returns the real implementation; `Start` launches `systray.Run` on a goroutine, registers menu items (Show RedShell, "Close button minimizes to tray" checkable, Quit RedShell), and wires callbacks to `runtime.WindowShow/Hide/Quit` and `prefs.SetCloseBehavior`
- [x] 2.4 Subscribe to `prefs.OnChange` from inside `Start` so the checkable item's state mirrors the persisted value when changed elsewhere (e.g. from the frontend modal)
- [x] 2.5 Add `internal/tray/tray_other.go` with `//go:build !windows` providing a no-op `Manager` whose `Available()` returns `false` and whose other methods are noops returning `nil`
- [x] 2.6 Add `internal/tray/assets/tray.ico` (32x32, optionally also 16x16 multi-resolution); embed it in `tray_windows.go` via `//go:embed assets/tray.ico` and pass the bytes to `systray.SetIcon`
- [x] 2.7 Add a brief `internal/tray/assets/README.md` noting that `tray.ico` is derived from `build/windows/icon.ico` and how to regenerate it

## 3. App-layer wrapper and Wails wiring

- [x] 3.1 Create `app/preferences.go` with `AppPreferencesApp` exposing bound methods `GetCloseBehavior()`, `SetCloseBehavior(value string)`, `RequestExit()`. Hold a `context.Context` set in `Startup(ctx)`, plus references to `*preferences.Service` and the tray `Manager`
- [x] 3.2 In `RequestExit`, set an internal `explicitQuit` flag and call `runtime.Quit(ctx)` so the close hook can recognize a programmatic quit
- [x] 3.3 Update `main.go` to construct `preferences.NewService()`, `tray.NewManager()` (returns the platform-correct impl), and `app.NewAppPreferencesApp(prefsSvc, trayMgr)`; bind it alongside the existing app wrappers
- [x] 3.4 Register `OnStartup` step that captures the Wails ctx into `AppPreferencesApp` and calls `trayMgr.Start(ctx, prefsSvc)` on Windows (no-op elsewhere because the stub returns immediately)
- [x] 3.5 Register `OnBeforeClose(ctx)` that returns `false` if the explicit-quit flag is set, otherwise reads `prefsSvc.GetCloseBehavior()` and: for `exit` returns `false`; for `minimize-to-tray` calls `runtime.WindowHide(ctx)` and returns `true`; for `unset` calls `runtime.EventsEmit(ctx, "tray:close-behavior-prompt")` and returns `true`
- [x] 3.6 Register `OnShutdown` step that calls `trayMgr.Stop()`
- [x] 3.7 Run `go fmt ./...` and `go vet ./...`; verify `go build ./...` succeeds on Windows and that `GOOS=darwin GOARCH=arm64 go build ./...` (or equivalent) compiles the non-Windows stubs

## 4. Frontend integration

- [x] 4.1 After `wails dev` regenerates bindings, confirm `frontend/wailsjs/go/app/AppPreferencesApp.{ts,js}` exists and exports `GetCloseBehavior`, `SetCloseBehavior`, `RequestExit`
- [x] 4.2 Create `frontend/src/stores/preferences.ts` (Pinia setup-style) wrapping the new bindings, exposing `closeBehavior` ref, `loadCloseBehavior()`, `setCloseBehavior(value)`, and `requestExit()`
- [x] 4.3 Create `frontend/src/components/system/CloseBehaviorPromptModal.vue` rendering an `AppModal` with two `AppButton`s ("Exit RedShell", "Minimize to tray"); the modal is non-dismissable (no close X, no backdrop click), exposes no Esc handler, and uses the preferences store to persist the choice and call `requestExit()` when "Exit" is chosen
- [x] 4.4 Mount the modal once in `frontend/src/App.vue` (or `frontend/src/layouts/DefaultLayout.vue` if `App.vue` is just a `<RouterView>`) so it is reachable from every route
- [x] 4.5 Subscribe to the `tray:close-behavior-prompt` Wails runtime event inside the modal component (or a small composable `useCloseBehaviorPrompt`) using `EventsOn` from `@wailsjs/runtime/runtime`, opening the modal when the event fires and re-focusing it on subsequent re-emits while open
- [x] 4.6 Add a Vitest test in `frontend/src/stores/__tests__/preferences.spec.ts` mocking the Wails bindings and verifying the round-trip (`loadCloseBehavior`, `setCloseBehavior`, error handling)

## 5. Validation

- [x] 5.1 Run `go test ./...` and confirm all preferences tests pass
- [x] 5.2 Run `pnpm type-check`, `pnpm lint`, `pnpm format` in `frontend/`
- [x] 5.3 Run `pnpm run test:unit` and confirm the new preferences-store test plus existing tests pass
- [x] 5.4 Run `wails dev` on Windows and manually verify the system-tray spec scenarios end-to-end:
  - Tray icon appears on launch and disappears on exit
  - Left-click toggles window visibility
  - Right-click menu shows the three items with the correct check state
  - First close with `unset` preference shows the modal and persists the chosen value; subsequent closes follow the persisted value without prompting
  - Tray "Quit RedShell" exits regardless of preference
  - Toggling the menu item from inside the tray flips the persisted preference; opening the menu again shows the new check state
- [x] 5.5 Run `wails build` on Windows to confirm a release binary boots with the same behavior; if a non-Windows machine is available, also run `wails build` to confirm the stubs compile

## 6. Documentation

- [x] 6.1 Update `CLAUDE.md` Project Overview with one sentence noting the Windows tray icon and the close-behavior preference
- [x] 6.2 Add a row (or sibling note) describing `~/.redshell/preferences.json` near the existing `Agent-specific paths` table in `CLAUDE.md`
- [x] 6.3 Run `openspec validate windows-tray-icon --strict` and resolve any validation issues
- [x] 6.4 Run `openspec status --change "windows-tray-icon"` and confirm all artifacts are `done`
