# Plugins UX Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace per-marketplace provider tabs with a single deduplicated plugin list showing provider badges, and allow reinstalling already-installed plugins.

**Architecture:** A `MergedPlugin` type is defined in the plugin store, grouping same-name plugins from different providers into one entry with `providers[]`, `sourcePlugins{}`, and `installedProviders[]`. A `mergedPlugins` computed builds this structure reactively from existing `plugins` and `installedPlugins` refs. `PluginCard` is updated to accept `MergedPlugin` and render provider badges. `BrowsePluginsView` removes all tab logic and uses the merged data.

**Tech Stack:** Vue 3 (Composition API), Pinia 3, TypeScript, DaisyUI 5, Vitest 4, @vue/test-utils 2

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `frontend/src/stores/plugin.ts` | Modify | Export `MergedPlugin` interface; add `mergedPlugins` + `mergedPluginsByMarketplace` computed; expose in return |
| `frontend/src/stores/__tests__/plugin.test.ts` | Create | Unit tests for `mergedPlugins` computed |
| `frontend/src/components/ui/PluginCard.vue` | Modify | Accept `MergedPlugin` prop; render provider badges; remove installed-disabled logic |
| `frontend/src/views/BrowsePluginsView.vue` | Modify | Remove `activeTabs` + tab buttons; use `mergedPluginsByMarketplace`; update install dispatch |

---

### Task 1: Add MergedPlugin type and mergedPlugins computed to plugin store

**Files:**
- Modify: `frontend/src/stores/plugin.ts`
- Create: `frontend/src/stores/__tests__/plugin.test.ts`

- [ ] **Step 1: Write failing tests for mergedPlugins computed**

Create `frontend/src/stores/__tests__/plugin.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePluginStore } from '../plugin'

vi.mock('../../../wailsjs/go/app/PluginApp', () => ({
  FetchAll: vi.fn(),
  Install: vi.fn(),
  ListInstalled: vi.fn(),
  Uninstall: vi.fn(),
}))

vi.mock('../../../wailsjs/go/app/MarketplaceApp', () => ({
  Refresh: vi.fn(),
}))

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
}))

const makePlugin = (name: string, provider: string, marketplace = 'mkt1', marketplaceName = 'My Market') => ({
  name,
  project: `owner/${name}`,
  marketplace,
  marketplaceName,
  installName: `${name}@${marketplaceName}`,
  description: '',
  provider,
})

const makeInstalled = (name: string, provider: string, marketplaceName = 'My Market') => ({
  displayName: name,
  uninstallName: `${name}@${marketplaceName}`,
  provider,
  marketplaceName,
})

describe('usePluginStore - mergedPlugins', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('merges same-name plugins from different providers into one entry', () => {
    const store = usePluginStore()
    store.plugins = [makePlugin('my-plugin', 'claude'), makePlugin('my-plugin', 'copilot')]
    expect(store.mergedPlugins).toHaveLength(1)
    expect(store.mergedPlugins[0].providers).toEqual(['claude', 'copilot'])
  })

  it('keeps separate entries for different plugin names', () => {
    const store = usePluginStore()
    store.plugins = [makePlugin('plugin-a', 'claude'), makePlugin('plugin-b', 'claude')]
    expect(store.mergedPlugins).toHaveLength(2)
  })

  it('keeps separate entries for same name in different marketplaces', () => {
    const store = usePluginStore()
    store.plugins = [
      makePlugin('my-plugin', 'claude', 'mkt1', 'Market One'),
      makePlugin('my-plugin', 'claude', 'mkt2', 'Market Two'),
    ]
    expect(store.mergedPlugins).toHaveLength(2)
  })

  it('populates installedProviders from installedPlugins', () => {
    const store = usePluginStore()
    store.plugins = [makePlugin('my-plugin', 'claude'), makePlugin('my-plugin', 'copilot')]
    store.installedPlugins = [makeInstalled('my-plugin', 'claude')]
    expect(store.mergedPlugins[0].installedProviders).toEqual(['claude'])
  })

  it('sourcePlugins maps provider to the original MarketplacePlugin entry', () => {
    const store = usePluginStore()
    const claudePlugin = makePlugin('my-plugin', 'claude')
    store.plugins = [claudePlugin]
    expect(store.mergedPlugins[0].sourcePlugins['claude']).toEqual(claudePlugin)
  })
})
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd frontend && npx vitest run src/stores/__tests__/plugin.test.ts
```

