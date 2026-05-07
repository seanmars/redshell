<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import DefaultLayout from '@/layouts/DefaultLayout.vue';
import PageContainer from '@/layouts/PageContainer.vue';
import PluginCard from '@/components/plugin/PluginCard.vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppModal from '@/components/ui/AppModal.vue';
import AppAlert from '@/components/ui/AppAlert.vue';
import AppCollapse from '@/components/ui/AppCollapse.vue';
import AppCheckbox from '@/components/ui/AppCheckbox.vue';
import AppIcon from '@/components/ui/AppIcon.vue';
import AppSkeleton from '@/components/ui/AppSkeleton.vue';
import AppEmptyState from '@/components/ui/AppEmptyState.vue';
import { useAgentStore } from '@/stores/agent';
import { useAgentSetupStore } from '@/stores/agentSetup';
import { usePluginStore } from '@/stores/plugin';
import { useMarketplaceStore } from '@/stores/marketplace';
import { usePageTitle } from '@/composables/usePageTitle';
import { usePluginInstaller } from '@/composables/usePluginInstaller';
import type { marketplace, plugin } from '@wailsjs/go/models';
import type { MergedPlugin } from '@/stores/plugin';

usePageTitle('Browse Plugins');

const agentStore = useAgentStore();
const setupStore = useAgentSetupStore();
const store = usePluginStore();
const marketplaceStore = useMarketplaceStore();
const enabledAgents = computed(() =>
  agentStore.agents.filter((agent) => setupStore.isAgentEnabled(agent.id)),
);
const {
  showInstallModal,
  targetAgents,
  installError,
  selectedMergedPlugins,
  availableAgents,
  handleInstall,
  closeModal,
} = usePluginInstaller(
  store,
  computed(() => enabledAgents.value.map((agent) => agent.id)),
);

const filteredPluginsByMarketplace = computed<Record<string, MergedPlugin[]>>(() => {
  const grouped: Record<string, MergedPlugin[]> = {};

  for (const merged of store.mergedPlugins) {
    const allowedAgents = merged.agents.filter((agentID) => setupStore.isAgentEnabled(agentID));
    if (allowedAgents.length === 0) {
      continue;
    }

    const sourcePlugins: Record<string, plugin.MarketplacePlugin> = {};
    for (const agentID of allowedAgents) {
      const source = merged.sourcePlugins[agentID];
      if (source) {
        sourcePlugins[agentID] = source;
      }
    }

    const next: MergedPlugin = {
      ...merged,
      agents: allowedAgents,
      installedAgents: merged.installedAgents.filter((agentID) =>
        setupStore.isAgentEnabled(agentID),
      ),
      sourcePlugins,
    };

    if (!grouped[next.marketplace]) {
      grouped[next.marketplace] = [];
    }
    grouped[next.marketplace]!.push(next);
  }

  return grouped;
});

onMounted(async () => {
  await setupStore.ensureLoaded();
  await Promise.all([
    agentStore.fetchAgents(),
    marketplaceStore.fetchList(),
    store.fetchAll(),
    ...setupStore.enabledAgents.map((agentID) => store.fetchInstalled(agentID)),
  ]);
  store.clearSelection();
});

function marketplaceDisplayName(m: marketplace.Marketplace): string {
  const names = m.name ? Object.values(m.name).filter(Boolean) : [];
  return names[0] ?? m.id;
}

function pluginsFor(id: string) {
  return filteredPluginsByMarketplace.value[id] ?? [];
}

function errorsFor(id: string) {
  return store.errorsByMarketplace[id] ?? [];
}

function refreshWarningFor(id: string): string | null {
  return store.refreshWarnings[id] ?? null;
}

async function handleRefresh() {
  await store.refreshAll();
  await store.fetchAll();
}
</script>

