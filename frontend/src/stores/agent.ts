import { defineStore } from 'pinia';
import { ref } from 'vue';
import { ListAgents } from '@wailsjs/go/app/AgentApp';
import type { agent } from '@wailsjs/go/models';

export const useAgentStore = defineStore('agent', () => {
  const agents = ref<agent.Agent[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchAgents() {
    loading.value = true;
    error.value = null;
    try {
      agents.value = await ListAgents();
    } catch (e) {
      error.value = String(e);
    } finally {
      loading.value = false;
    }
  }

  return { agents, loading, error, fetchAgents };
});