Expected: FAIL — `store.mergedPlugins` is `undefined`

- [ ] **Step 3: Add MergedPlugin interface (before the store export)**

In `frontend/src/stores/plugin.ts`, add after line 8 (`export type MarketplaceErrorEntry = ...`):

```typescript
export interface MergedPlugin {
  name: string
  project: string
  marketplace: string
  marketplaceName: string
  description?: string
  providers: string[]
  sourcePlugins: Record<string, plugin.MarketplacePlugin>
  installedProviders: string[]
}
```

- [ ] **Step 4: Add mergedPlugins and mergedPluginsByMarketplace computed inside the store**

In `frontend/src/stores/plugin.ts`, add after the `errorsByMarketplace` computed (after line 51):

```typescript
const mergedPlugins = computed<MergedPlugin[]>(() => {
  const map = new Map<string, MergedPlugin>()
  for (const p of plugins.value) {
    const key = `${p.name}@${p.marketplace}`
    if (!map.has(key)) {
      map.set(key, {
        name: p.name,
        project: p.project,
        marketplace: p.marketplace,
        marketplaceName: p.marketplaceName,
        description: p.description,
        providers: [],
        sourcePlugins: {},
        installedProviders: [],
      })
    }
    const entry = map.get(key)!
    entry.providers.push(p.provider)
    entry.sourcePlugins[p.provider] = p
  }
  for (const entry of map.values()) {
    entry.installedProviders = entry.providers.filter((prov) =>
      installedPlugins.value.some(
        (ip) => ip.provider === prov && ip.uninstallName === entry.sourcePlugins[prov]?.installName,
      ),
    )
  }
  return Array.from(map.values())
})

const mergedPluginsByMarketplace = computed<Record<string, MergedPlugin[]>>(() => {
  const grouped: Record<string, MergedPlugin[]> = {}
  for (const p of mergedPlugins.value) {
    if (!grouped[p.marketplace]) grouped[p.marketplace] = []
    grouped[p.marketplace]!.push(p)
  }
  return grouped
})
```

- [ ] **Step 5: Expose the new computed in the store return**

In `frontend/src/stores/plugin.ts`, add `mergedPlugins` and `mergedPluginsByMarketplace` to the `return {}` block (alongside `pluginsByMarketplace`):

```typescript
return {
  plugins,
  installedPlugins,
  selected,
  loading,
  refreshing,
  installing,
  installLog,
  fetchErrors,
  refreshWarnings,
  error,
  pluginsByMarketplace,
  mergedPlugins,
  mergedPluginsByMarketplace,
  errorsByMarketplace,
  fetchAll,
  refreshAll,
  fetchInstalled,
  toggleSelect,
  clearSelection,
  installSelected,
  uninstall,
}
```

- [ ] **Step 6: Run tests to confirm they pass**

```bash
cd frontend && npx vitest run src/stores/__tests__/plugin.test.ts
```

Expected: PASS — 5 tests passing, 0 failing

- [ ] **Step 7: Commit**

```bash
git add frontend/src/stores/plugin.ts frontend/src/stores/__tests__/plugin.test.ts
git commit -m "feat(store): add MergedPlugin type and mergedPlugins computed"
```

---

### Task 2: Update PluginCard to use MergedPlugin with provider badges

**Files:**
- Modify: `frontend/src/components/ui/PluginCard.vue`

- [ ] **Step 1: Replace PluginCard.vue with new implementation**

Replace the full content of `frontend/src/components/ui/PluginCard.vue`:

