## 1. Store surface

- [x] 1.1 Open `frontend/src/stores/sessionHistory.ts` and remove the `displayTitle` computed (currently lines 32-35 in the existing file); update the returned object so `displayTitle` is no longer exported
- [x] 1.2 Add a `currentDisplayName` computed: `computed(() => currentMeta.value?.displayName ?? '')`; export it from the store alongside `currentSessionID`
- [x] 1.3 Update `frontend/src/stores/__tests__/sessionHistory.test.ts` so any existing assertions on `displayTitle` migrate to either `currentSessionID` or `currentDisplayName`; add explicit cases asserting `currentDisplayName === ''` when the mocked `SessionMeta` returns an empty `displayName`

## 2. AppCopyButton primitive

- [x] 2.1 Create `frontend/src/components/ui/AppCopyButton.vue` (Composition API, `<script setup lang="ts">`); props: `text: string`, optional `size?: DaisySize` (default `'sm'`), optional `tooltip?: string` (default `'Copy'`)
- [x] 2.2 Inside the component, import `ClipboardSetText` from `@wailsjs/runtime/runtime` and `useToast` from `@/composables/useToast`; render an `AppButton` (variant `ghost`, size from prop, square shape via `class="btn-circle"` is acceptable inside the primitive) with an icon slot
- [x] 2.3 On click: call `await ClipboardSetText(props.text)`; on success swap the icon to a check mark for ~1200 ms and dispatch a "Copied" toast; on failure (rejection or `false` return) dispatch a "Failed to copy" toast and do not swap the icon
- [x] 2.4 Use the existing icon set in `AppIcon.vue` if it has `copy` and `check` glyphs; otherwise add a `copy` and `check` entry to `frontend/src/components/ui/AppIcon.vue`
- [x] 2.5 Run the daisyUI leak grep from `CLAUDE.md` against `frontend/src/views/`, `frontend/src/components/sessions/`, and the other listed folders; `frontend/src/components/ui/` is excluded from the grep, so the new primitive does not violate the boundary

## 3. PageContainer slot

- [x] 3.1 Open `frontend/src/layouts/PageContainer.vue`; keep the `titleSuffix?: string` prop and its existing behaviour intact for back-compat
- [x] 3.2 Add a named slot `title-suffix` to the existing `<span v-if="props.titleSuffix" class="text-xl font-normal opacity-60 break-words leading-tight">` region: when the slot has content, render the slot inside (or in place of) that span; when the slot is empty, fall back to the prop-driven text render (preserve the `v-if="props.titleSuffix"` guard for the prop path)
- [x] 3.3 Verify (read-only) that `BrowsePluginsView.vue`, `InstalledView.vue`, and any other current callers continue to compile by checking they still pass the `titleSuffix` prop; do not edit those views *(verified — only `SessionHistoryView.vue` passes `:title-suffix`; all other views just pass `title=`. The string-prop fallback path remains unchanged.)*

## 4. SessionHistoryView header

- [x] 4.1 Open `frontend/src/views/SessionHistoryView.vue`; remove the `titleSuffix` computed at line 70
- [x] 4.2 Replace the existing `<PageContainer ... :title-suffix="titleSuffix" ...>` opening tag with `<PageContainer title="Session History" max-width="max-w-7xl" fill>` (drop the prop)
- [x] 4.3 Inside the `PageContainer`, add a `<template #title-suffix>` block that renders only when `store.currentSessionID` is truthy; inside it render: a row containing the full `store.currentSessionID` (allowed to wrap via existing `break-words`) and an `<AppCopyButton :text="store.currentSessionID" tooltip="Copy session id" />`; below that row, render the secondary display-name line
- [x] 4.4 Compute `showDisplayName` in the view's `<script setup>`: `const showDisplayName = computed(() => { const id = store.currentSessionID; const name = store.currentDisplayName; return name !== '' && name !== id && !id.startsWith(name); });`
- [x] 4.5 In the slot template, gate the secondary line with `v-if="showDisplayName"`; render it as a `<span>` with smaller / lower-emphasis styling than the session id (e.g. `text-base opacity-50 break-words leading-tight`)
- [x] 4.6 Confirm by reading the final `<script setup>` that no `displayTitle` reference remains and that the imports list still type-checks

