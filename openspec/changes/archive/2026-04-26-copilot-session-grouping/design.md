## Context

The Session History viewer (`session-history-viewer` capability) currently emits two listing shapes from the backend: `Listing.Kind == "groups"` (Claude — keyed by working directory) and `Listing.Kind == "flat"` (Copilot — sorted by `created_at` descending). The Wails-bound type already carries both shapes, and `frontend/src/components/sessions/SessionList.vue` already renders both branches.

For Claude the grouping is computed inside `internal/sessionhistory/claude/reader.go`, which inspects each session JSONL and reads the first non-empty `cwd` event. For Copilot, every session has a sibling `workspace.yaml` (parsed in `internal/sessionhistory/copilot/manifest.go`) that already exposes `cwd`, `git_root`, `repository`, `branch`, and timestamp metadata, so the working directory is available without scanning the JSONL events.

The constraint we work under is the existing spec contract:

- Group header SHOULD render the last two path segments (`{parent}/{root}`), with parent dimmed and the full path in `title`.
- Groups SHOULD be sorted by the most recent session timestamp inside them, descending.
- Sessions inside a group SHOULD keep their existing intra-list sort order.

## Goals / Non-Goals

**Goals:**

- Make the Copilot session list visually consistent with the Claude session list — collapsible groups keyed by the session's working directory.
- Reuse the existing `Listing.Kind == "groups"` wire shape and the existing `SessionList.vue` `groups` branch — no new TypeScript types, no new Vue components.
- Keep the row metadata for Copilot (summary, branch, `created_at`) unchanged.
- Pick a single, well-defined grouping key with documented fallbacks so the same `cwd` always lands in the same group across sessions.

**Non-Goals:**

- Re-grouping or re-sorting Claude sessions; this change does not touch the Claude adapter.
- Replacing the `flat` listing shape from the wire — it stays in `models.ts` so a future agent can opt into flat without re-introducing the type.
- Reading or merging extra metadata from `events.jsonl` to determine `cwd`. We trust `workspace.yaml` for grouping; that is what the Copilot CLI itself writes.
- Persisting collapsed/expanded group state across navigation. Group expand state is per-mount, matching Claude's current behaviour.

## Decisions

### Group key: `workspace.yaml.cwd`, with documented fallbacks

We group by the first non-empty value from this ordered list:

1. `workspace.yaml.cwd`
2. `workspace.yaml.git_root`
3. `workspace.yaml.repository`
4. The literal string `"(unknown)"` if all of the above are empty.

**Rationale.** `cwd` is the most direct equivalent of Claude's grouping field — it represents the directory the user actually invoked the agent from. `git_root` is a near-equivalent fallback when the user launched from a subdirectory, and `repository` is the last-resort signal that at least groups same-repo sessions together. We considered grouping strictly by `repository` (more semantic), but Copilot's `repository` field is sometimes empty for sessions started outside a git repo, and it would split sessions in the same `cwd` across groups whenever the repo metadata changes. We also considered grouping by `git_root` first, but that would split a user's intentionally separate worktrees with the same `git_root` — `cwd` is the user-visible breakpoint.

The `"(unknown)"` bucket guarantees we never drop a session from the listing, mirroring how Claude handles sessions whose JSONL lacks a `cwd` event.

### Group display: reuse Claude's `shortPath()` helper

Group headers render via the existing `shortPath()` helper in `SessionList.vue` (`{parent}/{root}` with parent dimmed, full path in `title`). No frontend code change is required if the backend supplies the resolved path string in `SessionGroup.cwd` — same field name Claude already uses.

**Rationale.** Identical visual treatment is the whole point of the change; introducing a Copilot-specific renderer would defeat the consistency goal. The `(unknown)` bucket renders literally as its own header — a one-off cosmetic price for a guaranteed total.

### Group sort: descending by max(`created_at`, `updated_at`) inside the group