```vue
<script setup lang="ts">
import type { MergedPlugin } from '@/stores/plugin'

const props = defineProps<{
  plugin: MergedPlugin
  selected: boolean
}>()

const emit = defineEmits<{
  toggle: [key: string]
}>()

function handleClick() {
  emit('toggle', `${props.plugin.name}@${props.plugin.marketplace}`)
}

const providerLabels: Record<string, string> = {
  claude: 'Claude',
  copilot: 'Copilot',
}
</script>

<template>
  <div
    class="flex items-center gap-3 px-4 py-3 rounded-lg bg-base-200 cursor-pointer transition-all hover:bg-base-300"
    :class="{ 'ring-2 ring-primary': selected }"
    @click="handleClick"
  >
    <div class="shrink-0">
      <div
        v-if="selected"
        class="w-5 h-5 rounded-full bg-primary flex items-center justify-center"
      >
        <span class="text-xs text-primary-content font-bold">✓</span>
      </div>
      <div v-else class="w-5 h-5 rounded-full border-2 border-base-content/30" />
    </div>

    <div class="flex-1 min-w-0">
      <h3 class="font-semibold text-base">{{ plugin.name }}</h3>
      <div class="flex gap-1 mt-1">
        <span
          v-for="prov in plugin.providers"
          :key="prov"
          class="badge badge-sm"
          :class="plugin.installedProviders.includes(prov) ? 'badge-success' : 'badge-ghost opacity-50'"
        >
          {{ providerLabels[prov] ?? prov }}
        </span>
      </div>
      <p v-if="plugin.description" class="text-sm text-base-content/70 mt-0.5">
        {{ plugin.description }}
      </p>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ui/PluginCard.vue
git commit -m "feat(ui): update PluginCard to MergedPlugin with provider badges"
```

---

### Task 3: Update BrowsePluginsView to remove provider tabs

**Files:**
- Modify: `frontend/src/views/BrowsePluginsView.vue`

- [ ] **Step 1: Replace BrowsePluginsView.vue with new implementation**

Replace the full content of `frontend/src/views/BrowsePluginsView.vue`:

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import DefaultLayout from '@/layouts/DefaultLayout.vue'
import PluginCard from '@/components/ui/PluginCard.vue'
import AppButton from '@/components/ui/AppButton.vue'
import { usePluginStore } from '@/stores/plugin'
import { useMarketplaceStore } from '@/stores/marketplace'
import { usePageTitle } from '@/composables/usePageTitle'
import type { marketplace } from '../../wailsjs/go/models'

usePageTitle('Browse Plugins')

const store = usePluginStore()
const marketplaceStore = useMarketplaceStore()
const showInstallModal = ref(false)
const targetProviders = ref<string[]>(['claude'])
const installError = ref<string | null>(null)

onMounted(() => {
  marketplaceStore.fetchList()
  store.fetchAll()
  store.fetchInstalled('claude')
  store.fetchInstalled('copilot')
})

const selectedMergedPlugins = computed(() =>
  store.mergedPlugins.filter((mp) => store.selected.has(`${mp.name}@${mp.marketplace}`)),
)

function marketplaceDisplayName(m: marketplace.Marketplace): string {
  const names = m.name ? Object.values(m.name).filter(Boolean) : []
  return names[0] ?? m.id
}

function pluginsFor(id: string) {
  return store.mergedPluginsByMarketplace[id] ?? []
}

function errorsFor(id: string) {
  return store.errorsByMarketplace[id] ?? []
}

function refreshWarningFor(id: string): string | null {
  return store.refreshWarnings[id] ?? null
}

async function handleRefresh() {
  await store.refreshAll()
  await store.fetchAll()
}

async function handleInstall() {
  installError.value = null
  try {
    for (const prov of targetProviders.value) {
      const pluginsToInstall = selectedMergedPlugins.value
        .filter((mp) => mp.providers.includes(prov))
        .map((mp) => mp.sourcePlugins[prov])
      if (pluginsToInstall.length > 0) {
        await store.installSelected(prov, pluginsToInstall)
      }
    }
    showInstallModal.value = false
  } catch (e) {
    installError.value = String(e)
  }
}
</script>

