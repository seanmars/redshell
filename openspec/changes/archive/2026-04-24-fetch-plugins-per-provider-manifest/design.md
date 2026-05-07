## Context

`plugin.Service.FetchAll` loops `[]string{"claude", "copilot"}` over every registered marketplace and calls `fetchForProvider(m, prov, ghToken, glToken)`. `fetchForProvider` in turn calls `FetchFromGitHub(m.URL, ...)` or `FetchFromGitLab(...)`, both of which list directories under a hard-coded `plugins/` path at the repo root. The `prov` value is used only to choose a display name and to tag each resulting `MarketplacePlugin` — it does *not* steer the fetch. Consequences:

- If a repo stores both claude and copilot plugins side-by-side under `plugins/`, every plugin is emitted twice with different `provider` tags.
- If a repo stores plugins outside `plugins/<name>/` (e.g. source paths declared in a `marketplace.json` manifest), nothing is returned.
- Today's Browse Plugins page (just rewritten by `browse-plugins-by-marketplace`) groups by marketplace and hides provider, so the duplicates merge optically but the symptom reported by the user — "marketplace 的 plugin 卻沒正確顯示" — is the failure mode for the second case.

A second issue surfaced while debugging on a real private GitLab marketplace (`https://cggitlab.chinesegamer.net/mars/ai-tools`): the GitLab API returns `404 Project Not Found` for unauthorized callers (it deliberately conflates "missing" and "forbidden" to prevent enumeration). The existing `fetchFileFromGitLab` treats every 404 as "file absent", silently returning `(nil, nil)`. This suppresses the auth-failure case entirely — the user sees an empty Browse Plugins page with no error to investigate. Meanwhile `git clone` against the same URL succeeds because the user's git credential helper supplies SSH credentials transparently. This evidence drove the pivot in Decision 1 below.

A separate issue sits in `internal/marketplace/service.go:28`: `providerMarketplaceFiles["copilot"]` points at `.github/marketplace.json`, but the real path (confirmed against `github/copilot-plugins`) is `.github/plugin/marketplace.json`. So `fetchMarketplaceNames` has been silently returning nothing for every Copilot marketplace since it was written.

Both manifests have now been read end-to-end:

**Claude** (`.claude-plugin/marketplace.json`):
```
{ "$schema": "...", "name": "...", "version": "...", "description": "...",
  "owner": { "name": "...", "email": "..." },
  "plugins": [ { "name": "...", "description": "...", "source": "./plugins/foo",
                 "category": "...", "version"?: "...", "author"?: {...} } ] }
```

**Copilot** (`.github/plugin/marketplace.json`):
```
{ "name": "...", "metadata": { "description": "...", "version": "..." },
  "owner": { "name": "...", "email": "..." },
  "plugins": [ { "name": "...", "source": "./plugins/foo",
                 "description": "...", "version": "...", "skills": ["./skills/foo"] } ] }
```

Intersection required for our purposes: `plugins[].name`, `plugins[].source`, `plugins[].description`. Both optional fields (`category`, `skills`, `author`, `version`) are ignored by this change; a future change can surface them on `PluginCard`.

## Goals / Non-Goals

**Goals:**
- Each marketplace's plugin list reflects the provider-specific `marketplace.json` for that repo, not a directory listing.
- Claude plugins and Copilot plugins for the same marketplace are fetched independently; zero cross-provider duplication.
- Manifest reads are backed by an on-disk shallow clone cache; auth reuses the user's existing git credential helper; the GitHub/GitLab token plumbing for Browse Plugins is removed.
- `Add` populates the cache; `Remove` deletes it; `FetchAll` reads from it without network I/O; an explicit Refresh action is the only path that touches the network during browsing.
- Failed manifest reads (parse error, missing file in cache) produce a per-provider error on the affected marketplace section. Refresh failures keep the existing cache intact and surface a per-marketplace warning rather than blanking the section.
- Browse Plugins displays each marketplace section as a collapsible block with a claude/copilot tab strip inside; default state expanded, default tab = claude. A page-level Refresh button triggers `RefreshAll`.
- Fix the Copilot manifest path bug in `internal/marketplace/service.go` in the same commit range.

