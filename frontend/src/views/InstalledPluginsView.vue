<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import DefaultLayout from '@/layouts/DefaultLayout.vue';
import PageContainer from '@/layouts/PageContainer.vue';
import InstalledPluginCard from '@/components/plugin/InstalledPluginCard.vue';
import AppConfirmModal from '@/components/ui/AppConfirmModal.vue';
import AppTabs from '@/components/ui/AppTabs.vue';
import AppTab from '@/components/ui/AppTab.vue';
import AppSkeleton from '@/components/ui/AppSkeleton.vue';
import AppEmptyState from '@/components/ui/AppEmptyState.vue';
import { useAgentStore } from '@/stores/agent';
import { useAgentSetupStore } from '@/stores/agentSetup';
import { usePluginStore } from '@/stores/plugin';
import { useConfirm } from '@/composables/useConfirm';
import { usePageTitle } from '@/composables/usePageTitle';
import { useToast } from '@/composables/useToast';

usePageTitle('Installed Plugins');

const agentStore = useAgentStore();
const setupStore = useAgentSetupStore();
const store = usePluginStore();
const confirm = useConfirm();
const toast = useToast();

const activeAgent = ref('');

const enabledAgents = computed(() =>
  agentStore.agents.filter((agent) => setupStore.isAgentEnabled(agent.id)),
);

onMounted(async () => {
  await setupStore.ensureLoaded();
  await agentStore.fetchAgents();
});

watch(
  enabledAgents,
  (agents) => {
    if (agents.length === 0) {
      activeAgent.value = '';
      return;
    }
    if (!agents.some((agent) => agent.id === activeAgent.value)) {
      activeAgent.value = agents[0]!.id;
    }
  },
  { immediate: true },
);

watch(activeAgent, (agt) => {
  if (agt) {
    store.fetchInstalled(agt);
  }
});

function selectAgent(id: string) {
  if (enabledAgents.value.some((agent) => agent.id === id)) {
    activeAgent.value = id;
  }
}

const shownPlugins = computed(() =>
  store.installedPlugins
    .filter((p) => p.agent === activeAgent.value)
    .sort((a, b) => a.displayName.localeCompare(b.displayName, undefined, { sensitivity: 'base' })),
);

async function handleUninstall(pluginID: string) {
  const ok = await confirm.confirm({
    title: 'Uninstall Plugin',
    message: `Uninstall "${pluginID}" from ${activeAgent.value}?`,
    confirmLabel: 'Uninstall',
  });
  if (!ok) return;
  try {
    await store.uninstall(activeAgent.value, pluginID);
    toast.push({ type: 'success', message: `"${pluginID}" uninstalled successfully.` });
  } catch (e) {
    toast.push({ type: 'error', message: `Uninstall failed: ${e}` });
  }
}

async function handleUpdate(installName: string) {
  try {
    await store.update(activeAgent.value, installName);
    toast.push({ type: 'success', message: `"${installName}" updated successfully.` });
  } catch (e) {
    toast.push({ type: 'error', message: `Update failed: ${e}` });
  }
}
</script>

<template>
  <DefaultLayout>
    <PageContainer title="Installed Plugins">
      <AppEmptyState
        v-if="enabledAgents.length === 0"
        icon="installed"
        title="No enabled agents"
        description="Enable at least one agent in Settings to view installed plugins."
      >
        <RouterLink to="/settings?tab=agents" class="link link-primary text-sm">
          Open agent settings →
        </RouterLink>
      </AppEmptyState>

      <AppTabs v-else :active="activeAgent" variant="lift" @update:active="selectAgent">
        <AppTab v-for="agent in enabledAgents" :id="agent.id" :key="agent.id" :label="agent.label">
          <div v-if="store.loading" class="space-y-2" aria-busy="true" aria-live="polite">
            <div
              v-for="i in 4"
              :key="i"
              class="flex items-center justify-between gap-3 px-4 py-3 rounded-lg bg-base-200 ring-1 ring-base-content/5"
            >
              <div class="flex-1 space-y-2">
                <AppSkeleton height="h-4" width="w-2/5" />
                <AppSkeleton height="h-3" width="w-1/4" />
              </div>
              <AppSkeleton height="h-7" width="w-20" />
            </div>
          </div>

          <AppEmptyState
            v-else-if="shownPlugins.length === 0"
            icon="installed"
            :title="`Nothing installed for ${agent.label} yet`"
            description="Plugins you install from the browse page will show up here."
          >
            <RouterLink to="/browse" class="link link-primary text-sm"> Go to browse → </RouterLink>
          </AppEmptyState>

          <div v-else class="space-y-2">
            <InstalledPluginCard
              v-for="p in shownPlugins"
              :key="p.uninstallName"
              :plugin="p"
              :busy="store.isPluginBusy(p.uninstallName)"
              @uninstall="handleUninstall"
              @update="handleUpdate"
            />
          </div>
        </AppTab>
      </AppTabs>
    </PageContainer>

    <AppConfirmModal
      :is-open="confirm.isOpen.value"
      :title="confirm.options.value.title"
      :message="confirm.options.value.message"
      :confirm-label="confirm.options.value.confirmLabel"
      :cancel-label="confirm.options.value.cancelLabel"
      @confirm="confirm.onConfirm"
      @cancel="confirm.onCancel"
    />
  </DefaultLayout>
</template>
