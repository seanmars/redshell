## Why

`plugin.Service.FetchAll` currently lists directory entries under `plugins/` of every registered repo, once per provider, and tags each entry as both `claude` and `copilot`. The provider label is only metadata; nothing in the fetch path actually honors each provider's marketplace convention. A second, deeper failure was uncovered while debugging the empty Browse Plugins page on a real private GitLab marketplace: the GitHub/GitLab Contents API requires per-provider PATs that the user does not have configured, while `git clone` (which the system already uses in `marketplace.Service.fetchMarketplaceNames` to read display names on `Add`) succeeds via the user's existing git credential helper. The practical symptoms are:

1. Every plugin is emitted twice (duplicated across providers) or zero times if the repo lays plugins out differently.
2. A marketplace the user knows has plugins (per its `marketplace.json` manifest) appears empty in Browse Plugins because (a) its plugins are not structured as `plugins/<name>/` subdirectories and (b) on a private GitLab repo the API call returns `404 Project Not Found` (GitLab masks unauthorized as not-found), which the fetcher silently treats as "no plugins" with no error surfaced.
3. `internal/marketplace/service.go` has a separate bug: the Copilot manifest path constant is `.github/marketplace.json` but the real path is `.github/plugin/marketplace.json`, so the display-name lookup has been silently broken for every Copilot marketplace.

At the UI layer, the just-completed `browse-plugins-by-marketplace` change collapsed "all providers" into one per-marketplace list. With real per-provider manifests the two provider plugin sets become materially different again, so the user has also asked for collapsible marketplace sections with claude/copilot tabs nested inside them — a scoped provider filter, not a global one.

## What Changes

- **Backend (manifest source switches from HTTP API to cached `git clone`)**: `marketplace.Service.Add` shallow-clones each new marketplace repo into a persistent cache directory under `~/.redshell/.cache/<sanitized-id>/` instead of into a tmpdir that is wiped at function exit. `plugin.Service.FetchAll` reads both providers' `marketplace.json` files directly from the on-disk clone — no HTTP call. This unifies authentication on the user's existing git credentials (SSH key, git-credential-manager) and removes the GitHub/GitLab token requirement for plugin browse.
- **Backend (Refresh action)**: introduce an explicit `RefreshAll` (or `Refresh(marketplaceID)`) entry point that performs `git fetch --depth=1 && git reset --hard origin/<default>` on each cache. `FetchAll` itself becomes a pure read of the local cache and never touches the network. A user-triggered Refresh button on the Browse Plugins page calls the new entry point.
- **Backend (cache lifecycle)**: `marketplace.Service.Remove` deletes the corresponding cache directory in addition to removing the registry entry. Cache misses on `FetchAll` (e.g. user manually deleted the directory) trigger an automatic re-clone; refresh failures (network down, auth lost) keep the stale cache and surface a per-marketplace warning.
- **Backend (concurrency)**: a per-cache-directory mutex serializes `git fetch`/`reset` operations for the same marketplace to prevent corrupting the working tree if `RefreshAll` and a manual Refresh land at the same time.
- **Backend (manifest paths)**: fix `internal/marketplace/service.go` `providerMarketplaceFiles["copilot"]` from `.github/marketplace.json` to `.github/plugin/marketplace.json`. Same path is read from the cache by the plugin fetcher.
- **Backend (manifest parser)**: add a tolerant parser that accepts both Claude and Copilot manifest schemas — common required surface is `{ name, source, description }` per plugin entry; Claude adds `category`, Copilot adds `skills[]`. Extra fields are ignored; missing required fields drop the entry with a per-marketplace error.
- **Backend (delete HTTP path)**: remove `fetchFileFromGitHub` / `fetchFileFromGitLab` and the `ghToken` / `glToken` plumbing through `fetchForProvider`. `provider.Service.GetTokens()` is no longer called from the plugin browse flow (it remains for any other future use).
- **Spec (`plugin-browser`)**: re-introduce a provider-filter requirement, this time **scoped inside each marketplace section** (tabs for claude / copilot), and add a "collapsible marketplace section" requirement. Add scenarios for cache-backed reads, manual refresh, automatic re-clone on cache miss, and stale-cache fallback on refresh failure.
- **Spec (`marketplace-management`)**: amend Add and Remove requirements to include cache-directory creation and deletion as part of their atomic behavior; add a requirement that defines the cache directory layout.
- **Frontend (Browse Plugins)**: make each marketplace section in `BrowsePluginsView.vue` collapsible (default expanded), add a claude/copilot tab strip inside each section, and add a top-level "Refresh" button that calls the new backend refresh entry point. Sections whose currently selected provider has no plugins or has a refresh warning show the per-section message.
- **No model change**: the `MarketplacePlugin` JSON shape and `InstallName = name + "@" + marketplaceName` format stay the same, so the install flow and store getters (`pluginsByMarketplace`, `errorsByMarketplace`) need no edits.
- **Deferred (noted, not implemented here)**: rendering of Claude-only `category` and Copilot-only `skills[]` on `PluginCard`. Left for a future change. A "clear all caches" maintenance action is also deferred.

## Capabilities

### New Capabilities
<!-- None. -->

### Modified Capabilities
- `plugin-browser`: change the source of per-marketplace plugin lists from a directory listing of `plugins/` to a manifest-driven parse that reads from a local clone cache; re-introduce a per-section provider filter; add a collapsible section control; add an explicit Refresh action.
- `marketplace-management`: extend Add to include initial clone into the cache; extend Remove to delete the cache; add a cache-layout requirement.


## Impact

- Backend: `internal/plugin/service.go` (rewrite `fetchForProvider`, delete `fetchFileFromGitHub` / `fetchFileFromGitLab`, add cache-read helpers), `internal/marketplace/service.go` (`Add` writes to persistent cache, `Remove` cleans cache, add `Refresh` / `RefreshAll`, `providerMarketplaceFiles` path fix and export).
- Frontend: `frontend/src/views/BrowsePluginsView.vue` (collapsible sections + tabs + Refresh button), `frontend/src/stores/plugin.ts` (call new Refresh binding). Generated Wails models gain a Refresh entry.
- Network: removed from `FetchAll`; concentrated in `Add` (one clone) and Refresh (one fetch+reset per marketplace). The reduction means the rate-limiting risk noted in the previous iteration of this design is gone.
- Auth: HTTP PATs no longer required for plugin browse. Reuses the user's existing git credential helper (SSH key, OS keychain, git-credential-manager). Repos requiring credentials the user has not configured for git will fail at clone/refresh time with a surfaced error.
- Disk: each registered marketplace adds one shallow clone (typically <1 MB) under `~/.redshell/.cache/`. `Remove` cleans up. No automatic eviction.
- Data: no change to the on-disk format of `~/.redshell/marketplace.json` itself; the cache directory is parallel to it.
- Known risk: a corrupt cache (interrupted clone, partially-deleted directory) will be detected at read time; the chosen recovery is auto-reclone for missing/invalid working trees. A truly broken state may require the user to manually delete the cache directory, mitigated by surfacing a clear error.