**Non-Goals:**
- Surfacing Claude-only `category` or Copilot-only `skills[]` on `PluginCard` (future change).
- Changing `InstallName` format or any part of the install flow.
- Supporting manifests hosted outside `.claude-plugin/` / `.github/plugin/` (e.g. at a custom path configured by the user).
- Resolving `plugins[].source` paths — the fetcher uses manifest `name` directly; validating that `source` points to a real directory is the provider CLI's job at install time.
- Automatic background refresh, scheduled refresh, or per-section refresh granularity. One global Refresh button covers all caches in this iteration.
- Cache eviction policies, TTL-based refresh, or "clear all caches" maintenance UI. Deferred.

## Decisions

### Decision 1: Read manifests from a persistent local clone cache populated by `git clone`
**Choice:** `marketplace.Service.Add` performs `git clone --depth=1 <url> ~/.redshell/.cache/<sanitized-id>/` and keeps the working tree in place. `plugin.Service.FetchAll` reads `<cache>/.claude-plugin/marketplace.json` and `<cache>/.github/plugin/marketplace.json` directly from disk; no HTTP API call. The existing `marketplace.Service.fetchMarketplaceNames` (display-name extraction on `Add`) is folded into the same clone, replacing its tmpdir + `defer RemoveAll` with the persistent cache path.

A new `marketplace.Service.Refresh(id)` / `RefreshAll()` performs `git fetch --depth=1 && git reset --hard origin/<HEAD>` against each cache directory. This is the only browse-time path that touches the network.

**Why:** Empirically, `git clone` succeeds against the user's private GitLab via SSH and git-credential-manager, while the HTTP Contents API requires a separately-configured PAT and silently fails with 404 when no token is present. Centralizing on git auth removes a class of confusing failures and reuses credentials the user has already configured for development. A persistent cache also unlocks future "show plugin update available" features (compare cached HEAD vs. fetched HEAD) without further architectural change.

This decision intentionally reverses an earlier exploratory direction (Contents API, ephemeral fetches) that prioritized request count over auth simplicity. The reversal was triggered by reproducing the failure mode on a real private repo and seeing that no amount of API-side error handling can recover from "the user has no PAT and is unwilling to mint one". The previous direction's argument that clones are slow is mitigated by (a) shallow `--depth=1` clones being small and fast for typical marketplace repos and (b) `FetchAll` no longer cloning at all — only `Add` and explicit Refresh do.