## 5. Tests

- [x] 5.1 Add a focused Vitest spec at `frontend/src/components/ui/__tests__/AppCopyButton.spec.ts` that mocks `@wailsjs/runtime/runtime` to provide a fake `ClipboardSetText` and asserts: success path resolves and triggers the toast helper; failure path triggers the failure toast and leaves the icon unchanged *(file extension `.spec.ts` chosen to match the existing `components/ui/__tests__/` convention)*
- [x] 5.2 Add a Vitest spec at `frontend/src/views/__tests__/SessionHistoryView.spec.ts` that mounts `SessionHistoryView` with a mocked `useSessionHistoryStore` and asserts:
  - When `currentSessionID === ''`: the header renders only "Session History" — no session-id text, no copy button, no display-name line
  - When `currentSessionID === 'abc-123-...'` and `currentDisplayName === 'abc-123-..'` (i.e. the short-id fallback shape — strict prefix of the session id): the session id and copy button render but the display-name line does NOT render
  - When `currentSessionID === 'abc-123-...'` and `currentDisplayName === 'Refactor auth flow'`: both the session id, the copy button, and the display-name line render
  - Plus: empty `displayName` and `displayName === sessionID` exact-equality cases
- [x] 5.3 Run `cd frontend && pnpm run test:unit -- --run` and confirm green *(10 files / 58 tests passed)*

## 6. Verification

- [x] 6.1 Run `cd frontend && pnpm type-check` and confirm clean *(vue-tsc --build returned with no errors)*
- [x] 6.2 Run `cd frontend && pnpm lint` and confirm clean *(0 warnings, 0 errors after adding `vi.fn<>` type parameters)*
- [x] 6.3 Run `cd frontend && pnpm format` and confirm no formatting drift on the touched files *(prettier reformatted one wrap in `SessionHistoryView.spec.ts`; tests still green)*
- [x] 6.4 Run the daisyUI leak grep from `CLAUDE.md`:
  ```
  rg "class=\"[^\"]*\b(btn|card|alert|modal|tabs|tab|input|checkbox|collapse|badge|loading|toast|dropdown|avatar|select|textarea|radio|toggle|range|footer)\b" \
    frontend/src/views \
    frontend/src/components/settings \
    frontend/src/components/plugin \
    frontend/src/components/marketplace \
    frontend/src/components/agent \
    frontend/src/components/hooks
  ```
  Confirm zero matches. *(Zero matches; new `AppCopyButton` lives in `components/ui/` which is excluded from the grep.)*
- [ ] 6.5 Run `wails dev` and manually verify, with at least one configured agent and a session selected:
  - The header shows "Session History" with the full session id on a second line
  - A copy button appears next to the session id; clicking it copies the id (paste-test in another window) and shows the "Copied" feedback
  - When the underlying session has a real rich title (e.g. a Claude session with a `custom-title` event or a Copilot session with `workspace.yaml.summary` populated), the display name renders below the session id
  - When the underlying session has no rich title (the 8-char short-id fallback), the display-name line is suppressed
  - With no session selected, the header reads only "Session History"
  *(Manual smoke verification — requires interactive `wails dev`. Vitest specs cover the four shape cases; unchecked here pending live confirmation by user.)*
- [x] 6.6 Run `openspec validate session-history-header-show-session-id --strict` and confirm clean *(strict-validated from repo root: "Change 'session-history-header-show-session-id' is valid")*

## 7. Iteration: move session-info into main content + UUID-only id

