## Context

RedShell's domain noun for the AI assistants it manages is currently "provider" — a generic, overloaded term. The Go side has `internal/provider/` and `app.ProviderApp`; the frontend has `stores/provider.ts`, `components/provider/`, a `/providers` route, and a `ProvidersTab` settings pane. The OpenSpec capability is `provider-management`. None of the underlying file formats (`~/.claude/...`, `~/.copilot/...`, `~/.redshell/marketplace.json`) use the literal string "provider" as a key, so the rename is purely a code/spec/copy refactor with no on-disk migration.

Wails enforces an unusual constraint: the binding bridge between Go and TypeScript regenerates `frontend/wailsjs/go/app/<AppName>.{js,d.ts}` from the bound struct types and `frontend/wailsjs/go/models.ts` from the package basenames. Renaming `app.ProviderApp` to `app.AgentApp` and the package `internal/provider` to `internal/agent` therefore propagates automatically to a new generated import path (`@wailsjs/go/app/AgentApp`) and a new TS namespace (`agent.Agent` instead of `provider.Provider`). All TS imports must move in lockstep with the Go rename — a partial rename leaves the build broken.

The unarchived OpenSpec change `provider-card-paths-version` (status: Complete, 4 minutes old at proposal time) still targets the `provider-management` capability. It must be archived first so its delta applies to the original spec name; otherwise the rename would orphan its requirements.

## Goals / Non-Goals

**Goals:**
- Make "agent" the single domain noun across Go code, frontend code, OpenSpec, and `CLAUDE.md`.
- Keep behaviour identical — every test that passed before passes after, with renamed identifiers.
- Land the rename in a sequence where every intermediate commit compiles and `go test ./...` + `pnpm run build` both succeed (no half-renamed states left behind).
- Preserve the on-disk format of `~/.redshell/marketplace.json`, `~/.claude/...`, and `~/.copilot/...` so existing user installs are unaffected.

**Non-Goals:**
- Changing the on-disk schema or the marketplace `Name` map keys (`"claude"` / `"copilot"` stay as-is).
- Renaming the binary (`redshell`) or the window title (`RedShell`).
- Renaming the CLI commands of the underlying agents (`claude`, `copilot`).
- Touching archived changes under `openspec/changes/archive/` — those are historical records and stay frozen.
- Renaming `usePluginInstaller` or `mergedPlugins` semantics — only the field/parameter name `provider` → `agent` changes inside them.

## Decisions

### Decision 1: Drop the old capability rather than alias it
**Choice**: Remove `openspec/specs/provider-management/spec.md` and create `openspec/specs/agent-management/spec.md` with the full requirement text (renamed). The delta records this as ADDED requirements under `agent-management` and a REMOVED requirement under `provider-management` with a `**Migration**: see agent-management capability` note.
**Alternative considered**: Keep `provider-management` as a deprecated alias that re-exports `agent-management`. Rejected — OpenSpec capabilities are documentation, not code; an alias just means two specs to keep in sync forever.
**Rationale**: A clean cut is cheaper to maintain. Spec history lives in git; no runtime cost to a hard rename.

### Decision 2: Rename the JSON tag from `provider` to `agent`
**Choice**: The JSON tag on `MarketplacePlugin.Provider` and `InstalledPlugin.Provider` becomes `agent`. The frontend type imports (`models.ts`) regenerate to match.
**Alternative considered**: Keep `json:"provider"` on the struct so the wire format stays the same; only rename the Go field name and TS-side property in the store. Rejected — RedShell is a single-binary desktop app; both ends ship together. There is no third party reading the JSON, so wire compatibility buys nothing and creates a permanent naming mismatch between the field and its tag.
**Rationale**: A divergence between identifier and tag would re-introduce the exact ambiguity this proposal removes.

### Decision 3: Preserve `"claude"` / `"copilot"` as the agent-id strings
**Choice**: Wherever an agent is identified by string (the `Marketplace.Name` map keys, `runProviderCmd(prov, ...)` argument values, the `ProviderMarketplaceFiles` map keys), the values remain `"claude"` and `"copilot"`. Only the variable that holds them is renamed (`prov` → `agentID`, etc.) and the map symbol is renamed (`ProviderMarketplaceFiles` → `AgentMarketplaceFiles`).
**Rationale**: These literals appear in `~/.redshell/marketplace.json` (`Name: {"claude": "...", "copilot": "..."}`) on every existing user install. Renaming them would require a migration step, which is out of scope.

