import { createRouter, createWebHistory } from 'vue-router';
import { pinia } from '@/pinia';
import { useAgentSetupStore } from '@/stores/agentSetup';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/browse',
    },
    {
      path: '/agents',
      redirect: { path: '/settings', query: { tab: 'agents' } },
    },
    {
      path: '/providers',
      redirect: { path: '/settings', query: { tab: 'agents' } },
    },
    {
      path: '/marketplaces',
      redirect: { path: '/settings', query: { tab: 'marketplaces' } },
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/SettingsView.vue'),
    },
    {
      path: '/setup/agents',
      name: 'agent-setup',
      component: () => import('@/views/AgentSetupView.vue'),
    },
    {
      path: '/browse',
      name: 'browse',
      component: () => import('@/views/BrowsePluginsView.vue'),
    },
    {
      path: '/installed',
      name: 'installed',
      component: () => import('@/views/InstalledPluginsView.vue'),
    },
    {
      path: '/sessions',
      name: 'sessions',
      component: () => import('@/views/SessionHistoryView.vue'),
    },
    {
      path: '/hooks',
      name: 'hooks',
      component: () => import('@/views/HooksView.vue'),
    },
  ],
});

router.beforeEach(async (to) => {
  const setupStore = useAgentSetupStore(pinia);
  await setupStore.ensureLoaded();

  if (!setupStore.hasCompletedSetup && to.name !== 'agent-setup') {
    return { name: 'agent-setup' };
  }

  if (setupStore.hasCompletedSetup && to.name === 'agent-setup') {
    return { path: '/browse' };
  }

  return true;
});

export default router;
