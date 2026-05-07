import { defineStore } from 'pinia';
import { ref } from 'vue';
import { Add, List, Remove } from '@wailsjs/go/app/MarketplaceApp';
import { UpdateAgentMarketplace } from '@wailsjs/go/app/PluginApp';
import { GetEnabledAgents } from '@wailsjs/go/app/AgentApp';
import type { marketplace, plugin } from '@wailsjs/go/models';

export interface UpdateAllHooks {
  onAgentStart?: (agentId: string) => void;
  onAgentDone?: (outcome: plugin.AgentUpdateOutcome) => void;
}

export const useMarketplaceStore = defineStore('marketplace', () => {
  const marketplaces = ref<marketplace.Marketplace[]>([]);
  const loading = ref(false);
  const updating = ref(false);
  const error = ref<string | null>(null);

  async function fetchList() {
    loading.value = true;
    error.value = null;
    try {
      marketplaces.value = await List();
    } catch (e) {
      error.value = String(e);
    } finally {
      loading.value = false;
    }
  }

  async function add(url: string) {
    const m = await Add(url);
    marketplaces.value.push(m);
    return m;
  }

  async function remove(id: string) {
    await Remove(id);
    marketplaces.value = marketplaces.value.filter((m) => m.id !== id);
  }

  async function updateAll(hooks: UpdateAllHooks = {}): Promise<plugin.AgentUpdateOutcome[]> {
    updating.value = true;
    try {
      const agents = (await GetEnabledAgents()) ?? [];
      return await Promise.all(
        agents.map(async (agentId) => {
          hooks.onAgentStart?.(agentId);
          const outcome = await UpdateAgentMarketplace(agentId);
          hooks.onAgentDone?.(outcome);
          return outcome;
        }),
      );
    } finally {
      updating.value = false;
    }
  }

  return { marketplaces, loading, updating, error, fetchList, add, remove, updateAll };
});
