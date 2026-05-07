## 1. Store layer

- [x] 1.1 In `frontend/src/stores/plugin.ts`, add a `pluginsByMarketplace` computed getter that groups `plugins` by `MarketplacePlugin.marketplace` into a `Record<marketplaceID, MarketplacePlugin[]>`.
- [x] 1.2 Add a `errorsByMarketplace` computed getter that parses `fetchErrors` entries of the form `[<marketplaceID>/<provider>] <message>` into a `Record<marketplaceID, Array<{ provider: string; message: string }>>`. Entries that do not match the prefix format fall back into a `__global` bucket.
- [x] 1.3 Export the two new getters from the store `return` block so views can consume them.

## 2. Browse Plugins view

- [x] 2.1 In `frontend/src/views/BrowsePluginsView.vue`, import `useMarketplaceStore` and call `marketplaceStore.fetchList()` in the existing `onMounted` alongside the plugin fetches (fire all in parallel).
- [x] 2.2 Remove the `providerFilter` ref, the `providers` array, the `filteredPlugins` computed, and the `tabs tabs-boxed` markup.
- [x] 2.3 Replace the flat grid with a v-for over `marketplaceStore.marketplaces`. For each marketplace render a section containing a header (display name, falling back to ID) and a `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3` grid of `PluginCard`s sourced from `pluginStore.pluginsByMarketplace[marketplace.id] ?? []`.
- [x] 2.4 Inside each section render inline error text from `pluginStore.errorsByMarketplace[marketplace.id]` when present (one line per provider error). Show a "no plugins available in this marketplace" message when the list is empty and there are no errors.
- [x] 2.5 Replace the current "No plugins found" empty state with a page-level empty state shown only when `marketplaceStore.marketplaces.length === 0`; keep its link to `/marketplaces`.
- [x] 2.6 Delete the bottom-of-page `store.fetchErrors` warning list; errors are now shown inside their marketplace section.
- [x] 2.7 Keep the install button, selection count, and install confirmation modal unchanged (provider selection stays in the modal).

## 3. Specs and proposal bookkeeping

- [x] 3.1 Run `openspec validate browse-plugins-by-marketplace` and resolve any reported issues before archiving.

## 4. Manual verification

- [ ] 4.1 Empty registry: with `~/.redshell/marketplace.json` empty (or removed), open Browse Plugins and confirm a single page-level empty state with the Marketplaces link, and no section rendered.
- [ ] 4.2 Add a marketplace via the Marketplaces page, navigate to Browse Plugins, and confirm its section appears with grouped plugins; remove the marketplace and confirm the section disappears.
- [ ] 4.3 Configure an unreachable or non-plugin GitHub repo as a marketplace and confirm its section renders with the inline error text (instead of silently dropping).
- [ ] 4.4 Verify that plugins targeting both `claude` and `copilot` appear in the same marketplace section without a provider filter, and that the install modal still lets the user pick the target provider.