### Decision 4: Stage the rename layer-by-layer in a single change
**Choice**: Implement in this order: (1) Go domain package (`internal/provider` → `internal/agent`), (2) Wails binding layer (`app/provider.go` → `app/agent.go`, `main.go`), (3) regenerate Wails TS bindings, (4) frontend store + components + router + UI copy, (5) OpenSpec capability rename + `CLAUDE.md`. Each layer is a logical commit; tests run between layers.
**Alternative considered**: One giant find-and-replace commit. Rejected — the moment Wails regenerates TS bindings, the frontend imports break until the frontend is also renamed. A single staged rebase keeps the diff reviewable but the working tree must reach a green state at the end of every layer.
**Rationale**: Predictable failure modes. If the build breaks at layer N, layer N is the suspect.

### Decision 5: Use `agent` (not `aiAgent`, `assistant`, or `cli`) as the domain noun
**Choice**: Single-word `agent`. Field is `agent`, type is `Agent`, store is `agentStore`, route is `/agents`.
**Alternatives considered**:
- `assistant` — accurate (Anthropic uses "AI assistant" in some surfaces) but loses the "does work" connotation that "agent" carries; agents in this app actually run plugins.
- `aiAgent` — verbose; the domain context (an AI tooling app) makes the `ai` prefix redundant.
- `cli` — describes the transport, not the domain entity.
**Rationale**: "Agent" is industry-standard in this product space and carries the right action-oriented connotation.

## Risks / Trade-offs

- **Risk: Stale `@wailsjs/go/app/ProviderApp` imports persist after frontend rename and break the build silently** → Mitigation: After the Go rename, delete `frontend/wailsjs/go/app/ProviderApp.*` and run `wails generate module` (or `wails dev` once) before touching frontend imports. Run `rg "ProviderApp\|provider\\.Provider\|useProviderStore" frontend/src` after the rename — must return zero matches.
- **Risk: `CacheDirName` already replaces invalid chars; renaming the symbol on `ProviderMarketplaceFiles` does not touch on-disk state, but a developer might assume cache dirs need clearing** → Mitigation: Call out in tasks.md that cache dirs and `marketplace.json` are unaffected. Add a sanity check in the rename-PR description.
- **Risk: The unarchived `provider-card-paths-version` change still targets `provider-management`** → Mitigation: Archive `provider-card-paths-version` first (its status is already Complete). Validation: `openspec list` should show no active changes targeting `provider-management` before this rename starts.
- **Risk: External docs / READMEs / community references using the old names** → Mitigation: This change updates `CLAUDE.md` and the OpenSpec specs; broader docs (`docs/superpowers/`, README) are out of scope for this rename and may continue to use mixed terminology until separately updated.
- **Trade-off: Wide diff** → ~30 Go identifier sites, ~12 frontend files, 1 spec rename, 1 generated-bindings regeneration. Reviewable because the change is purely mechanical, but the PR will be large.

## Migration Plan

1. Pre-flight: archive `provider-card-paths-version` so all current `provider-management` deltas are folded into the live spec before this rename starts.
2. Implement the rename layer-by-layer per Decision 4. After each layer commit:
   - `go fmt ./... && go vet ./... && go test ./...`
   - `cd frontend && pnpm run build && pnpm run test:unit && pnpm run lint && cd ..`
3. After all layers land, run `wails dev` once and click through Settings → Agents tab + Browse + Installed to verify no runtime regressions.
4. Run the frontend leak check from `CLAUDE.md`:
   ```sh
   rg "class=\"[^\"]*\b(btn|card|alert|modal|tabs|tab|input|checkbox|collapse|badge|loading|toast|dropdown|avatar|select|textarea|radio|toggle|range|footer)\b" \
     frontend/src/views frontend/src/components/settings frontend/src/components/plugin frontend/src/components/marketplace frontend/src/components/agent
   ```
   Note the path change: `frontend/src/components/provider` becomes `frontend/src/components/agent`.
5. Rollback: `git revert` the rename commit(s). Because no on-disk schema changed, rollback is purely a code revert.

## Open Questions

- Should the i18n-ready UI copy use "Agent" (capital A, treating it as a proper noun for these specific tools) or "agent" (lowercase, treating it as a category)? **Default**: capitalised in UI labels ("Agents tab", "Agent card"), lowercase in body copy ("Install Claude Code to enable this agent."). Re-evaluate during frontend implementation.
- The router currently redirects `/providers` to `/settings?tab=providers`. Should the old `/providers` URL keep working for any users with bookmarks, redirecting to the new `/settings?tab=agents`? **Recommendation**: Yes — leave a `/providers` → `/settings?tab=agents` redirect so existing bookmarks continue to work. Drop after one release.