User feedback after the first pass: the page header strip is `h-20 overflow-hidden` and clipped the session id + display name; also the rendered id was the full Claude path `<encoded-cwd>/<uuid>` rather than just the UUID.

- [x] 7.1 Revert the `title-suffix` named slot on `frontend/src/layouts/PageContainer.vue`; the layout primitive returns to its pre-change shape (string `titleSuffix` prop only) since no caller now uses the slot
- [x] 7.2 Add a `basename(id)` helper and a `shortSessionID` computed in `frontend/src/views/SessionHistoryView.vue` that returns the substring after the final `/` (or the full id when no `/` is present), matching the existing helper in `SessionListItem.vue:22-24`
- [x] 7.3 Update `showDisplayName` to compare the display name against `shortSessionID` (not the full `currentSessionID`) so the strict-prefix short-id fallback suppression works for path-prefixed Claude ids
- [x] 7.4 Remove the `<template #title-suffix>` slot block from `SessionHistoryView.vue`; render a session-info bar inside the content `<template v-else>` block, above the tab control, gated by `v-if="store.currentSessionID"`; the bar shows `shortSessionID` + `<AppCopyButton :text="shortSessionID" tooltip="Copy session id">` and the optional display-name line
- [x] 7.5 Tag the bar with `data-testid="session-info-bar"` and the secondary line with `data-testid="session-display-name"` to give the spec stable selectors
- [x] 7.6 Rewrite `frontend/src/views/__tests__/SessionHistoryView.spec.ts` to assert against the new bar location and basename-id behaviour, including: bar absent when no session selected; bar shows UUID only for path-prefixed Claude id; copy button receives the same UUID-only string; display-name shown for rich titles; display-name hidden for empty / equal / strict-prefix shapes; Copilot bare-UUID shape unchanged
- [x] 7.7 Update the `Page header reflects the selected session` requirement in `openspec/changes/session-history-header-show-session-id/specs/session-history-viewer/spec.md` so its scenarios describe the content-area bar, the basename-id rule, and the path-prefix special case
- [x] 7.8 Re-run `pnpm type-check`, `pnpm lint`, `pnpm format`, `pnpm test:unit -- --run`, the daisyUI leak grep, and `openspec validate ... --strict`; confirm green

## 8. Iteration: pin session-info bar to a fixed height

User feedback after the second pass: clicking different sessions made the panes below jump because the bar grew/shrank as the display-name line appeared and disappeared.

- [x] 8.1 Add `h-14 shrink-0` to the session-info bar wrapper in `SessionHistoryView.vue` so the bar always reserves the same vertical space (~56 px), enough for the session-id row plus the optional display-name line, regardless of whether the second line renders
- [x] 8.2 Add a Vitest case in `SessionHistoryView.spec.ts` that toggles `currentMeta.displayName` between a rich title and an empty string and asserts the bar's outer class string is identical in both states (and contains `h-14` and `shrink-0`)
- [x] 8.3 Add a `Bar height is stable across display-name visibility changes` scenario to the modified `Page header reflects the selected session` requirement so the no-jump property is captured in spec
- [x] 8.4 Re-run `pnpm test:unit -- --run` and `openspec validate ... --strict`; confirm green

## 9. Iteration: Resume button launches `<agent> --resume` in pwsh

User feedback after the third pass: surface a button next to the Copy button that opens a terminal (default `pwsh`) and runs the agent's resume command — `claude --resume <id>` or `copilot --resume <id>`.

