## Context

RedShell is a Wails v2 desktop app whose `main.go` constructs domain services (`agent`, `marketplace`, `plugin`, `sessionhistory`, `hooks`) plus thin `app/*` wrappers and binds them via `options.App.Bind`. There is currently no `OnBeforeClose` hook, no tray integration, and no `app/system.go`-managed window lifecycle beyond `Startup`. Closing the main window terminates the process.

The existing settings storage pattern is `agent.SettingsService` writing `~/.redshell/settings.json`. That file holds agent-setup state; mixing shell-level UX preferences (tray, close behavior, future appearance toggles) into it would conflate concerns. The codebase already shows the pattern of spinning up small focused services (e.g. `internal/sessionhistory`, `internal/hooks`) rather than expanding existing ones.

Wails v2 does not ship a tray API. Wails v3 does, but migrating the project off v2 is a larger change unrelated to this proposal. The user has scoped the request to Windows ("Windows tray icon"); macOS / Linux support is desirable later but not required now.

The frontend already has the `AppModal` primitive, `useConfirm` for imperative confirms, and the `EventsOn` runtime binding pattern for backend → frontend events (used today by `plugin:install-log`). The first-run prompt can lean on these instead of inventing new infrastructure.

## Goals / Non-Goals

**Goals:**
- Keep the main window alive in the system tray when the user prefers it, so launching RedShell once per session is enough.
- Make the close-button choice explicit (one-time prompt) and reversible (tray menu toggle) without burying it in settings.
- Persist the choice durably across restarts in a file separate from agent config.
- Keep non-Windows builds compiling and runnable — tray and close-intercept are no-ops on those platforms.
- Keep the design.md / proposal contract: tray + preference logic is testable Go code; the UI integration is a thin Vue layer.

**Non-Goals:**
- Cross-platform parity in this change. macOS menu-bar item and Linux StatusNotifierItem support are deferred. The build-tag boundary is set up so they can be added without restructuring.
- Notifications / toasts from the tray icon (e.g. balloon tips when a plugin install completes). The tray exists only for window control + close-behavior toggle in this change.
- Rich tray menu beyond the three items described in proposal.md.
- A general-purpose preferences UI in the Settings page. The tray context menu is the primary surface for the close-behavior toggle; surfacing it elsewhere can come later.
- Migrating to Wails v3 to use its native tray API.
- Auto-start-on-login, single-instance enforcement, or background services unrelated to window control.

## Decisions

### Decision 1: Tray library — `getlantern/systray` vs. `energye/systray` vs. raw Win32

**Chosen:** `github.com/getlantern/systray`.

**Rationale:** It is the most widely used Go tray library, has a simple `OnReady` / `Run` API, supports checkable menu items (which we need for the "Close button minimizes to tray" toggle), and works on Windows without CGO once `windows`-specific build is targeted (it uses pure Go for the Win32 calls and only requires CGO on Linux through GTK). Wails v2 examples in the wild commonly pair with `getlantern/systray`.

**Alternative considered:**
- `energye/systray` (a maintained fork) — viable but smaller community and fewer integration examples. Easy to swap to later if `getlantern/systray` stalls.
- Raw Win32 via `golang.org/x/sys/windows` — full control but a lot of code (icon resource management, menu item IDs, message pump) for no advantage given our needs.
- Wails v3 native tray — out of scope as established in Non-Goals.

### Decision 2: Where the tray runs in the process lifecycle

**Chosen:** Start the tray in a dedicated goroutine launched from Wails' `OnStartup` hook, after the Wails context is captured. Stop it in `OnShutdown` by calling `systray.Quit()`. The tray's `OnReady` callback receives the Wails `context.Context` (via a closure) so it can call `runtime.WindowShow` / `runtime.WindowHide` / `runtime.Quit`.

**Rationale:** `systray.Run` blocks the calling goroutine until `Quit` is invoked; running it on the main goroutine would deadlock Wails' own event loop. Launching it from `OnStartup` ensures we have a usable Wails context to drive window visibility from menu callbacks.

**Alternative considered:** Spawn the tray before `wails.Run`. Rejected because `runtime.Window*` requires a `context.Context` Wails injects only after startup; we would have to plumb the context post-hoc and accept a window of time where the tray exists but cannot act on the window.

### Decision 3: Close intercept — return-to-cancel vs. WindowHide-then-allow

**Chosen:** Implement `OnBeforeClose(ctx)` that consults the persisted preference:
- `closeBehavior == "minimize-to-tray"` → call `runtime.WindowHide(ctx)`, return `true` to cancel close.
- `closeBehavior == "exit"` → return `false` to allow close.
- `closeBehavior == "unset"` → emit a `tray:close-behavior-prompt` runtime event to the frontend and return `true` to cancel. The frontend opens the modal; on user choice, the frontend writes the preference via `AppPreferencesApp.SetCloseBehavior` and then calls a new `AppPreferencesApp.ApplyPendingClose` binding which either re-issues the close (now with `exit`) via `runtime.Quit` or just leaves the window hidden (already done implicitly via `WindowHide`).

