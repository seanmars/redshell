## 1. Preconditions

- [x] 1.1 Confirm the in-flight `browse-plugins-by-marketplace` change has been archived (manual verification tasks 4.1–4.4 complete and `openspec archive browse-plugins-by-marketplace` run). This pins the spec baseline that this change modifies.

## 2. Reset prior HTTP-API implementation

The first iteration of this change implemented manifest fetch via the GitHub/GitLab Contents API. The pivot to a local clone cache (Decision 1 in design.md) supersedes that work. Before adding the cache code, revert the HTTP path so two manifest-fetch implementations do not coexist.

- [x] 2.1 In `internal/plugin/service.go`, delete `fetchFileFromGitHub` and `fetchFileFromGitLab` and the `httpClient` / `githubAPIBase` / `gitlabAPIBase` fields on `Service`.
- [x] 2.2 Remove the `ghToken` / `glToken` parameters from `fetchForProvider` and from the `FetchAll` call site that derives them via `provider.Service.GetTokens()`.
- [x] 2.3 Delete `internal/plugin/service_test.go` tests that depend on `httptest.Server` (`TestFetchAll_GitHub_200_404_500`, `TestFetchAll_GitLab_200_404_500`, and the `httpsToHTTP` / `writeGHFile` / `newGitHubServer` / `httpsToHTTP.RoundTrip` helpers). Keep the parser tests (`TestManifestParser_*`); the `TestFetchForProvider_RequiredFieldsEnforced` test will be re-introduced in task 5.4 against a fixture cache directory.

## 3. Backend: keep the Copilot manifest path fix and parser plumbing

These tasks were completed in the prior iteration and survive the pivot — re-verify they still hold after the reset.

- [x] 3.1 In `internal/marketplace/service.go`, `providerMarketplaceFiles["copilot"]` is `".github/plugin/marketplace.json"`.
- [x] 3.2 `ProviderMarketplaceFiles` is exported so `internal/plugin` can import it.
- [x] 3.3 Internal types `manifestPlugin { Name, Source, Description string }` and `marketplaceManifest { Name string; Plugins []manifestPlugin }` exist in `internal/plugin/service.go`.

## 4. Backend: cache infrastructure in `marketplace`

- [x] 4.1 Add `marketplace.CacheDirName(id string) string` that returns the marketplace ID with every char in the set `: / \ * ? " < > |` replaced by `-`. Pure function; unit-tested.
- [x] 4.2 Add `marketplace.CacheRoot() string` that returns `filepath.Join(home, ".redshell", ".cache")`. Mirrors how `NewService` resolves `filePath`.
- [x] 4.3 Add `marketplace.CacheDir(id string) string` that returns `filepath.Join(CacheRoot(), CacheDirName(id))`.
- [x] 4.4 Add a per-ID mutex registry on `marketplace.Service`: `cacheMu map[string]*sync.Mutex` guarded by a `sync.Mutex`. Provide an internal helper `cacheLock(id) *sync.Mutex` that lazily allocates and returns the per-ID mutex.
- [x] 4.5 In `marketplace.Service.Add`, replace the body of `fetchMarketplaceNames` with a "clone or refresh into cache" routine: lock the cache mutex; if `CacheDir(id)` exists with a `.git/` subdirectory, leave it alone; otherwise `git clone --depth=1 <url> <CacheDir>`. Then read both manifest paths from the cache directory to populate the `name` map. The `defer os.RemoveAll(tmpDir)` is removed; the cache is persistent. Renamed to `ensureCacheAndReadNames`.
- [x] 4.5a If `git clone` in 4.5 fails for any reason, `os.RemoveAll(CacheDir(id))` to clean up the partially-created directory before propagating the error. Do not append the registry entry. Covers the `marketplace-management` "Clone fails during add" scenario.
- [x] 4.6 In `marketplace.Service.Remove`, after deleting the registry entry, lock the cache mutex and `os.RemoveAll(CacheDir(id))`. Surface but do not fail the removal if the cache delete errors (best-effort).
- [x] 4.7 Add `marketplace.Service.Refresh(id string) error`: lock the cache mutex; if `CacheDir(id)` does not exist or has no `.git/`, perform `git clone --depth=1`; otherwise `git -C <CacheDir> fetch --depth=1 origin` then `git -C <CacheDir> reset --hard FETCH_HEAD`. Return any git error verbatim, prefixed with `git refresh: `.
- [x] 4.8 Add `marketplace.Service.RefreshAll() ([]string, []string)` returning `(refreshedIDs, errors)`. Errors shaped `[<id>] git refresh: <reason>`. Iterate registered marketplaces; per-marketplace failures do not abort the loop.

