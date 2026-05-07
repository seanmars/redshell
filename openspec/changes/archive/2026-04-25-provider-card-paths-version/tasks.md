## 1. Backend — `osopen` capability

- [x] 1.1 Create `internal/osopen/osopen.go` with an exported `OpenPath(path string) error`. Expand a leading `~` or `~/` using `os.UserHomeDir()`, then `os.Stat` the result and return an error wrapping the missing path on failure.
- [x] 1.2 Add a `runtime.GOOS` switch inside `OpenPath` that dispatches to `cmd /c start "" <path>` on `windows`, `open <path>` on `darwin`, and `xdg-open <path>` elsewhere; use `exec.Command(...).Start()` so the call is fire-and-forget.
- [x] 1.3 Add `internal/osopen/osopen_test.go` with at least: a tilde-expansion unit test (replace the dispatcher with a fake recorder) and a missing-path error test.
- [x] 1.4 Create `app/system.go` with a `SystemApp` struct that stores `context.Context` (set in `Startup`) and exposes `OpenPath(path string) error` delegating to `internal/osopen`.
- [x] 1.5 Construct `SystemApp` in `main.go`, append it to `options.App.Bind`, and wire `Startup` like the other `app.*App` wrappers.

## 2. Backend — provider version detection

- [x] 2.1 In `internal/provider/service.go`, drop the `CommandsDir` and `SkillsDir` fields from `Provider` and add `Version string` and `SettingsFile string`.
- [x] 2.2 Introduce `type execRunner func(ctx context.Context, name string, args ...string) ([]byte, error)` and store it on `*Service`. Provide `NewService()` (production: `exec.CommandContext(...).CombinedOutput`) and a `NewServiceWithRunner(execRunner)` test constructor.
- [x] 2.3 Add a private `probeVersion(ctx context.Context, bin string) string` that runs `<bin> --version` under a 2-second `context.WithTimeout`, captures combined stdout+stderr, runs the regex `\b(\d+\.\d+\.\d+)\b`, and returns the first match (empty string on any failure).
- [x] 2.4 Refactor `ListProviders` to: build the static slice with `Version: ""` first, fan out one goroutine per provider to call `probeVersion`, wait on a `sync.WaitGroup`, then merge the version results back. Set `SettingsFile` to `~/.claude/settings.json` and `~/.copilot/config.json` respectively.
- [x] 2.5 Add `internal/provider/service_test.go` cases: Claude-format string, Copilot-format string, empty output, non-zero exit, and timeout — each asserting the returned `Version` value.

## 3. Wails binding regeneration

- [x] 3.1 Run `wails generate module` from the repo root and confirm `frontend/wailsjs/go/app/SystemApp.d.ts`, `frontend/wailsjs/go/app/SystemApp.js`, and updated `frontend/wailsjs/go/models.ts` (no more `commandsDir`/`skillsDir`, plus new `version`/`settingsFile`) appear in the diff.
- [x] 3.2 Stage the regenerated `frontend/wailsjs/go/**` files in the same commit as the Go struct change so the binding contract stays consistent on `main`.

## 4. Frontend — ProviderCard rewrite

- [x] 4.1 In `frontend/src/components/provider/ProviderCard.vue`, replace the body so it renders only two rows: Directory (label + clickable `~/.<dir>` value) and Configuration (label + clickable `~/.<file>` value). Drop the Commands and Skills rows entirely.
- [x] 4.2 Make each row a semantic `<button>` styled with Tailwind utilities (no daisyUI `btn` class — keeps the row layout instead of a pill), wired to a click handler that calls the new `OpenPath` Wails binding from `@wailsjs/go/app/SystemApp`. (Deviates from initial AppButton plan; semantic `<button>` avoids daisyUI component-class leak per CLAUDE.md leak check while preserving info-row visual layout.)
- [x] 4.3 Replace the existing badge logic: when `provider.version` is non-empty, render `<AppBadge variant="info" size="sm">{{ provider.version }}</AppBadge>`; otherwise render an `AppIcon` `warning` glyph inside a span with `aria-label="Not installed"`. Extended `AppIcon` `IconName` to add `warning`, `folder`, and `file` glyphs in the same change.
- [x] 4.4 Keep the existing "Install … to enable this provider." hint, with visibility keyed off `!provider.configured` (unchanged) — the hint is independent of the version probe.
- [x] 4.5 Wrap the click-handler call in a try/catch and surface failures via `useToast()` with the path that failed.

## 5. Frontend — store and types

- [x] 5.1 Confirm `frontend/src/stores/provider.ts` continues to compile after the model regeneration; remove any `commandsDir`/`skillsDir` references if they exist anywhere else under `frontend/src/`. (`vue-tsc --build` passed; `rg commandsDir|skillsDir frontend/src` returned zero matches.)
- [x] 5.2 Run the mechanical leak check from `CLAUDE.md` against `frontend/src/components/provider/`. Expected: zero matches. (Confirmed: zero matches.)

## 6. Verification

- [x] 6.1 Run `go fmt ./...` and `go vet ./...`. (Both clean; `gofmt` only trimmed a trailing blank line in `app/provider.go`.)
- [x] 6.2 Run `go test ./...` and confirm new `internal/provider` and `internal/osopen` tests pass. (`internal/osopen` 3/3 and `internal/provider` 7/7 pass; `internal/marketplace` failures are pre-existing — verified by `git stash` reproduction on `main` before any changes — and unrelated to this change.)
- [x] 6.3 From `frontend/`, run `pnpm format`, `pnpm lint`, and `pnpm run build` (which includes `vue-tsc --build`). (All clean; oxlint 0 warnings, eslint clean, type-check + vite build green; vitest 15/15.)
- [x] 6.4 Run `wails dev` and verify on the Providers tab: (a) the badge shows `2.1.119`-style versions when CLIs are installed, (b) the badge shows the warning icon when a CLI is removed from PATH, (c) clicking Directory opens the folder in Explorer/Finder, (d) clicking Configuration opens the JSON file in the default editor, (e) clicking a path that does not exist surfaces a toast and does not crash. (**Pending manual verification by user** — `wails dev` opens an interactive desktop window which can't be driven from this non-interactive shell session.)
