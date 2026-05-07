import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import {
  GetAgentSetupState,
  GetEnabledAgents,
  IsAgentEnabled,
  SetEnabledAgents,
} from '@wailsjs/go/app/AgentApp';

export interface AgentSetupState {
  enabledAgents: string[];
  agentSetupCompleted: boolean;
}

export const useAgentSetupStore = defineStore('agent-setup', () => {
  const enabledAgents = ref<string[]>([]);
  const hasCompletedSetup = ref(false);
  const loading = ref(false);
  const saving = ref(false);
  const loaded = ref(false);
  const error = ref<string | null>(null);

  let loadPromise: Promise<void> | null = null;

  const enabledAgentSet = computed(() => new Set(enabledAgents.value));

  function applyState(state: AgentSetupState) {
    enabledAgents.value = Array.isArray(state.enabledAgents) ? state.enabledAgents : [];
    hasCompletedSetup.value = Boolean(state.agentSetupCompleted);
    loaded.value = true;
  }

  async function fetchState() {
    loading.value = true;
    error.value = null;
    try {
      applyState(await GetAgentSetupState());
    } catch (e) {
      error.value = String(e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function ensureLoaded() {
    if (loaded.value) return;
    if (!loadPromise) {
      loadPromise = fetchState().finally(() => {
        loadPromise = null;
      });
    }
    await loadPromise;
  }

  async function refreshEnabledAgents() {
    error.value = null;
    try {
      enabledAgents.value = await GetEnabledAgents();
      loaded.value = true;
    } catch (e) {
      error.value = String(e);
      throw e;
    }
  }

  async function saveEnabledAgents(agentIDs: string[]) {
    saving.value = true;
    error.value = null;
    try {
      await SetEnabledAgents(agentIDs);
      await fetchState();
    } catch (e) {
      error.value = String(e);
      throw e;
    } finally {
      saving.value = false;
    }
  }

  function isAgentEnabled(agentID: string) {
    return enabledAgentSet.value.has(agentID);
  }

  async function checkAgentEnabled(agentID: string) {
    return IsAgentEnabled(agentID);
  }

  return {
    enabledAgents,
    enabledAgentSet,
    hasCompletedSetup,
    loading,
    saving,
    loaded,
    error,
    ensureLoaded,
    fetchState,
    refreshEnabledAgents,
    saveEnabledAgents,
    isAgentEnabled,
    checkAgentEnabled,
  };
});