## 5. Backend: `plugin.Service` reads from cache

- [x] 5.1 Rewrite `plugin.Service.fetchForProvider(m, prov)` (note: parameters reduced; `ghToken`/`glToken` gone): construct `path = filepath.Join(marketplace.CacheDir(m.ID), filepath.FromSlash(marketplace.ProviderMarketplaceFiles[prov]))`. `os.ReadFile` it; if `os.IsNotExist`, return error shaped `cache missing; click Refresh to re-clone`. On other I/O error, return that error verbatim. Unmarshal into `marketplaceManifest`; on parse error, return `manifest parse error: <err>`.
- [x] 5.2 Build a `[]MarketplacePlugin` from `manifest.Plugins`, requiring non-empty `name` and `source`. Drop entries missing either and continue. Keep `Project` populated from `parseGitURL(m.URL).repo` and `InstallName = name + "@" + marketplaceName` unchanged.
- [x] 5.3 In `FetchAll`, drop the `tokens, _ := s.providerSvc.GetTokens()` line and the `ghToken`/`glToken` arguments. Preserve the `[<marketplaceID>/<provider>] <message>` error format expected by the frontend's `errorsByMarketplace` getter.
- [x] 5.4 Re-introduced `TestFetchForProvider_RequiredFieldsEnforced` against a fixture cache directory; added `marketplace.NewServiceWithCacheRoot(filePath, cacheRoot string)` constructor and a `seedCache` test helper. The test writes a `.claude-plugin/marketplace.json` with three entries (two malformed, one good) and asserts only the well-formed entry survives.

## 6. Backend: Wails bindings

- [x] 6.1 In the app glue (`app/marketplace_app.go` or wherever `marketplace.Service` is exposed), add a `Refresh()` method that returns `RefreshAll`'s aggregated result. Implemented in `app/marketplace.go` as `MarketplaceApp.Refresh() RefreshResult`.
- [x] 6.2 Ran `wails generate module` from the project root; regenerated `frontend/wailsjs/go/app/MarketplaceApp.{d.ts,js}` and `frontend/wailsjs/go/models.ts` (now exports `app.RefreshResult`). Project has no checked-in regen script in `wails.json`; the manual command stands as the documented regen step.

## 7. Frontend: Refresh button and existing collapsible/tab UI

The collapsible-section + provider-tab UI from the previous iteration of this change can stay (tasks below are marked done where they survive). The new work is the Refresh button and the tweak that section content reads from the cache-backed plugin list.

- [x] 7.1 Reactive `Record<marketplaceID, 'claude' | 'copilot'>` tab-state map exists, default `'claude'`.
- [x] 7.2 Each marketplace section is wrapped in a collapsible (`<details>` or daisyUI `collapse`) defaulting to expanded.
- [x] 7.3 Each section has a daisyUI `tabs` control bound to that section's tab-state entry: "Claude Code" (`claude`) and "GitHub Copilot" (`copilot`).
- [x] 7.4 Plugin list inside the section is filtered to `pluginsByMarketplace[m.id].filter(p => p.provider === activeTab)`. Errors filtered similarly.
- [x] 7.5 Empty-tab message: "No plugins available for this provider in this marketplace." Error-only state shows the error text.
- [x] 7.6 Added a "Refresh" button to `BrowsePluginsView.vue` page header (ghost variant, with spinner). Click handler calls `store.refreshAll()` then `store.fetchAll()`. Disabled while refreshing or loading.
- [x] 7.7 Added `refreshing: ref(false)` and `refreshWarnings: ref<Record<string,string>>` to `frontend/src/stores/plugin.ts`. `refreshAll()` action parses the backend's `[<id>] <message>` errors with a separate regex (kept distinct from `errorsByMarketplace`'s scoped regex) so per-marketplace refresh warnings live in their own getter and render at section-header level rather than under a specific tab.
- [x] 7.8 Section header renders `refreshWarningFor(m.id)` as a daisyUI `alert alert-warning` block above the provider tabs.