**Rationale:** Wails' `OnBeforeClose` is the documented hook for cancelling a close. Returning `true` aborts the OS-level close; we then either show the prompt or hide the window. Doing the prompt asynchronously is required because `OnBeforeClose` cannot block on a frontend round-trip — the user might take an arbitrary amount of time to click a button.

**Alternative considered:**
- Skip the hook and disable the OS close button entirely, redirecting to a custom in-app close button. Rejected — overrides native window chrome the user expects, and breaks Alt+F4 / taskbar close.
- Hide unconditionally and re-prompt every run until the user explicitly opts out. Rejected — the prompt is meant to be one-time per the user's request.

### Decision 4: Where the close-behavior preference lives

**Chosen:** New file `~/.redshell/preferences.json` managed by a new `internal/preferences` package, separate from `agent.SettingsService`'s `settings.json`.

**Rationale:** Agent-setup state and shell UX preferences have different lifetimes, audiences, and migration risks. Mixing them would force every preference change to load and re-validate the agent state, and would force `agent.SettingsService` to know about UX concerns. A focused service with its own file keeps both clean. The file is small — a single JSON object — so the storage cost is negligible.

**Alternative considered:** Add a `closeBehavior` field to `settings.json`. Rejected for the reasons above. The user already has a clear separation between `~/.claude/`, `~/.copilot/`, and `~/.redshell/`; one more file in `~/.redshell/` is consistent.

### Decision 5: First-run prompt UI — frontend modal vs. native dialog

**Chosen:** Render the prompt in the frontend using the existing `AppModal` primitive, triggered by a Wails event from the backend.

**Rationale:** The frontend already has `AppModal`, daisyUI styling, and the `EventsOn` event binding pattern. A native Win32 `MessageBox` would not match the app's visual identity, would lack daisyUI light/dark theming, and would be platform-specific code we have to maintain. Triggering via `runtime.EventsEmit("tray:close-behavior-prompt", ...)` matches the existing `plugin:install-log` precedent.

**Alternative considered:** Use a native `MessageBoxW` via `golang.org/x/sys/windows`. Rejected — visual inconsistency, more platform-specific code, harder to localize later.

### Decision 6: Tray menu state synchronization

**Chosen:** The tray's "Close button minimizes to tray" item is checkable. The tray code subscribes to a Go-side observer on the `preferences.Service` (a registered callback) and updates the menu item's checked state when the preference changes (whether via tray click or via the future settings UI). Conversely, the tray click handler calls `preferences.Service.SetCloseBehavior` so any other observers (e.g. a future settings page) can react.

**Rationale:** Single source of truth (`preferences.Service`) with a thin in-process pub/sub avoids the tray and the frontend reading divergent state. Wails' frontend-side `EventsEmit` is fine for backend → frontend; for backend → backend (preferences → tray) a Go observer is simpler and synchronous.

**Alternative considered:** Have the tray re-read the file on every menu open. Rejected — adds I/O on every right click and makes external changes (e.g. a future settings page) feel sticky until the menu is reopened.

### Decision 7: Tray icon asset

**Chosen:** Embed a 32×32 PNG (or 16×16 + 32×32 ICO) at `internal/tray/assets/tray.ico` via `//go:embed`. Use the existing `build/windows/icon.ico` as the source; downscale and commit a tray-sized variant alongside the package.

**Rationale:** `getlantern/systray` accepts an icon as `[]byte`. Embedding via `go:embed` keeps the binary self-contained and the asset path build-tag-isolated. The existing `build/windows/icon.ico` is sized for the executable; using it directly would produce a blurry tray rendering on standard DPI. A dedicated small variant is one-time work.

**Alternative considered:** Reuse `build/windows/icon.ico` as-is. Rejected for visual quality reasons described above.

## Risks / Trade-offs