Each Copilot session has both `created_at` and `updated_at`. We pick `max(created_at, updated_at)` per session, then take the max of those across the group, then sort groups descending. Sessions inside the group keep `created_at` descending — unchanged.

**Rationale.** Using `updated_at` when newer surfaces recently resumed sessions to the top of their group, which matches user intent ("show me the project I touched most recently"). We considered using only `created_at` for parity with the existing flat sort, but that would push a freshly-resumed session under newer-but-untouched siblings. Claude already uses session-file `mtime` (its closest equivalent of "last touched"), so this choice keeps both adapters semantically aligned.

### Backend shape change is local to the Copilot adapter

The change is contained inside `internal/sessionhistory/copilot/reader.go` plus the Copilot branch of `internal/sessionhistory/service.go`. The adapter:

1. Iterates session directories as today.
2. Parses each `workspace.yaml` (already done).
3. Buckets each `SessionMeta` into a `map[string][]SessionMeta` keyed by the resolved `cwd` string.
4. Materializes `[]SessionGroup{ Cwd: key, Sessions: sortedSessions }`.
5. Returns `Listing{ Kind: "groups", Groups: ..., Flat: nil }`.

**Rationale.** The change is one shape transformation; it does not need a new package. Keeping it inside the existing reader avoids introducing a new "grouping" abstraction we'd have to maintain separately for each agent.

## Risks / Trade-offs

- **[Risk] Sessions started without a workspace context all collapse into `(unknown)`.** If a user has many such sessions, the group becomes a long catch-all with low utility.
  → Mitigation: the fallback chain (`cwd` → `git_root` → `repository`) covers the common cases. The `(unknown)` group is documented and ordered by recency like any other group, so a power user with many such sessions still gets recency ordering.

- **[Risk] Two visually identical group headers (`{parent}/{root}`) for different absolute paths.** Two repos with the same name under different parents collapse to the same display string but the actual `cwd` differs, so they render as two distinct groups with the same visible label. The full path is in the `title` tooltip, but at-a-glance the user cannot tell them apart.
  → Mitigation: same trade-off Claude already accepts; consistency wins. A future change could disambiguate by appending the host or a longer path suffix when collisions are detected, but that is out of scope.

- **[Risk] `workspace.yaml` is malformed for some session directory.** The current Copilot reader skips such sessions silently (returns the error up the dispatch). With grouping, the same skip behaviour applies — a corrupt session never reaches a group.
  → Mitigation: keep the existing skip-on-parse-error path. No new failure modes introduced.

- **[Trade-off] We lose the strict global "newest session first" ordering.** With flat listing, a recently-created session in a rarely-used repo appeared at the top. With grouping, it appears at the top of its repo group, but its repo group sits where the group sort places it (which may not be first if a different repo has an even newer session). Group sort by max-timestamp keeps this close to "newest at top" but not identical.
  → Mitigation: this is the explicit ask in the proposal — match Claude's UX. Users who want pure recency can collapse all groups and read the headers in order.

- **[Trade-off] The `flat` arm of `SessionList.vue` becomes dead code for the only agent that used it.** We keep the branch to avoid churning the wire type and the component, both of which may be reused by a future agent.
  → Mitigation: leave the branch with no consumer; flag it for removal if no other agent claims it within two further changes.

## Migration Plan

No on-disk migration is needed — Copilot session files are not modified. Rollout is purely a code change:

1. Land the backend grouping in `internal/sessionhistory/copilot/reader.go` plus the service-level dispatch.
2. Update `internal/sessionhistory/copilot/reader_test.go` to assert the grouped shape (multiple groups, header sort, intra-group sort).
3. The frontend already renders `kind === "groups"`; smoke-test in `wails dev` to confirm the Copilot tab now shows collapsibles.
4. Ship in a single commit — no feature flag, since the listing is visual-only and the JSON shape was already a discriminated union.

Rollback is a single revert of the adapter change; the wire type and the frontend branch for `flat` are never removed in this change, so a revert restores prior behaviour cleanly.
