<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { EventsOff, EventsOn } from '@wailsjs/runtime/runtime';
import DefaultLayout from '@/layouts/DefaultLayout.vue';
import PageContainer from '@/layouts/PageContainer.vue';
import AppTabs from '@/components/ui/AppTabs.vue';
import AppTab from '@/components/ui/AppTab.vue';
import AgentsTab from '@/components/settings/AgentsTab.vue';
import MarketplacesTab from '@/components/settings/MarketplacesTab.vue';
import UpdatesTab from '@/components/settings/UpdatesTab.vue';
import { usePageTitle } from '@/composables/usePageTitle';

usePageTitle('Settings');

const TRAY_OPEN_UPDATES_EVENT = 'tray:open-updates';

type TabId = 'marketplaces' | 'agents' | 'updates';

const VALID_TABS: TabId[] = ['marketplaces', 'agents', 'updates'];
const DEFAULT_TAB: TabId = 'marketplaces';

const route = useRoute();
const router = useRouter();

const activeTab = computed<TabId>(() => {
  const raw = Array.isArray(route.query.tab) ? route.query.tab[0] : route.query.tab;
  return VALID_TABS.includes(raw as TabId) ? (raw as TabId) : DEFAULT_TAB;
});

watch(
  () => route.query.tab,
  (raw) => {
    const value = Array.isArray(raw) ? raw[0] : raw;
    if (!value || !VALID_TABS.includes(value as TabId)) {
      router.replace({ path: '/settings', query: { tab: DEFAULT_TAB } });
    }
  },
  { immediate: true },
);

function selectTab(id: string) {
  if (!VALID_TABS.includes(id as TabId)) return;
  if (activeTab.value === id) return;
  router.replace({ path: '/settings', query: { tab: id } });
}

onMounted(() => {
  EventsOn(TRAY_OPEN_UPDATES_EVENT, () => {
    router.replace({ path: '/settings', query: { tab: 'updates' } });
  });
});

onUnmounted(() => {
  EventsOff(TRAY_OPEN_UPDATES_EVENT);
});
</script>

<template>
  <DefaultLayout>
    <PageContainer title="Settings">
      <AppTabs :active="activeTab" variant="lift" @update:active="selectTab">
        <AppTab id="marketplaces" label="Marketplaces">
          <MarketplacesTab />
        </AppTab>
        <AppTab id="agents" label="Agents">
          <AgentsTab />
        </AppTab>
        <AppTab id="updates" label="Updates">
          <UpdatesTab />
        </AppTab>
      </AppTabs>
    </PageContainer>
  </DefaultLayout>
</template>