- **[Risk]** `getlantern/systray` adds an indirect dependency surface and is not a Wails-blessed library. **Mitigation:** Isolate the import behind `internal/tray/tray_windows.go` with a small Go interface so swapping libraries is mechanical. Pin a known-good version in `go.mod`.
- **[Risk]** `OnBeforeClose` runs on the main UI thread; emitting an event and returning `true` is fast, but if the frontend is slow to respond to the prompt the user might press Alt+F4 again and see no effect. **Mitigation:** Track a `pendingPrompt` flag in `AppPreferencesApp`; subsequent close attempts while the modal is open simply re-emit the event so the modal regains focus. Document the modal as non-dismissable except via the two action buttons.
- **[Risk]** Hiding the window with `WindowHide` on Windows can leave the taskbar icon visible briefly depending on Wails version. **Mitigation:** Verify behavior on Windows 11 24H2 during implementation; if the taskbar entry lingers, follow up with a `runtime.WindowSetAlwaysOnTop(false)` + minimize-then-hide sequence.
- **[Risk]** The user "Quits RedShell" via tray menu while the close-behavior preference is `unset`. **Mitigation:** Tray "Quit" calls `runtime.Quit(ctx)` directly and bypasses the prompt; `OnBeforeClose` distinguishes "quit triggered programmatically" from "user clicked X" via a flag set on the `AppPreferencesApp` before invoking quit.
- **[Risk]** First-run prompt shows up even on macOS / Linux if a user there clicks close, because the close intercept runs on every platform. **Mitigation:** Gate `OnBeforeClose` behavior on the tray availability — if `tray.Available()` is false (non-Windows stub), skip the intercept and let the close happen normally. Alternatively, treat `closeBehavior == "unset"` as `"exit"` on platforms with no tray, so the preference is never written from the prompt path. Choose the second; it keeps `~/.redshell/preferences.json` valid if the user later moves their home dir to a tray-capable OS.
- **[Trade-off]** Tray menu does not yet show "Open Settings" or per-page shortcuts. Adding them is cheap later but they pull more bindings into the tray callback closure; keeping the menu to three items in v1 limits the API surface to validate.
- **[Trade-off]** Embedding a tray-sized icon adds ~5 KB to the binary, but avoids runtime icon I/O and matches the rest of the asset embedding pattern (`//go:embed all:frontend/dist`).

## Migration Plan

1. **Backend, isolated**:
   - Introduce `internal/preferences/service.go` + `service_test.go` with `GetCloseBehavior`, `SetCloseBehavior`, and an `OnChange(callback)` observer. Default `unset`.
   - Introduce `internal/tray/` with `tray_windows.go` (real) and `tray_other.go` (no-op interface satisfying stub). Define a `Manager` interface (`Start(ctx, prefs)`, `Stop()`, `Available() bool`).
   - Add `app/preferences.go` exposing `GetCloseBehavior`, `SetCloseBehavior`, `RequestExit` (used by frontend after the prompt to bypass the intercept) bound via `options.App.Bind`.
2. **Wire-up**:
   - Update `main.go` to construct `preferences.NewService()`, `tray.NewManager()`, and a new `app.NewAppPreferencesApp(prefs, tray)`.
   - Add `OnStartup` hook step that starts the tray and captures the context.
   - Add `OnBeforeClose` hook implementing Decision 3.
   - Add `OnShutdown` hook that stops the tray.
3. **Frontend**:
   - Generate Wails bindings (`pnpm` not needed; `wails dev` regenerates).
   - Add `frontend/src/stores/preferences.ts`, `frontend/src/components/system/CloseBehaviorPromptModal.vue`.
   - Mount the modal in `App.vue` and have it subscribe to `tray:close-behavior-prompt`.
4. **Asset**:
   - Add `internal/tray/assets/tray.ico` (32×32). Include a `README.md` in that folder noting the source-of-truth file (`build/windows/icon.ico`) for future regeneration.
5. **Validate**:
   - `go test ./...`, `go vet ./...`, `go fmt`.
   - `wails dev` on Windows: verify (a) close while `unset` triggers prompt; choosing "Minimize" hides window and persists `minimize-to-tray`; choosing "Exit" closes and persists `exit`. (b) Subsequent close uses persisted preference without prompting. (c) Tray icon left-click toggles window visibility. (d) Tray right-click menu shows three items, the "Close button minimizes to tray" item is checked iff preference is `minimize-to-tray`, toggling it updates the preference and the next close behaves accordingly. (e) Tray "Quit RedShell" exits regardless of preference.
   - `wails build` on macOS / Linux to confirm the no-op stubs compile.
6. **Docs**: Update `CLAUDE.md` Project Overview to mention the tray; update the agent-paths-table preamble (or add a sibling table) to list `~/.redshell/preferences.json`.

**Rollback strategy:** Revert the branch. The new `~/.redshell/preferences.json` file is harmless if left behind; nothing else reads it. No schema migrations on existing files.

## Open Questions

- Should the modal also support **Alt+F4** / **Esc** to mean "exit this once without persisting"? Inclination: no — it would create a third invisible state ("the user hit Esc, did that mean exit-just-now or cancel?"), which contradicts the goal of forcing an explicit one-time choice. Treat the modal as non-dismissable except via the two buttons.
- Do we want a "Don't ask again" checkbox separate from the action choice, or is the act of choosing implicitly the persistence event? Inclination: implicit, matching the user's request ("選擇後記住使用者的選擇").
- Should the tray menu display the current app version (read-only header item) for diagnostic purposes, or keep it strictly action-only? Defer — easy to add later.
- Should we offer a hidden CLI flag (`--reset-preferences`) to clear `preferences.json` for QA? Defer — `del %USERPROFILE%\.redshell\preferences.json` is sufficient until a real need arises.