## 8. Tests

- [x] 8.1 Existing parser fixture tests (`TestManifestParser_OptionalFieldsIgnored_Claude`, `..._Copilot`, `..._MalformedJSON`) remain valid and unchanged.
- [x] 8.2 `TestCacheDirName` in `internal/marketplace/service_test.go` covers `:` / `/` / `\` / `*` / `?` / `"` / `<` / `>` / `|` replacement, the GitLab-subgroup case, and idempotency.
- [x] 8.3 `TestService_RefreshUpdatesCache` initializes a bare git repo via `git init --bare` + a working clone that pushes a manifest (helper `makeBareRepo`), runs `Add`, mutates the bare repo through a second working clone, calls `Refresh`, and asserts the cache file content has updated.
- [x] 8.4 `TestService_AddCreatesCache`, `TestService_AddCloneFailureCleansCache`, and `TestService_RemoveDeletesCache` cover Add/Remove cache lifecycle including the partial-cache cleanup path from task 4.5a.
- [x] 8.5 `TestFetchAll_CacheMiss` in `internal/plugin/service_test.go` seeds two cache directories (one full, one claude-only) and asserts the union of plugins plus a `[<id>/copilot] cache missing` error for the half-populated marketplace.
- [x] 8.6 `TestService_RefreshConcurrentSameID` fires two `Refresh(id)` calls in parallel and asserts neither errors and no `.git/index.lock` is left behind.

## 9. Specs and validation

- [x] 9.1 `specs/plugin-browser/spec.md` reflects cache-backed reads, per-section provider tabs, and the Refresh action.
- [x] 9.2 `specs/marketplace-management/spec.md` delta describes Add/Remove cache lifecycle, the cache layout, and the Refresh requirements.
- [x] 9.3 `openspec validate fetch-plugins-per-provider-manifest` reports valid.

## 10. Manual verification

- [ ] 10.1 Register `https://github.com/anthropics/claude-code` as a marketplace. Confirm `~/.redshell/.cache/github.com--anthropics@claude-code/` exists with a `.git/` subdirectory and the expected manifest file (note the `@` is preserved per Decision 6, only `:`/`/`/`\\`/etc. are sanitized). Browse Plugins shows its section expanded, default tab `Claude Code`, listing every entry from the claude manifest. Switching to `GitHub Copilot` tab shows the cache-missing message ("cache missing; click Refresh to re-clone" only if the copilot manifest truly is not present).
- [ ] 10.2 Register `https://github.com/github/copilot-plugins` as a marketplace. Confirm the `GitHub Copilot` tab lists the copilot plugins and the `Claude Code` tab shows the empty-state message for the missing claude manifest.
- [ ] 10.3 Register the user's actual private GitLab marketplace `https://cggitlab.chinesegamer.net/mars/ai-tools` (the original failing case). Confirm `Add` succeeds without a configured GitLab PAT (uses git credential helper) and Browse Plugins shows the expected plugins on both tabs.
- [ ] 10.4 With the previous marketplace registered, manually delete its cache directory under `~/.redshell/.cache/`. Reload Browse Plugins; confirm both tabs show the cache-missing error. Click Refresh; confirm the cache is rebuilt and the plugins reappear.
- [ ] 10.5 Disable network (or block the marketplace host); click Refresh; confirm the page surfaces the per-marketplace `git refresh:` warning at the section header but continues to render the previously-cached plugin list (stale-cache fallback per Decision 9).
- [ ] 10.6 Click Remove on a marketplace card; confirm both the registry entry and the cache directory are gone.
- [ ] 10.7 Click a marketplace header; confirm the section collapses and re-clicks expand it. Confirm tab selection persists across collapse/expand within a session.
- [ ] 10.8 Install a plugin from each provider tab via the install modal; confirm the existing install flow still works end-to-end (provider selection in the modal, log streamed, plugin appears in Installed Plugins).
- [ ] 10.9 Trigger two Refresh clicks in rapid succession; confirm no `.git/index.lock` file is left behind in any cache directory and the UI does not double-render errors.