- [x] 9.1 Create `internal/sessionhistory/resume.go` with `Service.ResumeSession(agentID, sessionID) error`; extract the basename from `sessionID`, strictly validate against `^[A-Za-z0-9_-]+$`, look up the cli binary from a closed `agentCLI` map, then call a package-level `launchResumeTerminal(cli, sessionID)` function variable so tests can stub the launcher
- [x] 9.2 Add `internal/sessionhistory/terminal_windows.go` (build tag `//go:build windows`) implementing `defaultLaunchResumeTerminal` as `exec.Command("pwsh", "-NoExit", "-Command", "& <cli> --resume <id>")` with `SysProcAttr{CreationFlags: 0x10}` (`CREATE_NEW_CONSOLE`)
- [x] 9.3 Add `internal/sessionhistory/terminal_other.go` (build tag `//go:build !windows`) returning a typed `ErrTerminalUnsupported`
- [x] 9.4 Add `internal/sessionhistory/resume_test.go` covering: Claude path-prefix is stripped to basename; Copilot bare UUID is passed through; unknown agent rejected with `ErrUnknownAgent`; empty session id rejected with `ErrInvalidSessionID`; eight injection-shape session ids are rejected (semicolons, backticks, env-var expansion, pipes, quotes, spaces, mixed)
- [x] 9.5 Add `(a *SessionHistoryApp) ResumeSession(agentID, sessionID string) error` to `app/sessionhistory.go`
- [x] 9.6 Hand-stage the Wails binding additions in `frontend/wailsjs/go/app/SessionHistoryApp.d.ts` and `.js` (signature `ResumeSession(arg1: string, arg2: string): Promise<void>`); match the existing comment header so a future `wails dev` regen produces zero diff
- [x] 9.7 Add a `play` icon glyph (`mdi:play`) to `frontend/src/components/ui/AppIcon.vue` (both the `IconName` union and the `mdiByName` map)
- [x] 9.8 Create `frontend/src/components/ui/AppResumeButton.vue` primitive: props `agentId`, `sessionId`, optional `size`, optional `tooltip`; on click awaits `ResumeSession(agentId, sessionId)` and dispatches a success or failure toast via `useToast`; disables itself while a launch is in flight and when `sessionId` is empty; renders a round ghost button (`btn btn-ghost btn-circle`) inside the primitive (`components/ui/` is excluded from the daisyUI leak grep)
- [x] 9.9 Add a Vitest spec `frontend/src/components/ui/__tests__/AppResumeButton.spec.ts` covering: click forwards args; success toast; error toast preserves the underlying error message; in-flight disabled; empty-`sessionId` disabled; tooltip → `title` and `aria-label`
- [x] 9.10 Wire `<AppResumeButton :agent-id="store.currentAgent" :session-id="shortSessionID" tooltip="Resume session in terminal" size="sm" />` into `SessionHistoryView.vue`, immediately to the right of `<AppCopyButton>` inside the same flex row
- [x] 9.11 Extend `SessionHistoryView.spec.ts` to assert: when a session is selected, the resume button renders, its `agentId` prop equals `store.currentAgent`, and its `sessionId` prop equals the UUID-only `shortSessionID`; mock `ResumeSession` in the existing `vi.mock('@wailsjs/go/app/SessionHistoryApp', ...)` block
- [x] 9.12 Add an ADDED Requirement `Session-info bar exposes a Resume control` to the spec delta with scenarios for: Resume control rendering position, per-agent command construction, `pwsh -NoExit -Command "& ..."` with `CREATE_NEW_CONSOLE` on Windows, strict regex validation, unknown-agent rejection, unsupported-platform error, and frontend toast feedback
- [x] 9.13 Update `proposal.md` to reflect the new resume affordance (What Changes), the modified capability scope, and the backend/binding deltas (Impact)
- [x] 9.14 Run `go test ./internal/sessionhistory/...`, `go vet ./internal/sessionhistory/... ./app/...`, `pnpm type-check`, `pnpm lint`, `pnpm format`, `pnpm test:unit -- --run`, the daisyUI leak grep, and `openspec validate ... --strict`; confirm green

## 10. Iteration: start the resumed terminal in the session's project cwd

User feedback after the fourth pass: the resumed terminal should `cd` into the session's project directory (the `SessionMeta.cwd`), not the session-file directory.

