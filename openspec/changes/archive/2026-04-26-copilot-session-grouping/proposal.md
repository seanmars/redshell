## Why

The Session History page renders Claude sessions grouped by working directory but renders Copilot sessions as a flat, unstructured list. When a user has worked in many repos with Copilot, the list becomes long and lacks the contextual grouping users already learn from the Claude tab. Aligning Copilot's listing with Claude's grouped layout makes per-agent UX consistent and lets users locate sessions by project at a glance.

## What Changes

- The Copilot session adapter (`internal/sessionhistory/copilot`) SHALL return a grouped `Listing` (`kind == "groups"`) instead of the current flat `Listing`.
- Sessions SHALL be grouped by their resolved working directory, taken from `workspace.yaml.cwd` (with `git_root` and `repository` as ordered fallbacks when `cwd` is empty).
- Group headers SHALL display the same shortened `{parent}/{root}` format used by Claude, with the parent segment dimmed and the full path available in the native tooltip.
- Groups SHALL be ordered by the most recent `created_at` (or `updated_at` when newer) of any session inside the group, descending.
- Sessions inside a group SHALL retain the existing `created_at` descending order.
- The frontend `SessionList.vue` no longer needs to switch on `listing.kind === "flat"` for Copilot; the existing `groups` rendering path SHALL be reused.
- The Copilot session row content (summary, branch, `created_at`) is preserved unchanged — grouping does not alter row metadata.

## Capabilities

### New Capabilities

<!-- None: this is a refinement of an existing capability. -->

### Modified Capabilities

- `session-history-viewer`: The "Copilot session list is a flat list" requirement is replaced with a grouped variant. The shared "Session list rows show summary metadata cheaply" requirement is unchanged for Copilot row content. The shared two-pane / pagination / redaction requirements are unaffected.

## Impact

- Backend: `internal/sessionhistory/copilot/reader.go` and the Copilot adapter in `internal/sessionhistory/service.go` (the path that returns `Kind: "flat"`). New helper to compute group keys from `cwd`/`git_root`/`repository` with the same shortening rules already used for Claude.
- Frontend: `frontend/src/components/sessions/SessionList.vue` — the existing `kind === "groups"` branch is reused; the `kind === "flat"` branch becomes dead code for Copilot but is left intact for any future agent that needs flat listing.
- Tests: `internal/sessionhistory/copilot/reader_test.go` and any service-level tests that assert `Listing.Kind == "flat"` for Copilot must be updated.
- Wails bindings (`frontend/wailsjs/go/models.ts`): no shape change — `Listing` already supports both `groups` and `flat` discriminants.
- Docs: project README and `CLAUDE.md` need no edits; the spec under `openspec/specs/session-history-viewer/spec.md` will be updated via the delta in this change.
- No data migration: session files on disk are read-only and untouched.
