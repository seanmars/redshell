<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import DefaultLayout from '@/layouts/DefaultLayout.vue';
import PageContainer from '@/layouts/PageContainer.vue';
import AppTabs from '@/components/ui/AppTabs.vue';
import AppTab from '@/components/ui/AppTab.vue';
import AppEmptyState from '@/components/ui/AppEmptyState.vue';
import HooksAgentPane from '@/components/hooks/HooksAgentPane.vue';
import { useAgentStore } from '@/stores/agent';
import { useAgentSetupStore } from '@/stores/agentSetup';
import { useHooksStore } from '@/stores/hooks';
import { usePageTitle } from '@/composables/usePageTitle';

usePageTitle('Hooks');

const agentStore = useAgentStore();
const setupStore = useAgentSetupStore();
const store = useHooksStore();

const enabledAgents = computed(() =>
  agentStore.agents.filter((agent) => setupStore.isAgentEnabled(agent.id)),
);

const activeAgent = ref('');

onMounted(async () => {
  await setupStore.ensureLoaded();
  await agentStore.fetchAgents();
});

watch(
  enabledAgents,
  (agents) => {
    if (agents.length === 0) {
      activeAgent.value = '';
      store.clearSelection();
      return;
    }
    if (!agents.some((a) => a.id === activeAgent.value)) {
      activeAgent.value = agents[0]!.id;
    }
  },
  { immediate: true },
);

watch(
  activeAgent,
  async (agt) => {
    if (!agt) return;
    store.setActiveAgent(agt);
    if (!store.listings[agt]) {
      await store.fetchHooks(agt);
    }
  },
  { immediate: true },
);

function handleTabChange(id: string) {
  if (enabledAgents.value.some((a) => a.id === id)) {
    activeAgent.value = id;
  }
}

function handleSelect(hookID: string) {
  store.selectHook(activeAgent.value, hookID);
}

const showTabs = computed(() => enabledAgents.value.length > 1);
</script>

<template>
  <DefaultLayout>
    <PageContainer title="Hooks" max-width="max-w-7xl" fill>
      <AppEmptyState
        v-if="enabledAgents.length === 0"
        icon="hooks"
        title="No enabled agents"
        description="Enable at least one agent in Settings to view hooks."
      >
        <RouterLink to="/settings?tab=agents" class="link link-primary text-sm">
          Open agent settings →
        </RouterLink>
      </AppEmptyState>

      <template v-else>
        <AppTabs
          v-if="showTabs"
          :active="activeAgent"
          variant="lift"
          fill
          class="flex-1 min-h-0"
          @update:active="handleTabChange"
        >
          <AppTab
            v-for="agent in enabledAgents"
            :id="agent.id"
            :key="agent.id"
            :label="agent.label"
            class="flex-1 min-h-0 flex flex-col"
          >
            <HooksAgentPane
              :agent-id="agent.id"
              :listing="store.listings[agent.id] ?? null"
              :loading="store.loading[agent.id] ?? false"
              :error="store.errors[agent.id] ?? ''"
              :selected-hook-id="store.currentAgent === agent.id ? store.currentHookID : ''"
              :selected-hook="store.currentAgent === agent.id ? store.currentHook : null"
              :selected-source="store.currentAgent === agent.id ? store.currentSource : null"
              class="mt-2"
              @select="handleSelect"
            />
          </AppTab>
        </AppTabs>

        <HooksAgentPane
          v-else
          :agent-id="activeAgent"
          :listing="store.listings[activeAgent] ?? null"
          :loading="store.loading[activeAgent] ?? false"
          :error="store.errors[activeAgent] ?? ''"
          :selected-hook-id="store.currentHookID"
          :selected-hook="store.currentHook"
          :selected-source="store.currentSource"
          class="flex-1 min-h-0"
          @select="handleSelect"
        />
      </template>
    </PageContainer>
  </DefaultLayout>
</template>
