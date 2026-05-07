## Context

`internal/provider/service.go` returns a static `[]Provider` with `ConfigDir`, `CommandsDir`, `SkillsDir` plus a `Configured` bool derived from `os.Stat` on the home directory. `ProviderCard.vue` renders all three paths as text rows and the `Configured` field as a green/gray daisyUI badge wrapped by `AppBadge`.

The Providers tab is the first screen of the four-screen flow and acts as a launchpad — users want to verify CLI presence, jump into the directory, and edit the settings file. The current card is read-only and tells the user nothing about whether the CLI itself (not just the dotfile dir) is installed and at what version.

This change extends provider detection to invoke the CLIs themselves and adds a small cross-cutting `OpenPath` capability so the card content becomes actionable. Both pieces are needed together: the Directory and Configuration rows need a click target, and the badge needs version data.

## Goals / Non-Goals

**Goals:**

- Show installed CLI version (`x.y.z`) in the badge, with a fallback icon when not installed or unreadable.
- Make the Directory and Configuration rows open the corresponding folder/file in the OS default handler.
- Keep version probing fast enough that the Providers tab continues to render under ~250 ms in the common case (both CLIs installed).
- Keep the OpenPath capability generic so future cards (e.g. marketplace cache) can reuse it.

**Non-Goals:**

- Editing the settings file in-app (we only delegate to the OS default handler).
- Triggering installs from this card (that would belong to a separate "install CLI" flow).
- Supporting providers beyond Claude Code and GitHub Copilot in this change.
- Detection of CLIs installed under non-standard names or shims (e.g. `npx claude`).
- Live-watching the version (no polling; the value is captured at `ListProviders` time and refreshes when the user navigates back to the tab).

## Decisions

### Decision 1: Probe version inside `ListProviders`, parallelized with timeout

We invoke `claude --version` and `copilot --version` from `internal/provider/service.go` during `ListProviders`. Probes run in parallel goroutines, each with a 2-second context timeout, and short-circuit to an empty version string on any error (binary missing, non-zero exit, timeout).

**Rationale:** keeps the Wails surface unchanged (no new "GetVersion" round-trip) and matches the existing pattern where `ListProviders` is the single source of truth. Parallelization avoids serial 2 × 2-second latency in the worst case.

**Alternatives considered:**
- Separate `GetProviderVersion(id)` Wails method called per card after mount. Rejected: doubles the round-trips, complicates loading states, and adds flicker on every navigation back to the tab.
- Cache version in `~/.redshell` JSON. Rejected: stale-version risk after the user updates the CLI, and the cost of a fresh probe is already bounded.

### Decision 2: Version parsing via shared regex

Both CLIs print a semver triple somewhere in their `--version` output. We use a single regex `\b(\d+\.\d+\.\d+)\b` taking the first match in stdout (falling back to combined stdout+stderr). Output samples:

- Claude Code: `2.1.119 (Claude Code)\n` → matches `2.1.119`.
- GitHub Copilot: `GitHub Copilot CLI 1.0.34.\nRun 'copilot update' to check for updates.\n` → matches `1.0.34`.

**Rationale:** robust to format drift (Copilot already prints a multi-line message; future versions might add suffixes). Keeps a single parser for both providers.

**Alternative:** per-provider parser with strict prefix match. Rejected as brittle.

### Decision 3: Inject an `execRunner` interface for testing

Define `type execRunner func(ctx context.Context, name string, args ...string) ([]byte, error)` and store it on `*Service`. Production wires it to `exec.CommandContext(...).Output()`; tests inject a stub.

**Rationale:** lets us assert the full parsing pipeline in `internal/provider` tests without requiring `claude` and `copilot` on the CI host. Mirrors the existing pattern in `internal/plugin` of using helper constructors (`NewServiceWithCacheRoot`).

### Decision 4: New `internal/osopen` package + `app/system.go` Wails wrapper

`internal/osopen.OpenPath(path string) error`:

1. Expand a leading `~` or `~/` using `os.UserHomeDir()`.
2. Call `os.Stat` to ensure the path exists. Return a typed error if not.
3. Dispatch by `runtime.GOOS`:
   - `windows` → `exec.Command("cmd", "/c", "start", "", absPath)` (the empty quoted title prevents `start` from treating a quoted path as the title).
   - `darwin` → `exec.Command("open", absPath)`.
   - default → `exec.Command("xdg-open", absPath)`.
4. Run with `.Start()` (fire-and-forget) so the GUI thread isn't blocked by the launched application.