- [x] 10.1 Extend `Service.ResumeSession` to accept `(agentID, sessionID, cwd string)`; add a `sanitizeCwd(cwd string) string` helper that returns the cleaned absolute path when it exists and is a directory, or the empty string otherwise (defense in depth — frontend's cached cwd may be stale if the project folder was moved)
- [x] 10.2 Extend `defaultLaunchResumeTerminal(cli, sessionID, cwd string) error` in `terminal_windows.go` to set `cmd.Dir = cwd` when non-empty so the spawned pwsh inherits that as its working directory via `CreateProcessW.lpCurrentDirectory`; do NOT interpolate the cwd into the `-Command` string (avoids any shell quoting concerns through paths with spaces or apostrophes)
- [x] 10.3 Update the `terminal_other.go` stub signature to match `(cli, sessionID, cwd string) error`
- [x] 10.4 Extend `app/sessionhistory.go` `ResumeSession` to accept and forward the cwd parameter
- [x] 10.5 Update `frontend/wailsjs/go/app/SessionHistoryApp.{d.ts,js}` to expose the 3-arg form
- [x] 10.6 Add a `cwd?: string` prop (default `''`) to `AppResumeButton.vue` and pass it through to the Wails call
- [x] 10.7 In `SessionHistoryView.vue`, bind `:cwd="store.currentMeta?.cwd ?? ''"` on the `<AppResumeButton>` so the cached cwd from `SessionMeta` is forwarded
- [x] 10.8 Extend `resume_test.go`: existing-dir cwd is forwarded as the cleaned absolute path; missing-path cwd falls back to empty; relative-path cwd falls back to empty; file-instead-of-directory cwd falls back to empty; bare-Copilot test asserts empty-cwd passthrough
- [x] 10.9 Extend `AppResumeButton.spec.ts` to assert the cwd is forwarded as the third arg, and that omitting the prop sends an empty string
- [x] 10.10 Extend `SessionHistoryView.spec.ts` to assert the resume button receives `currentMeta.cwd` via its `cwd` prop
- [x] 10.11 Add `Spawned terminal starts in the session's project cwd` and `Cwd falls back gracefully when invalid` scenarios to the `Session-info bar exposes a Resume control` requirement
- [x] 10.12 Update `proposal.md` `What Changes` to document the cwd behavior, the `cmd.Dir` choice, and the graceful fallback
- [x] 10.13 Re-run all checks (`go test ./internal/sessionhistory/...`, `go vet`, `pnpm type-check`, `pnpm lint`, `pnpm format`, `pnpm test:unit -- --run`, leak grep, strict validate)

## 11. Iteration: keep the resumed terminal open after the agent CLI exits

User feedback after the fifth pass: the spawned terminal auto-closes immediately, even though `pwsh -NoExit` was being passed. Switching to the `cmd /c start "" pwsh ...` idiom (which is the canonical Windows path for "open a new detached console window that stays open") keeps `-NoExit` reliable.

- [x] 11.1 Rewrite `internal/sessionhistory/terminal_windows.go` to spawn `cmd.exe /c start "" pwsh -NoExit -NoProfile -Command "<cli> --resume <id>"` instead of invoking `pwsh` directly with `CREATE_NEW_CONSOLE`; the `start` builtin opens a fresh detached console for pwsh, and `-NoExit -NoProfile` together survive both clean exits and a hostile user profile
- [x] 11.2 Drop the `&` call operator from the inner command; bare `<cli> --resume <id>` is parsed by pwsh as an external-command invocation and is simpler (no behavioural difference for the supported agents)
- [x] 11.3 Replace the `CREATE_NEW_CONSOLE` flag on `cmd.SysProcAttr` with `CREATE_NO_WINDOW` plus `HideWindow: true` on the cmd.exe spawn so the transient cmd.exe step does NOT flash a console; the user-visible window is only the pwsh window opened by `start`
- [x] 11.4 Keep `cmd.Dir = cwd` on the cmd.exe spawn — pwsh inherits cmd.exe's working directory, so the project-cwd-on-resume property from §10 is preserved
- [x] 11.5 Replace the `Default terminal is pwsh on Windows` scenario in the spec delta with the more specific `start "" pwsh -NoExit -NoProfile -Command` invocation, and add two new scenarios: `Resumed terminal stays open until the user closes it` (captures the persistent-shell guarantee using `-NoExit -NoProfile`) and `Spawning launcher does not flash a visible console` (captures the `CREATE_NO_WINDOW` + `HideWindow` choice)
- [x] 11.6 Update `proposal.md` `What Changes` to reflect the new launcher recipe
- [x] 11.7 Re-run `go test ./internal/sessionhistory/...` (the stub-based tests are unaffected by the platform-code change), `go vet`, and `openspec validate ... --strict`

## 12. Iteration: swap the resume icon from play-triangle to terminal-style

User feedback after the sixth pass: the `mdi:play` triangle is misleading for "open in terminal" — switch to a terminal-style glyph similar to the Windows Terminal app icon.

- [x] 12.1 Replace the `play` entry in `frontend/src/components/ui/AppIcon.vue` with `terminal` mapped to `mdi:console-line` (outline variant to match the existing icon-set aesthetic of `cog-outline`, `folder-outline`, etc.); the `>_` console glyph is universally recognised as "open a terminal"
- [x] 12.2 Update `frontend/src/components/ui/AppResumeButton.vue` to render `<AppIcon name="terminal" />` instead of `name="play"`
- [x] 12.3 Re-run `pnpm type-check`, `pnpm test:unit -- --run`, and the daisyUI leak grep; confirm green

## 13. Iteration: error out when the resumed session's project cwd is missing

User feedback after the seventh pass: the silent-fallback behaviour from §10 is wrong. If the recorded project directory does not exist, the user should see an error and the terminal should NOT open — not be dropped into an unrelated default cwd.

- [x] 13.1 Add a typed sentinel `ErrProjectCwdMissing = errors.New("project directory does not exist")` to `internal/sessionhistory/resume.go`
- [x] 13.2 Replace `sanitizeCwd(cwd) string` with `resolveCwd(cwd) (string, error)`: empty input → `("", nil)`; non-empty + absolute + existing directory → `(cleaned, nil)`; non-empty + invalid (not absolute, missing, or not a directory) → `("", fmt.Errorf("%w: %s", ErrProjectCwdMissing, cwd))`. The wrapped path lets the frontend toast tell the user which directory is missing
- [x] 13.3 Update `Service.ResumeSession` to short-circuit with the resolveCwd error before invoking the launcher
- [x] 13.4 Update `resume_test.go`: rename the three "FallsBackWhen…" cases to `ErrorsWhenCwdMissing`, `ErrorsWhenCwdIsRelative`, `ErrorsWhenCwdIsFile`; assert `errors.Is(err, ErrProjectCwdMissing)` and that the launcher stub is NOT called; add `TestResumeSession_LaunchesWhenCwdIsEmpty` to lock the empty-input passthrough; verify the missing-path test additionally asserts the offending path appears in the error message
- [x] 13.5 Replace the `Cwd falls back gracefully when invalid` scenario in the spec delta with two new scenarios: `Empty cwd inherits the spawning process's cwd` and `Non-existent project cwd aborts the launch with an error`
- [x] 13.6 Update `proposal.md` `What Changes` to reflect the new error-out behaviour and the typed `ErrProjectCwdMissing` error
- [x] 13.7 Re-run `go test ./internal/sessionhistory/...`, `go vet`, `pnpm test:unit -- --run` (no frontend code changes; `AppResumeButton`'s existing failure-toast path already surfaces the new error message), and `openspec validate ... --strict`; confirm green
