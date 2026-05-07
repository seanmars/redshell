## Why

The Providers tab currently shows raw config/commands/skills directories with no way to act on them, and the configured/not-configured badge gives a binary signal that hides whether the CLI itself is actually installed and at what version. Users want to (a) jump from the card to the relevant folder or settings file, and (b) see the installed CLI version at a glance so they can confirm an upgrade or diagnose a mismatch.

## What Changes

- Stop displaying the `commands` and `skills` directory rows on each provider card.
- Add a clickable **Directory** row that opens the provider's home folder in the OS file manager (`~/.claude` for Claude Code, `~/.copilot` for GitHub Copilot).
- Replace the **Config** row with a clickable **Configuration** row that opens the provider's primary settings file in the OS default handler (`~/.claude/settings.json` for Claude Code, `~/.copilot/config.json` for GitHub Copilot).
- Replace the binary "Configured / Not Configured" badge with the installed CLI version (e.g. `2.1.119`). When the CLI is not installed or the version cannot be detected, render an icon (warning glyph) instead of text.
- **BREAKING (Wails binding only):** the `Provider` struct shape returned by `ProviderApp.ListProviders` changes — `commandsDir` and `skillsDir` fields are removed, and a new `version` field plus a `settingsFile` field are added. Frontend imports auto-regenerate from `wails generate module`.
- Add new Wails-bound methods to (a) detect the CLI version per provider and (b) open a path in the host OS — both required to wire up the new UI controls.

## Capabilities

### New Capabilities
- `os-path-opener`: thin Wails-side capability for opening a filesystem path (file or directory) in the user's OS using the platform default handler. Used by ProviderCard and reusable by future cards.

### Modified Capabilities
- `provider-management`: drops the commands/skills path requirements, adds the Directory + Configuration clickable rows, and replaces the configured-status badge with a version-or-icon badge backed by `claude --version` / `copilot --version` output.

## Impact

- Affected backend code:
  - `internal/provider/service.go` — `Provider` struct gains `Version` and `SettingsFile`, drops `CommandsDir`/`SkillsDir`; `ListProviders` runs `claude --version` / `copilot --version` and parses semver.
  - `app/provider.go` — exposes the version-aware list. May add a refresh method if version probing is moved off the synchronous `ListProviders` path (decided in design.md).
  - New `internal/osopen/` (or similar) package — `OpenPath(path string) error` shelling to `explorer` on Windows, `open` on macOS, `xdg-open` on Linux.
  - New `app/system.go` (or fold into existing) — Wails-bound method `OpenPath`.
- Affected frontend code:
  - `frontend/src/components/provider/ProviderCard.vue` — full rewrite of body and badge.
  - `frontend/src/stores/provider.ts` — no API change beyond model regeneration.
  - `frontend/wailsjs/go/**` — regenerated, do not hand-edit.
- Tests: extend `internal/provider` tests with a fake exec runner so version parsing is unit-tested without requiring the CLIs to be installed on CI.
- Docs: update `CLAUDE.md` "Provider-specific paths" table only if the Configuration column wording changes meaningfully (likely no change — the table already lists per-provider paths).