<template>
  <DefaultLayout>
    <div class="max-w-5xl mx-auto">
      <div class="flex items-center justify-between mb-4">
        <h1 class="text-2xl font-bold">Browse Plugins</h1>
        <div class="flex items-center gap-2">
          <AppButton
            variant="ghost"
            :loading="store.refreshing"
            :disabled="store.refreshing || store.loading"
            @click="handleRefresh"
          >
            Refresh
          </AppButton>
          <AppButton
            v-if="store.selected.size > 0"
            @click="showInstallModal = true"
          >
            Install ({{ store.selected.size }})
          </AppButton>
        </div>
      </div>

      <div v-if="store.loading" class="flex justify-center py-12">
        <span class="loading loading-spinner loading-lg" />
      </div>

      <div
        v-else-if="marketplaceStore.marketplaces.length === 0"
        class="text-center py-16 opacity-50"
      >
        <p class="text-lg">No marketplaces registered.</p>
        <RouterLink to="/marketplaces" class="link link-primary text-sm">
          Add a marketplace
        </RouterLink>
      </div>

      <div v-else class="space-y-4">
        <details
          v-for="m in marketplaceStore.marketplaces"
          :key="m.id"
          open
          class="collapse collapse-arrow bg-base-200"
        >
          <summary class="collapse-title text-lg font-semibold cursor-pointer">
            {{ marketplaceDisplayName(m) }}
          </summary>

          <div class="collapse-content space-y-3">
            <div
              v-if="refreshWarningFor(m.id)"
              class="alert alert-warning text-xs py-2"
            >
              <span>Refresh failed: {{ refreshWarningFor(m.id) }}</span>
            </div>

            <template v-if="errorsFor(m.id).length > 0">
              <p
                v-for="err in errorsFor(m.id)"
                :key="err.provider + err.message"
                class="text-xs text-warning"
              >
                {{ err.provider ? `[${err.provider}] ` : '' }}{{ err.message }}
              </p>
            </template>
            <template v-else>
              <div v-if="pluginsFor(m.id).length > 0" class="space-y-2">
                <PluginCard
                  v-for="p in pluginsFor(m.id)"
                  :key="`${p.name}@${p.marketplace}`"
                  :plugin="p"
                  :selected="store.selected.has(`${p.name}@${p.marketplace}`)"
                  @toggle="store.toggleSelect"
                />
              </div>
              <p v-else class="text-sm opacity-60">
                No plugins available in this marketplace.
              </p>
            </template>
          </div>
        </details>
      </div>
    </div>

    <!-- Install Confirmation Modal -->
    <dialog :open="showInstallModal" class="modal modal-bottom sm:modal-middle">
      <div class="modal-box max-w-md">
        <h3 class="font-bold text-lg mb-3">Install Plugins</h3>

        <div class="mb-3">
          <label class="label"><span class="label-text">Install to providers</span></label>
          <div class="space-y-2">
            <label class="flex items-center gap-3 cursor-pointer">
              <input type="checkbox" class="checkbox checkbox-primary" value="claude" v-model="targetProviders" />
              <div>
                <span class="font-medium">Claude Code</span>
                <span class="block text-xs text-base-content/50">~/.claude/plugins/</span>
              </div>
            </label>
            <label class="flex items-center gap-3 cursor-pointer">
              <input type="checkbox" class="checkbox checkbox-primary" value="copilot" v-model="targetProviders" />
              <div>
                <span class="font-medium">GitHub Copilot</span>
                <span class="block text-xs text-base-content/50">~/.copilot/plugins/</span>
              </div>
            </label>
          </div>
        </div>

        <div class="mb-3">
          <p class="text-sm font-medium mb-2">Selected plugins ({{ selectedMergedPlugins.length }}):</p>
          <ul class="space-y-1 max-h-40 overflow-auto">
            <li
              v-for="p in selectedMergedPlugins"
              :key="`${p.name}@${p.marketplace}`"
              class="text-sm opacity-70"
            >
              • {{ p.name }} ({{ p.marketplaceName }})
            </li>
          </ul>
        </div>

        <div v-if="store.installing" class="mb-3">
          <div class="bg-base-300 rounded p-2 max-h-32 overflow-auto text-xs font-mono space-y-0.5">
            <div v-for="(line, i) in store.installLog" :key="i">{{ line }}</div>
          </div>
        </div>

        <p v-if="installError" class="text-error text-sm mb-2">{{ installError }}</p>

        <div class="modal-action">
          <AppButton variant="ghost" :disabled="store.installing" @click="showInstallModal = false">Cancel</AppButton>
          <AppButton :loading="store.installing" :disabled="targetProviders.length === 0" @click="handleInstall">Install</AppButton>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop" @submit.prevent="showInstallModal = false">
        <button>close</button>
      </form>
    </dialog>
  </DefaultLayout>
</template>
```

- [ ] **Step 2: Verify TypeScript type-check passes**

```bash
cd frontend && npm run type-check
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/BrowsePluginsView.vue
git commit -m "feat(ui): remove provider tabs, show plugins with provider badges"
```
