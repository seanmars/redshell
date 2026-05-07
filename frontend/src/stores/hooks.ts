import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { ListHooks } from '@wailsjs/go/app/HooksApp';
import type { hooks } from '@wailsjs/go/models';

export const useHooksStore = defineStore('hooks', () => {
  const listings = ref<Record<string, hooks.Listing>>({});
  const loading = ref<Record<string, boolean>>({});
  const errors = ref<Record<string, string>>({});

  const currentAgent = ref<string>('');
  const currentHookID = ref<string>('');

  const currentListing = computed<hooks.Listing | null>(() => {
    if (!currentAgent.value) return null;
    return listings.value[currentAgent.value] ?? null;
  });

  const currentHook = computed<hooks.Hook | null>(() => {
    const list = currentListing.value;
    if (!list || !currentHookID.value) return null;
    return (list.hooks ?? []).find((h) => h.id === currentHookID.value) ?? null;
  });

  const currentSource = computed<hooks.Source | null>(() => {
    const hook = currentHook.value;
    const list = currentListing.value;
    if (!hook || !list) return null;
    return (list.sources ?? []).find((s) => s.id === hook.sourceID) ?? null;
  });

  async function fetchHooks(agentID: string) {
    loading.value = { ...loading.value, [agentID]: true };
    errors.value = { ...errors.value, [agentID]: '' };
    try {
      const result = await ListHooks(agentID, { workspace: '' });
      listings.value = { ...listings.value, [agentID]: result };
    } catch (e) {
      errors.value = { ...errors.value, [agentID]: String(e) };
    } finally {
      loading.value = { ...loading.value, [agentID]: false };
    }
  }

  function selectHook(agentID: string, hookID: string) {
    currentAgent.value = agentID;
    currentHookID.value = hookID;
  }

  function clearSelection() {
    currentAgent.value = '';
    currentHookID.value = '';
  }

  function setActiveAgent(agentID: string) {
    if (currentAgent.value !== agentID) {
      currentHookID.value = '';
    }
    currentAgent.value = agentID;
  }

  return {
    listings,
    loading,
    errors,
    currentAgent,
    currentHookID,
    currentListing,
    currentHook,
    currentSource,
    fetchHooks,
    selectHook,
    clearSelection,
    setActiveAgent,
  };
});
