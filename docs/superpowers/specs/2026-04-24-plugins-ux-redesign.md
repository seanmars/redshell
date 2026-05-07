# Plugins UX Redesign

**Date**: 2026-04-24
**Scope**: BrowsePluginsView, PluginCard, plugin store

## Goals

1. Allow reinstalling already-installed plugins (remove installed=disabled restriction)
2. Remove provider tabs from BrowsePluginsView; replace with provider badges on each plugin card
3. A badge lights up when the plugin is installed for that provider

## Out of Scope

- InstalledPluginsView (no changes)
- Backend / Go service (no changes)
- Install modal flow (kept as-is: select → Install button → modal → pick provider)

---

## Data Model

### New Frontend Type: `MergedPlugin`

```typescript
interface MergedPlugin {
  name: string
  project: string
  marketplace: string
  marketplaceName: string
  description?: string
  providers: string[]                              // e.g. ["claude", "copilot"]
  sourcePlugins: Record<string, MarketplacePlugin> // provider → original entry
  installedProviders: string[]                     // providers with this plugin installed
}
```

**Merge key**: `${name}@${marketplace}` — same plugin name in the same marketplace is treated as one entry regardless of provider.

**installedProviders**: derived by cross-referencing `store.installedPlugins` after each `fetchInstalled()` call.

**Install dispatch**: when the user selects a provider in the modal, pick `sourcePlugins[prov]` from each selected `MergedPlugin` to get the original `MarketplacePlugin` entries expected by the existing `install()` backend call.

---

## Store Changes (`stores/plugin.ts`)

### New Computed Properties

```typescript
// Deduplicated plugin list
const mergedPlugins = computed<MergedPlugin[]>(() => {
  const map = new Map<string, MergedPlugin>()
  for (const p of plugins.value) {
    const key = `${p.name}@${p.marketplace}`
    if (!map.has(key)) {
      map.set(key, {
        name: p.name, project: p.project,
        marketplace: p.marketplace, marketplaceName: p.marketplaceName,
        description: p.description,
        providers: [], sourcePlugins: {}, installedProviders: []
      })
    }
    const entry = map.get(key)!
    entry.providers.push(p.provider)
    entry.sourcePlugins[p.provider] = p
  }
  return Array.from(map.values())
})

// Grouped by marketplace ID
const mergedPluginsByMarketplace = computed(() => {
  const groups = new Map<string, MergedPlugin[]>()
  for (const p of mergedPlugins.value) {
    const list = groups.get(p.marketplace) ?? []
    list.push(p)
    groups.set(p.marketplace, list)
  }
  return groups
})
```

### installedProviders Update

After every `fetchInstalled(prov)` call, recompute `installedProviders` on each `MergedPlugin`. Matching strategy: an `InstalledPlugin` is considered a match if `installedPlugin.provider === prov` AND `installedPlugin.marketplaceName === mergedPlugin.marketplaceName` AND the plugin name is contained in `installedPlugin.uninstallName` or `installedPlugin.displayName`.

> **Implementation note**: Verify exact field values at runtime — `uninstallName` format (e.g. `name@marketplaceName`) may allow direct string comparison against `MergedPlugin.name`.

### selected Set

Key format changes from `installName` to `${name}@${marketplace}` to align with `MergedPlugin`.

### installSelected() Adjustment

```typescript
async function installSelected(prov: string) {
  const pluginsToInstall = [...selected.value]
    .map(key => mergedPlugins.value.find(
      mp => `${mp.name}@${mp.marketplace}` === key
    ))
    .filter(mp => mp?.providers.includes(prov))
    .map(mp => mp!.sourcePlugins[prov])
  // pass to existing Install backend call
}
```

---

## PluginCard Component (`components/ui/PluginCard.vue`)

### Props

```typescript
// Before
props: { plugin: MarketplacePlugin, selected: boolean, installed: boolean }

// After
props: { plugin: MergedPlugin, selected: boolean }
```

### Behavior Changes

- **Remove** `installed` guard in `handleClick()` — all plugins are always clickable
- **Remove** `opacity-60 cursor-default` class binding for installed state
- **Remove** green installed badge on the card

### Provider Badges

Added below the plugin name:

```vue
<div class="flex gap-1 mt-1">
  <span
    v-for="prov in plugin.providers"
    :key="prov"
    class="badge badge-sm"
    :class="plugin.installedProviders.includes(prov)
      ? 'badge-success'
      : 'badge-ghost opacity-50'"
  >{{ prov }}</span>
</div>
```

---

## BrowsePluginsView Changes (`views/BrowsePluginsView.vue`)

### Removals

- `activeTabs` ref
- `getActiveTab()` and `setActiveTab()` functions
- Provider tab `<button>` elements inside each Marketplace collapse

### Data Source

Replace `store.pluginsByMarketplace` with `store.mergedPluginsByMarketplace`.

`pluginsFor(marketplaceId)` returns `MergedPlugin[]` directly — no provider filter needed.

### targetProviders Computed (in Install Modal)

```typescript
const targetProviders = computed(() =>
  [...store.selected]
    .flatMap(key => {
      const mp = store.mergedPlugins.find(
        p => `${p.name}@${p.marketplace}` === key
      )
      return mp?.providers ?? []
    })
    .filter((v, i, a) => a.indexOf(v) === i)
)
```

---

## File Change Summary

| File | Change Type |
|------|-------------|
| `frontend/src/stores/plugin.ts` | Add `mergedPlugins`, `mergedPluginsByMarketplace` computed; adjust `selected` key format and `installSelected()` |
| `frontend/src/components/ui/PluginCard.vue` | Update props to `MergedPlugin`; add provider badges; remove installed-disabled logic |
| `frontend/src/views/BrowsePluginsView.vue` | Remove provider tabs; use merged data source; update `targetProviders` |

No backend changes required.
