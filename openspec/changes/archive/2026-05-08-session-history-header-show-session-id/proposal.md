## Why

The Session History page header currently shows a single string `Session History — <displayName>`, where `displayName` falls back to the bare session id when no rich title is resolvable. This conflates two distinct pieces of information — the canonical, copy-friendly session id and the human-readable display name — into one slot. Users who want to grep `~/.claude/projects/` or `~/.copilot/session-state/` for the active session, share a session id with a teammate, or open the underlying jsonl file have to either retype a UUID by eye or click into the disk path themselves. Splitting the two and adding a one-click copy affordance turns the header into the primary handle for "this exact session on disk".

## What Changes

- Replace the current `titleSuffix` (which holds either the resolved `displayName` or, when absent, the session id) with the **full session id** as the primary suffix line on `SessionHistoryView.vue`.
- Add a copy-to-clipboard control immediately to the right of the rendered session id; clicking it writes the session id to the system clipboard and gives transient visual feedback (icon swap and/or toast).
- When the backend resolves a non-empty rich `displayName` that is **not** a fallback to the session id, render that display name on a second, smaller line directly below the session id. When `displayName` is empty or equal to the session id (the existing fallback case), the second line is omitted entirely.
- Keep the primary `<h1>` text "Session History" unchanged, and keep the no-selection state (no suffix, no copy button, no display-name line) unchanged.
- Extend `frontend/src/layouts/PageContainer.vue` so the suffix area can host structured content (session id + copy button + optional secondary line) instead of a plain string. Existing call sites that pass `titleSuffix` as a string SHALL keep working without code changes.
- Refine the `sessionHistory` Pinia store so the view can distinguish "real display name resolved" from "fell back to session id" — the current `displayTitle` getter collapses both into one string and SHALL be replaced (or augmented) by separate `sessionID` and `displayName` exposures.
- Add a Resume control immediately to the right of the Copy control in the session-info bar that opens a new `pwsh` window and runs the agent's `--resume <session-id>` command (`claude --resume ...` or `copilot --resume ...`) in the **session's project directory** (the `SessionMeta.cwd`, not the session-file directory). Backend exposes a new `SessionHistoryApp.ResumeSession(agentID, sessionID, cwd)` Wails method; session id is strictly validated against `^[A-Za-z0-9_-]+$` after basename extraction to prevent shell injection; cwd is set on the spawned process via Go's `cmd.Dir` (passed to Win32 `CreateProcessW`) rather than interpolated into the shell command, eliminating any quoting concerns through paths. An empty cwd is allowed (launches in the default cwd); a non-empty cwd that fails validation (not absolute, missing, or a non-directory) returns a typed `ErrProjectCwdMissing` error wrapped with the offending path so the frontend can show it in a toast — the terminal is NOT opened in this case. The launcher is `cmd /c start "" pwsh -NoExit -NoProfile -Command "<inner>"` so the pwsh window stays open after the agent CLI exits (the user closes it explicitly) and is fully detached from RedShell's process tree; the transient cmd.exe step is suppressed via `CREATE_NO_WINDOW`.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `session-history-viewer`: The "Page header reflects the selected session" requirement changes shape — the primary suffix becomes the full session id with a copy control, and the rich display name moves to an optional secondary line that is suppressed when it would duplicate the session id. A new "Session-info bar exposes a Resume control" requirement is added to the same capability for the resume affordance.

## Impact

- **Frontend (modified)**:
  - `frontend/src/views/SessionHistoryView.vue` — replace the `titleSuffix` computed with a header-content structure; use the new `PageContainer` suffix slot/contract.
  - `frontend/src/layouts/PageContainer.vue` — add a structured suffix slot (named slot `title-suffix` and/or richer prop) while keeping the string `titleSuffix` prop backward-compatible for `BrowsePluginsView`, `InstalledView`, etc.
  - `frontend/src/stores/sessionHistory.ts` — expose the session id and the resolved display name separately, so the view can decide whether the display-name line should render.
  - `frontend/src/components/ui/` — likely a new `AppCopyButton` (or equivalent) primitive so the daisyUI `btn`/`btn-ghost btn-circle` boundary stated in `CLAUDE.md` is preserved; or a tightly scoped use of the existing `AppButton` if a generic copy primitive is overkill.
  - `frontend/src/composables/useToast.ts` (existing) — reused for the "Copied" feedback; no changes expected.
- **Frontend (unchanged)**: `SessionList`, `SessionListItem`, `SessionEventList`, `SessionEventItem`, router, and all other views.
- **Backend**:
  - `internal/sessionhistory/resume.go` — new `Service.ResumeSession(agentID, sessionID)` method with strict basename validation.
  - `internal/sessionhistory/terminal_windows.go` — `pwsh -NoExit -Command "& <cli> --resume <id>"` spawn with `CREATE_NEW_CONSOLE`, build-tagged for Windows.
  - `internal/sessionhistory/terminal_other.go` — non-Windows stub returning `ErrTerminalUnsupported`.
  - `app/sessionhistory.go` — new `ResumeSession` Wails method.
  - Other backend code (`SessionMeta`, listings, paginated reads) is unchanged.
- **Tests**:
  - `frontend/src/stores/__tests__/sessionHistory.test.ts` — adjust to assert separate `sessionID` / `displayName` exposure if the store surface changes.
  - Add a focused Vitest spec for `SessionHistoryView` (or for the new `AppCopyButton` primitive) covering: copy-button writes to `navigator.clipboard`, secondary line is hidden when `displayName === sessionID`, secondary line renders when they differ.
- **Docs / spec**: update `openspec/specs/session-history-viewer/spec.md` via the delta in this change; no other docs changed.
- **Wails bindings**: `frontend/wailsjs/go/app/SessionHistoryApp.{d.ts,js}` gain a `ResumeSession(arg1, arg2)` symbol; hand-staged in the existing generated format so the next `wails dev` regeneration leaves no diff.
- **No new dependencies** added to either `go.mod` or `frontend/package.json`.
