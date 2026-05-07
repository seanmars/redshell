## Why

The codebase currently uses "provider" to refer to the AI coding assistants RedShell manages (Claude Code, GitHub Copilot). This term is ambiguous — in software it more often denotes an OAuth/identity provider, a cloud provider, or a Vue/React `provide` injection key — and it does not match how the user surface in the rest of the ecosystem refers to these tools (Claude Code calls them "agents"; GitHub markets Copilot Coding Agent). Aligning the domain noun on "agent" makes the UI, code, and specs read naturally and removes a long-standing source of mental translation when reading the code or talking about it.

## What Changes

- **BREAKING** Rename the Go package `internal/provider` to `internal/agent`. The exported type `provider.Provider` becomes `agent.Agent`; constructors and helpers (`NewService`, `ListProviders`, `providerSpec`, `providerSpecs`, `probeVersion`) are renamed accordingly (`ListProviders` → `ListAgents`).
- **BREAKING** Rename the Wails-bound app type `app.ProviderApp` (file `app/provider.go`) to `app.AgentApp` (`app/agent.go`). The bound method `ListProviders` becomes `ListAgents`. The Wails CodeGen output `frontend/wailsjs/go/app/ProviderApp.*` is regenerated as `AgentApp.*` and the `provider` namespace in `models.ts` becomes `agent`.
- **BREAKING** Rename the cross-cutting domain field `provider` on `MarketplacePlugin` and `InstalledPlugin` (Go struct field + `json:"provider"` tag) to `agent` (`json:"agent"`). All call sites in `internal/plugin/service.go` (parameters named `prov`, the `runProviderCmd` helper, the `EnsureMarketplace`/`Install`/`ListInstalled`/`Uninstall` signatures) are renamed. The error format `[<marketplaceID>/<provider>] <msg>` becomes `[<marketplaceID>/<agent>] <msg>`; the frontend parser in `stores/plugin.ts` is updated in lockstep.
- **BREAKING** Rename `marketplace.ProviderMarketplaceFiles` to `marketplace.AgentMarketplaceFiles`. Map keys (`"claude"`, `"copilot"`) are unchanged — only the symbol name moves.
- **BREAKING** Rename the OpenSpec capability `provider-management` to `agent-management`. The spec file moves from `openspec/specs/provider-management/spec.md` to `openspec/specs/agent-management/spec.md`. All requirement text inside the spec replaces "provider" with "agent" (Provider → Agent in capitalised contexts, "AI provider" → "AI agent", "provider card" → "agent card").
- Rename frontend assets: store `frontend/src/stores/provider.ts` → `agent.ts` (`useProviderStore` → `useAgentStore`, store id `'provider'` → `'agent'`), folder `frontend/src/components/provider/` → `frontend/src/components/agent/`, component `ProviderCard.vue` → `AgentCard.vue`, settings tab `components/settings/ProvidersTab.vue` → `AgentsTab.vue`. Router redirect `/providers` → `/agents` and query param `tab=providers` → `tab=agents`. UI copy ("Install <Label> to enable this provider.", "Providers" tab label, etc.) is updated to use "agent" wording.
- Update the project documentation: `CLAUDE.md` (Project Overview, Domain layering, Provider-specific paths, Frontend folder layout sections) is updated to use "agent" terminology. The on-disk `~/.claude` / `~/.copilot` paths and the `~/.redshell/marketplace.json` schema (whose `Name` keys are `"claude"` / `"copilot"`) remain unchanged — this rename only touches code and prose, not user data.
- Existing tests (`internal/provider/service_test.go`, `internal/plugin/service_test.go`, `frontend/src/stores/__tests__/plugin.test.ts`) are renamed and updated so the suite still passes after the rename.

## Capabilities

### New Capabilities
- `agent-management`: Replacement for the existing `provider-management` capability. Same behaviour, but every requirement text and scenario uses "agent" terminology and references the renamed types/methods.

### Modified Capabilities
- `provider-management`: Removed. Its requirements are reissued verbatim (with the rename applied) under the new `agent-management` capability.

## Impact

- **Code**: ~30 Go identifiers, ~12 Vue/TS files, the Wails-generated bindings, and one entry in the `Bind` array of `main.go` change. The change is mechanical but wide.
- **APIs**: The Wails bridge surface (visible to the frontend) changes — `ListProviders` becomes `ListAgents`, the `provider` model namespace becomes `agent`, and JSON field `provider` on plugin DTOs becomes `agent`. There is no external HTTP API to keep stable.
- **Persisted state**: None. The on-disk JSON files (`~/.redshell/marketplace.json`, `~/.claude/plugins/installed_plugins.json`, `~/.copilot/config.json`) keep their existing keys; the "claude"/"copilot" map keys are not renamed.
- **Specs / docs**: One capability is renamed in `openspec/specs/`. `CLAUDE.md` is updated. Archived changes under `openspec/changes/archive/` are left untouched (they are historical records).
- **In-flight work**: The unarchived change `provider-card-paths-version` will be archived before this rename lands so its delta still applies to the original `provider-management` spec.
