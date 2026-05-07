<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import DefaultLayout from '@/layouts/DefaultLayout.vue';
import PageContainer from '@/layouts/PageContainer.vue';
import AppTabs from '@/components/ui/AppTabs.vue';
import AppTab from '@/components/ui/AppTab.vue';
import AppEmptyState from '@/components/ui/AppEmptyState.vue';
import SessionList from '@/components/sessions/SessionList.vue';
import SessionEventList from '@/components/sessions/SessionEventList.vue';
import { useAgentStore } from '@/stores/agent';
import { useAgentSetupStore } from '@/stores/agentSetup';
import { useSessionHistoryStore } from '@/stores/sessionHistory';
import { usePageTitle } from '@/composables/usePageTitle';

usePageTitle('Session History');

const agentStore = useAgentStore();
const setupStore = useAgentSetupStore();
const store = useSessionHistoryStore();

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
  async (agt, prev) => {
    if (!agt) return;
    if (agt !== prev) store.clearSelection();
    if (!store.listings[agt]) {
      await store.fetchListing(agt);
    }
  },
  { immediate: true },
);

function handleTabChange(id: string) {
  if (enabledAgents.value.some((a) => a.id === id)) {
    activeAgent.value = id;
  }
}

function handleSelect(agentID: string, sessionID: string) {
  store.selectSession(agentID, sessionID);
}

const titleSuffix = computed(() => store.displayTitle ?? '');

const showTabs = computed(() => enabledAgents.value.length > 1);
</script>

<template>
  <DefaultLayout>
    <PageContainer title="Session History" :title-suffix="titleSuffix" max-width="max-w-7xl" fill>
      <AppEmptyState
        v-if="enabledAgents.length === 0"
        icon="installed"
        title="No enabled agents"
        description="Enable at least one agent in Settings to view session history."
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
            <div class="grid grid-cols-1 md:grid-cols-[20rem_1fr] gap-3 flex-1 min-h-0 mt-2">
              <div class="bg-base-200/40 rounded-md border border-base-300/60 overflow-hidden">
                <SessionList
                  :listing="store.listings[agent.id]"
                  :loading="store.listingLoading[agent.id] ?? false"
                  :error="store.listingErrors[agent.id] ?? ''"
                  :selected-session-id="
                    store.currentAgent === agent.id ? store.currentSessionID : ''
                  "
                  @select="handleSelect"
                />
              </div>
              <div class="bg-base-100 rounded-md border border-base-300/60 overflow-hidden">
                <AppEmptyState
                  v-if="!store.currentSessionID || store.currentAgent !== agent.id"
                  icon="file"
                  title="Select a session"
                  description="Choose a session from the list to see its events."
                />
                <SessionEventList
                  v-else
                  :events="store.events"
                  :has-more="store.hasMore"
                  :loading="store.eventsLoading"
                  :total="store.total"
                  :skipped-lines="store.skippedLines"
                  @load-more="store.loadNextPage"
                />
              </div>
            </div>
          </AppTab>
        </AppTabs>

        <div v-else class="grid grid-cols-1 md:grid-cols-[20rem_1fr] gap-3 flex-1 min-h-0">
          <div class="bg-base-200/40 rounded-md border border-base-300/60 overflow-hidden">
            <SessionList
              :listing="store.listings[activeAgent]"
              :loading="store.listingLoading[activeAgent] ?? false"
              :error="store.listingErrors[activeAgent] ?? ''"
              :selected-session-id="store.currentSessionID"
              @select="handleSelect"
            />
          </div>
          <div class="bg-base-100 rounded-md border border-base-300/60 overflow-hidden">
            <AppEmptyState
              v-if="!store.currentSessionID"
              icon="file"
              title="Select a session"
              description="Choose a session from the list to see its events."
            />
            <SessionEventList
              v-else
              :events="store.events"
              :has-more="store.hasMore"
              :loading="store.eventsLoading"
              :total="store.total"
              :skipped-lines="store.skippedLines"
              @load-more="store.loadNextPage"
            />
          </div>
        </div>
      </template>
    </PageContainer>
  </DefaultLayout>
</template>