`app/system.go` exposes `SystemApp` with one method `OpenPath(path string) error`, bound in `main.go` alongside the other apps.

**Rationale:** decoupled from `provider` so other domains (marketplace cache dir, plugin install logs) can call it later. Tilde expansion centralizes the only path-formatting concern; the frontend can keep displaying the literal `~/.claude` string.

**Alternative:** use Wails' built-in `runtime.BrowserOpenURL` with a `file://` URL. Rejected: doesn't reliably reveal a folder in the file manager on Windows, and `start` with the path string is the documented Wails pattern for "open in OS handler".

### Decision 5: Drop `CommandsDir` and `SkillsDir` from the struct, add `Version` and `SettingsFile`

The fields disappear from the Go struct and from `frontend/wailsjs/go/models.ts` after `wails generate module`. This is a binding-shape break, but the only consumer is the project's own frontend, regenerated in the same commit.

**Rationale:** matches the rule "delete unused code completely" from the project conventions. Keeping them as legacy fields would silently ship dead data over IPC.

`SettingsFile` is `~/.claude/settings.json` for Claude and `~/.copilot/config.json` for Copilot, mirroring the existing `ProviderMarketplaceFiles` table for the installed-plugins file (they happen to coincide for Copilot — that's deliberate, the registry and settings are the same JSON file).

**`Configured` field is retained** as a coarse "the dotfile dir exists" signal. The new badge keys off `Version` (non-empty) for the version label vs. icon decision. `Configured` keeps influencing the secondary "Install … to enable this provider" hint message, so the user gets a clear cue when neither the directory nor the CLI is present.

### Decision 6: Frontend uses `AppBadge` for the version label, daisyUI icon for the fallback

When `provider.version` is non-empty: render `<AppBadge variant="info" size="sm">{{ provider.version }}</AppBadge>`.

When empty: render an inline icon-only span using the existing `AppIcon` primitive (introduced in commit `713aa10`) with a warning glyph and an `aria-label` of `Not installed`. No daisyUI `badge` class leaks into the view — both branches stay inside primitives, satisfying the design-system rule.

The Directory and Configuration rows become buttons wrapped in `AppButton` (`variant="ghost"` `size="sm"`) so they stay inside the daisyUI `btn` boundary defined in CLAUDE.md. Each button's click handler calls the new `OpenPath` Wails method and surfaces failures via the existing `useToast()` composable.

## Risks / Trade-offs

- **[Risk] CLI probe blocks the tab when a binary hangs** → Mitigation: each probe runs under `context.WithTimeout(ctx, 2*time.Second)`, and the top-level `ListProviders` waits on a `sync.WaitGroup` so the longest probe bounds the total latency at ~2 s, not 4 s.
- **[Risk] User PATH differs from app PATH on macOS launches from Finder** → Mitigation: rely on `exec.LookPath` returning empty when the binary isn't visible to the app process; surface this as "not installed" icon. Document the macOS PATH caveat in a follow-up if it bites in practice (out of scope for this change).
- **[Risk] `OpenPath` could be misused to open arbitrary attacker-supplied paths** → Mitigation: the Wails app is a single-user desktop tool; the only callers are our own UI, and the OS handler will refuse non-existent paths. We do not whitelist by directory because future callers (marketplace cache) live outside `~/.claude`.
- **[Risk] Regex over-matches a non-version triple in CLI output** → Mitigation: anchor on word boundaries and take the first match. If output ever leads with a non-version triple we revisit per-provider parsers.
- **[Risk] `wails generate module` regeneration drift** → Mitigation: tasks.md mandates running it after `Provider` struct edits and committing the regenerated `frontend/wailsjs/go/**` files in the same commit.

## Migration Plan

1. Backend change is additive at the API surface (one new bound app, one new field). The struct field removal is the only break.
2. After backend changes land, run `wails generate module` to regenerate `frontend/wailsjs/go/**`.
3. Update `ProviderCard.vue` and `provider-management` consumers in the same commit; existing tests catch any reference to `commandsDir`/`skillsDir`.
4. No data migration; no persisted state references the removed fields.
5. Rollback: revert the commit. No external state is modified by this change.

## Open Questions

- Should the badge variant be `info` (current daisyUI blue) or a new `version` variant in `AppBadge`? Current call: reuse `info` to avoid expanding the primitive's API in this change.
- Should we localize the "Not installed" aria label (zh-TW)? Current call: defer until the broader i18n pass; keep the codebase's existing English-default convention for now.
