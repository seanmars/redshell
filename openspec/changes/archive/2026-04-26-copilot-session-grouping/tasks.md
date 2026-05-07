## 1. Backend grouping in the Copilot adapter

- [x] 1.1 Add a `resolveCwd(workspace)` helper in `internal/sessionhistory/copilot/reader.go` that returns the first non-empty value from `cwd`, `git_root`, `repository`, falling back to the literal `"(unknown)"`.
- [x] 1.2 Add a `sessionRecency(meta)` helper that returns `max(parsedTime(created_at), parsedTime(updated_at))` for a `SessionMeta`, with safe fallback when both are unparseable.
- [x] 1.3 In the Copilot listing path, bucket parsed `SessionMeta` rows into a `map[string][]SessionMeta` keyed by the resolved cwd, preserving the per-bucket `created_at` descending order already produced by `ListSessions`.
- [x] 1.4 Materialize a `[]SessionGroup` whose `Cwd` field carries the full resolved cwd (not the shortened form) and whose `Sessions` slice is the bucket contents.
- [x] 1.5 Sort the `[]SessionGroup` slice by max `sessionRecency` inside each group, descending.
- [x] 1.6 Update the Copilot branch of `internal/sessionhistory/service.go` so it returns `Listing{ Kind: "groups", Groups: groups, Flat: nil }` instead of the current flat shape.

## 2. Backend tests

- [x] 2.1 Update `internal/sessionhistory/copilot/reader_test.go` (or add new `..._grouping_test.go`) to assert the listing kind is now `"groups"` and that `Listing.Flat` is empty.
- [x] 2.2 Add a fixture covering two sessions in the same `cwd` and one session in a different `cwd`, and assert exactly two groups exist with the expected memberships.
- [x] 2.3 Add a fixture where `cwd` is empty but `git_root` is set, and assert the session lands under the `git_root` group key (not under `(unknown)`).
- [x] 2.4 Add a fixture where all three fallback fields are empty, and assert the session lands in a single `(unknown)` group with the literal header.
- [x] 2.5 Add a multi-group fixture where group A's newest session is older than group B's newest, and assert group B sorts before group A.
- [x] 2.6 Add a within-group ordering test confirming sessions inside a group remain sorted by `created_at` descending.

## 3. Frontend wiring

- [x] 3.1 Confirm `frontend/src/components/sessions/SessionList.vue` renders the Copilot listing through the existing `kind === "groups"` branch with no code change. If `SessionListItem.vue` (or its Copilot variant) hides any field that should still be visible inside a group row (summary / branch / `created_at`), keep them visible.
- [x] 3.2 Run `pnpm run test:unit` in `frontend/` and ensure the `stores/sessionHistory` tests still pass; add an assertion that the Copilot fixture mock returns `kind: "groups"`.
- [x] 3.3 Update any existing frontend mock or fixture that previously asserted `listing.kind === "flat"` for Copilot to expect `"groups"`.

## 4. Verification

- [x] 4.1 Run `go fmt ./internal/sessionhistory/... ./app/... ` and `go vet ./...` on changed Go packages.
- [x] 4.2 Run `go test ./internal/sessionhistory/...` and confirm all tests pass.
- [x] 4.3 Run `pnpm run lint`, `pnpm run format`, and `pnpm run test:unit` inside `frontend/`. (Note: pre-existing `'tail' is assigned a value but never used` ESLint error in `SessionListItem.vue` reproduces on `main` without this change — out of scope.)
- [ ] 4.4 **(USER ACTION REQUIRED — not auto-verifiable)** Run `wails dev`, open the Session History page, switch to the Copilot tab, and visually confirm: (a) sessions appear under collapsible groups, (b) the group header shows `{parent}/{root}` with the parent dimmed and the full path in the tooltip, (c) the `(unknown)` bucket renders only when applicable, (d) selecting a session inside a group still loads its event timeline correctly. Vue store mocks and Go tests do not exercise `AppCollapse` rendering, so this gate must be cleared by hand before archive.
- [x] 4.5 Run `openspec validate copilot-session-grouping --strict` and resolve any reported issues.

## 5. Spec sync and archive

- [ ] 5.1 Once implementation is verified end to end, run `openspec archive copilot-session-grouping` so the delta is folded into `openspec/specs/session-history-viewer/spec.md` and the change moves to `openspec/changes/archive/`.
- [ ] 5.2 Verify the post-archive `openspec/specs/session-history-viewer/spec.md` no longer contains the "Copilot session list is a flat list" requirement and now contains "Copilot session list is grouped by working directory" with all six scenarios.