**Alternatives considered:**
- *Keep the HTTP Contents API path but add 404-vs-auth disambiguation by parsing the GitLab error body.* Rejected — fragile (depends on GitLab's exact error string), still requires the user to provision a PAT, and does nothing for the future-update use case.
- *TTL-based auto-refresh on `FetchAll`* (e.g. re-fetch if cache older than 1 h). Rejected for this iteration — adds invisible network I/O and a TTL knob that needs explanation. Manual Refresh is sufficient and matches the user's stated preference.
- *Full clone (no `--depth=1`).* Rejected — multi-MB downloads with no benefit; we only ever read the two manifest files.

### Decision 2: One tolerant parser, both schemas
**Choice:** Introduce an internal struct in `plugin/service.go`:

```go
type manifestPlugin struct {
    Name        string `json:"name"`
    Source      string `json:"source"`
    Description string `json:"description"`
    // category/skills/author/version intentionally unmapped for now
}
type marketplaceManifest struct {
    Name    string           `json:"name"`
    Plugins []manifestPlugin `json:"plugins"`
}
```

Both Claude and Copilot manifests share these three plugin fields at the required level, and both use top-level `name` + `plugins`. Top-level `description` (Claude) vs `metadata.description` (Copilot) is not consumed by this change, so the divergence is invisible to us. Parser rejects entries missing `name` or `source` and surfaces one aggregated error per marketplace-provider.

**Why:** Simpler than schema-per-provider parsing; the intersection is sufficient. If we need provider-specific fields later we can extend the struct with pointer / `omitempty` fields without breaking the other provider.

**Alternatives considered:**
- *Separate `claudeManifest` / `copilotManifest` structs selected by `prov`.* Rejected as premature — no diverging required field at this stage.

### Decision 3: Path resolution table lives in `marketplace`, consumed by `plugin`
**Choice:** Export `marketplace.ProviderMarketplaceFiles` (already done in the in-flight implementation) and have `plugin/service.go` import it for both the cache-relative read path and any error-message path-disclosure. Apply the Copilot-path correction (`.github/plugin/marketplace.json`) in this single source of truth.

**Why:** Keeping the authoritative copy in `marketplace` and re-using it from `plugin` avoids drift. Exporting is cheap (one identifier) and makes the bug-fix atomic: fix it in one place.

**Alternatives considered:**
- *Duplicate the map locally in `plugin/service.go`.* Rejected — invites the same drift that produced the current bug.

### Decision 4: Per-marketplace section collapsible, per-section provider tabs
**Choice:** In `BrowsePluginsView.vue`, each marketplace section becomes a `<details>`-style collapsible (native `<details>`/`<summary>` or a daisyUI `collapse` class, default `open`). Inside each section, a local `ref` holds the selected provider tab (`'claude' | 'copilot'`), defaulting to `'claude'`. The rendered plugin list is `pluginsByMarketplace[m.id].filter(p => p.provider === activeTab)`. Error list inside the section is also filtered to the active provider.

**Why:** This matches the user's UX ask (click to expand, tabs for claude/copilot, then plugin list). Keeping tab state local to the section avoids a global provider filter ref — each marketplace can be browsed independently, which is the whole point of grouping by marketplace. Default `claude` is arbitrary but matches the existing install-modal default.

**Alternatives considered:**
- *Global provider filter above the marketplace list.* Rejected — that's what `browse-plugins-by-marketplace` just deleted, and the user's description explicitly asks for tabs inside the section ("點擊該marketplace會展開然後有tab(copilot跟claude)").
- *Default tab follows whichever provider has more plugins in that section.* Rejected — unstable, surprising; pick a deterministic default.

### Decision 5: Spec delta re-introduces provider filter as a new requirement, not as a revival of the removed one
**Choice:** In `specs/plugin-browser/spec.md` (delta), MODIFY the existing "Browse plugins from all registered marketplaces" requirement to describe cache-backed manifest fetch + collapsible sections + Refresh, and ADD new requirements for "Filter plugins per marketplace section by provider" and "Refresh marketplace data on demand". Do not attempt to re-add the removed "Filter plugins by provider" requirement — its REMOVED reason in `browse-plugins-by-marketplace` stands; the new scoped filter is a materially different requirement (per-section, not global).

**Why:** Keeps the archive log honest: the global filter was removed because it duplicated card metadata and fought marketplace grouping. The new per-section filter exists because, post-manifest-driven fetch, each marketplace actually has two meaningfully distinct provider views.

### Decision 6: Cache directory naming
**Choice:** Cache subdirectory name = the marketplace's `id` with every filesystem-unsafe character replaced by `-`. The unsafe set is `:`, `/`, `\`, `*`, `?`, `"`, `<`, `>`, `|` — covers Windows reserved characters and POSIX path separators. `@` is intentionally **not** in the replacement set: it is filesystem-safe on every supported OS and preserving it keeps the directory name parseable back to the marketplace ID. Implemented as `marketplace.CacheDirName(id string) string` so `plugin` and `marketplace` agree on the same name without re-deriving it from the URL.

For `cggitlab.chinesegamer.net::mars@ai-tools` → `cggitlab.chinesegamer.net--mars@ai-tools`. For `github.com::anthropics@claude-code` → `github.com--anthropics@claude-code`. For a GitLab subgroup like `gitlab.com::group/subgroup@proj` → `gitlab.com--group-subgroup@proj`. Collisions are theoretically possible if two marketplaces' IDs differ only in `:` vs `-` placement; in practice the marketplace ID format `host::owner@repo` already prevents that because `:`/`@`/`/` appear at fixed positions.

**Why:** The marketplace ID is already canonical and deduplicated by `marketplace.Service.Add`. Deriving the cache name from the ID (rather than from the URL) keeps the mapping 1-to-1 with the registry and survives URL normalization differences (trailing slash, `.git` suffix). Single replacement char `-` keeps the directory name human-readable on inspection.

**Note on naming refinement from brainstorm:** An earlier brainstorm sketch used `{host}__{owner}__{repo}` as a `__`-delimited field template. The shipped rule (char-replace on the ID) was preferred during spec authoring because it (a) preserves `@`, which is filesystem-safe and aids human reading, (b) does not lose information (the brainstorm template would be ambiguous if `owner` contained `__`), and (c) is implementable as a one-line `strings.NewReplacer(...).Replace(id)` keyed off the canonical ID rather than a re-parsed URL. Both schemes are 1-to-1 with the marketplace ID; the shipped one is strictly less lossy.

**Alternatives considered:**
- *Hash the URL (e.g. SHA-256 hex prefix).* Rejected — opaque under `~/.redshell/.cache/` makes manual inspection hostile.
- *Mirror the URL path structure (`<host>/<owner>/<repo>/`).* Rejected — adds nesting and conflicts with single-character path-separator semantics on Windows.
- *Brainstorm-time `{host}__{owner}__{repo}` template.* Rejected for the reasons in the note above.

### Decision 7: Refresh is explicit; `FetchAll` is offline-only
**Choice:** `FetchAll` reads cache files only and never invokes git. A new `RefreshAll` (Wails-bound) walks every registered marketplace, runs `git fetch --depth=1 && git reset --hard origin/<HEAD>` per cache, and returns an aggregated result `{ refreshed: []string, errors: []string }`. The frontend Browse Plugins page gains a Refresh button that calls `RefreshAll` then re-runs `FetchAll`.

**Why:** Decoupling read from network avoids the implicit-network-on-page-load behavior of the current code, which is a poor fit for an interactive desktop app. Explicit Refresh also makes failures intelligible — the user clicked Refresh, it failed, here is why. Per-marketplace Refresh granularity is deferred; a single Refresh keeps the UI surface small.

**Alternatives considered:**
- *Refresh on every `FetchAll`.* Rejected — turns every page navigation into N git fetches.
- *TTL-gated refresh.* Rejected — see Decision 1.

### Decision 8: Per-cache-directory mutex
**Choice:** `marketplace.Service` holds an internal `map[id]*sync.Mutex` (lazy-initialized, guarded by a `sync.Mutex` of its own). Every operation that mutates a cache directory — initial clone in `Add`, `Refresh(id)`, `Remove(id)` — locks the cache mutex before invoking git or filesystem operations. `RefreshAll` locks each cache in turn (or in bounded-parallel goroutines, each holding its own mutex). `FetchAll` performs read-only file reads without taking the lock; if a refresh is rewriting the working tree concurrently, `FetchAll` may briefly observe an inconsistent file (manifest open during `git reset --hard`), surfaced as a parse error in that one cycle.

**Why:** `git fetch` and `git reset` are not safe under concurrent invocation against the same working tree; without a lock, two simultaneous Refreshes can leave a corrupt index or stuck `.git/index.lock` file. Per-directory locking is sufficient and avoids serializing unrelated marketplaces. Read-side lock-free is an accepted small risk: the rare race produces a one-shot parse error that disappears on the next read, not data corruption.

**Alternatives considered:**
- *Single global mutex over all cache operations.* Rejected — serializes RefreshAll across unrelated marketplaces, slow.
- *File-system advisory locking (`flock`).* Rejected — complexity not justified for a single-process app.

### Decision 9: Cache-miss recovery and stale-cache fallback
**Choice:**
- *Cache miss on `FetchAll`* (cache directory does not exist or has no `.git/`): the read returns a per-provider error shaped `[<id>/<provider>] cache missing; click Refresh to re-clone`. The next Refresh treats this as a fresh `git clone` (delete any partial directory, clone again).
- *Refresh failure with cache present*: keep the existing cache untouched, surface a marketplace-scoped warning shaped `[<id>] refresh failed: <reason>`. `FetchAll` continues to read the stale cache so the UI shows old plugins instead of going blank.
- *Refresh failure with cache absent*: surface the same error; section shows the cache-missing message.

**Why:** Maximum graceful degradation. Stale data is more useful than a blank page when the user is offline or the remote is temporarily unreachable; a clear Refresh-failed badge tells them why the timestamp is old. Auto-reclone on cache miss handles the "user nuked the cache directory" case transparently without requiring a separate Re-add flow.

**Alternatives considered:**
- *Refresh failure wipes the cache.* Rejected — loses the only known-good copy on transient network errors.
- *Cache miss returns no error and an empty section.* Rejected — that is exactly the current bug being fixed; silent empty state is the worst UX outcome.

## Risks / Trade-offs

- **Risk:** Manifests with non-standard `source` paths (e.g. sibling repos referenced via `../`) will still fetch but fail at install time with a confusing CLI error. → **Mitigation:** Out of scope for this change. File a follow-up to validate `source` shape when rendering.
- **Risk:** Disk usage grows with the number of registered marketplaces. → **Mitigation:** `--depth=1` keeps each clone small (typically <1 MB). `Remove` cleans up. A future "clear all caches" action is deferred but trivial.
- **Risk:** Users on machines with no git credentials configured for the marketplace's host will see clone/refresh failures. → **Mitigation:** This is the same surface area that already exists for `git clone` in any developer environment; the surfaced error in `RefreshAll` should mention "git authentication" so users know where to look. We accept that this UI does not embed credential setup itself.
- **Risk:** A truly corrupt cache directory (e.g. `git reset` killed mid-write) may produce repeated read failures until the user manually deletes it. → **Mitigation:** Decision 9 already covers cache-missing as auto-reclone; a future enhancement could detect "cache exists but `.git/` is broken" and re-clone too. Out of scope for this iteration.
- **Risk:** A repo missing one manifest (claude-only or copilot-only marketplace) shows an empty state on the missing provider's tab on every page load. If most marketplaces are single-provider, users see the empty state on the Copilot tab for every claude-only repo. → **Mitigation:** Acceptable. The empty state per tab is strictly more informative than today's silent drop. A future refinement could hide tabs for providers the marketplace does not advertise, but that requires a separate signal.
- **Trade-off:** Ignoring `plugins[].source` means the `project` field on `MarketplacePlugin` loses its current meaning (today it is the repo slug from `parseGitURL`). We keep populating it from the repo slug to minimize frontend churn; noted as deferred work.
- **Trade-off:** `FetchAll` no longer surfaces network errors at all — they only appear on Refresh. If a user never clicks Refresh, they may not realize a cached marketplace is months out of date. Acceptable for v1; a "last refreshed" timestamp on each section is a cheap follow-up.
- **Trade-off:** Backend tests now require the `git` binary on `PATH` because the cache-refresh tests exercise real git operations against a local bare-repo fixture (task 8.3). Previously the HTTP path could be tested entirely with `httptest.Server`. The new dependency is acceptable on dev machines and on most CI images; document it in the project README before this change archives.

## Migration Plan

No data migration. No format change to `~/.redshell/marketplace.json`. The new cache directory `~/.redshell/.cache/` does not exist on upgrade — it is populated lazily.

For users with marketplaces already registered before this change ships:
1. On first `RefreshAll` (or first `FetchAll` after upgrade), every registered marketplace will hit the cache-miss path. The chosen recovery (auto-reclone on next Refresh) covers this — Browse Plugins will show one round of "cache missing; click Refresh" until the user clicks the button. **Alternative considered:** auto-reclone in a one-shot upgrade hook on `marketplace.NewService()`. Rejected — invisible startup network I/O, hard to surface failures.
2. Removed code: `fetchFileFromGitHub`, `fetchFileFromGitLab`, the `ghToken`/`glToken` parameters threaded through `fetchForProvider`, and any test fixtures that mocked the HTTP servers. Replace tests with cache-fixture-based tests (write a fake cache directory containing manifest files; assert the read path returns expected `MarketplacePlugin`s).
3. `provider.Service.GetTokens()` is no longer called from the plugin browse flow but remains in the codebase for any future consumer. No removal.

Rollback: revert the change set; no on-disk cleanup required (stale `~/.redshell/.cache/` is harmless and ignored by the previous code path).

Implementation order, once this proposal is approved:

1. Confirm `browse-plugins-by-marketplace` is archived (already done; precondition pinned).
2. Reset the previously-implemented HTTP API code paths (revert `fetchFileFromGitHub`/`fetchFileFromGitLab`, related tests).
3. Add cache-key helper, per-cache mutex, and rewrite `marketplace.Service.Add`/`Remove` to manage the cache.
4. Add `Refresh`/`RefreshAll` and rewrite `plugin.Service.fetchForProvider` as a cache read.
5. Frontend Refresh button + adjusted error rendering.
6. Tests against fixture cache directories. Manual verification.
7. `openspec validate fetch-plugins-per-provider-manifest` and archive.

## Open Questions

- `parseGitURL` currently does not know how to derive owner/repo for self-hosted GitLab under `/groups/<group>/<subgroup>/<repo>`. Not introduced by this change, but we will hit it the moment a user registers a subgrouped repo. Out of scope; flag for follow-up.
- Should a marketplace missing its claude manifest show no Claude tab (smarter UX) or an empty/errored Claude tab (simpler)? This design goes with the simpler option for now.
- Should the Refresh button be per-marketplace as well as global? Deferred — one global Refresh is sufficient for v1.
- Should we track and display a "last refreshed at" timestamp per cache? Cheap to add; deferred to follow-up.