<template>
  <DefaultLayout>
    <PageContainer title="Browse Plugins">
      <template #actions>
        <AppButton
          variant="primary"
          v-if="store.selected.size > 0"
          @click="showInstallModal = true"
        >
          Install ({{ store.selected.size }})
        </AppButton>

        <AppButton
          variant="secondary"
          :loading="store.refreshing"
          :disabled="store.refreshing || store.loading"
          @click="handleRefresh"
        >
          <AppIcon v-if="!store.refreshing" name="refresh" size="sm" />
          Refresh
        </AppButton>
      </template>

      <div v-if="store.loading" class="space-y-2" aria-busy="true" aria-live="polite">
        <div
          v-for="i in 5"
          :key="i"
          class="flex items-center gap-3 px-4 py-3 rounded-lg bg-base-200 ring-1 ring-base-content/5"
        >
          <AppSkeleton shape="circle" height="h-5" width="w-5" />
          <div class="flex-1 space-y-2">
            <AppSkeleton height="h-4" width="w-1/3" />
            <AppSkeleton height="h-3" width="w-2/3" />
          </div>
        </div>
      </div>

      <AppEmptyState
        v-else-if="marketplaceStore.marketplaces.length === 0"
        icon="installed"
        title="No marketplaces yet"
        description="Add a git repository that publishes a Claude Code or Copilot plugin marketplace to start browsing."
      >
        <RouterLink to="/settings?tab=marketplaces" class="link link-primary text-sm">
          Open marketplace settings →
        </RouterLink>
      </AppEmptyState>

      <div v-else class="space-y-4">
        <AppCollapse
          v-for="m in marketplaceStore.marketplaces"
          :key="m.id"
          :title="marketplaceDisplayName(m)"
          :default-open="true"
        >
          <div class="space-y-3">
            <AppAlert v-if="refreshWarningFor(m.id)" type="warning">
              <span class="text-xs">Refresh failed: {{ refreshWarningFor(m.id) }}</span>
            </AppAlert>

            <template v-if="errorsFor(m.id).length > 0">
              <p
                v-for="err in errorsFor(m.id)"
                :key="err.agent + err.message"
                class="text-xs text-warning"
              >
                {{ err.agent ? `[${err.agent}] ` : '' }}{{ err.message }}
              </p>
            </template>

            <div v-if="pluginsFor(m.id).length > 0" class="space-y-2">
              <PluginCard
                v-for="p in pluginsFor(m.id)"
                :key="`${p.name}@${p.marketplace}`"
                :plugin="p"
                :selected="store.selected.has(`${p.name}@${p.marketplace}`)"
                @toggle="store.toggleSelect"
              />
            </div>
            <p v-else-if="errorsFor(m.id).length === 0" class="text-sm opacity-60">
              No plugins available in this marketplace.
            </p>
          </div>
        </AppCollapse>
      </div>
    </PageContainer>

    <AppModal :is-open="showInstallModal" size="md" @close="closeModal">
      <template #header>Install Plugins</template>

      <div class="mb-3">
        <label class="label"><span class="label-text">Install to agents</span></label>
        <div class="space-y-2">
          <AppCheckbox
            v-for="agent in enabledAgents"
            :key="agent.id"
            v-model="targetAgents"
            :value="agent.id"
            :disabled="!availableAgents.has(agent.id)"
          >
            <div>
              <span class="font-medium">{{ agent.label }}</span>
              <span class="block text-xs text-base-content/50">
                {{
                  availableAgents.has(agent.id)
                    ? 'Available for the selected plugins'
                    : 'Selected plugins do not support this agent'
                }}
              </span>
            </div>
          </AppCheckbox>
        </div>
      </div>

      <div class="mb-3">
        <p class="text-sm font-medium mb-2">
          Selected plugins ({{ selectedMergedPlugins.length }}):
        </p>
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

      <template #actions>
        <AppButton variant="ghost" :disabled="store.installing" @click="closeModal">
          Cancel
        </AppButton>
        <AppButton
          :loading="store.installing"
          :disabled="targetAgents.length === 0"
          @click="handleInstall"
        >
          Install
        </AppButton>
      </template>
    </AppModal>
  </DefaultLayout>
</template>
