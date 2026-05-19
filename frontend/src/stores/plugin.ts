import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import {
  FetchAll,
  Install,
  ListInstalled,
  Uninstall,
  UpdatePlugin,
} from '@wailsjs/go/app/PluginApp';
import { Refresh as RefreshMarketplaces } from '@wailsjs/go/app/MarketplaceApp';
import { EventsOn } from '@wailsjs/runtime/runtime';
import type { plugin } from '@wailsjs/go/models';

export type MarketplaceErrorEntry = { agent: string; message: string };

export interface MergedPlugin {
  name: string;
  project: string;
  marketplace: string;
  marketplaceName: string;
  description?: string;
  agents: string[];
  sourcePlugins: Record<string, plugin.MarketplacePlugin>;
  installedAgents: string[];
}

export const usePluginStore = defineStore('plugin', () => {
  const plugins = ref<plugin.MarketplacePlugin[]>([]);
  const installedPlugins = ref<plugin.InstalledPlugin[]>([]);
  const selected = ref<Set<string>>(new Set());
  const loading = ref(false);
  const refreshing = ref(false);
  const installing = ref(false);
  const updatingPlugins = ref<Set<string>>(new Set());
  const installLog = ref<string[]>([]);
  const fetchErrors = ref<string[]>([]);
  const refreshWarnings = ref<Record<string, string>>({});
  const error = ref<string | null>(null);

  EventsOn('plugin:install-log', (msg: string) => {
    installLog.value.push(msg);
  });

  const pluginsByMarketplace = computed<Record<string, plugin.MarketplacePlugin[]>>(() => {
    const grouped: Record<string, plugin.MarketplacePlugin[]> = {};
    for (const p of plugins.value) {
      const key = p.marketplace;
      if (!grouped[key]) grouped[key] = [];
      grouped[key].push(p);
    }
    return grouped;
  });

  const errorsByMarketplace = computed<Record<string, MarketplaceErrorEntry[]>>(() => {
    const grouped: Record<string, MarketplaceErrorEntry[]> = {};
    const scopedPattern = /^\[([^/\]]+)\/([^\]]+)\]\s*(.*)$/;
    for (const entry of fetchErrors.value) {
      const match = scopedPattern.exec(entry);
      if (match) {
        const [, marketplaceID = '', agent = '', message = ''] = match;
        if (!grouped[marketplaceID]) grouped[marketplaceID] = [];
        grouped[marketplaceID]!.push({ agent, message });
      } else {
        if (!grouped.__global) grouped.__global = [];
        grouped.__global.push({ agent: '', message: entry });
      }
    }
    return grouped;
  });

  const mergedPlugins = computed<MergedPlugin[]>(() => {
    const map = new Map<string, MergedPlugin>();
    for (const p of plugins.value) {
      const key = `${p.name}@${p.marketplace}`;
      if (!map.has(key)) {
        map.set(key, {
          name: p.name,
          project: p.project,
          marketplace: p.marketplace,
          marketplaceName: p.marketplaceName,
          description: p.description,
          agents: [],
          sourcePlugins: {},
          installedAgents: [],
        });
      }
      const entry = map.get(key)!;
      entry.agents.push(p.agent);
      entry.sourcePlugins[p.agent] = p;
    }
    for (const entry of map.values()) {
      entry.installedAgents = entry.agents.filter((agt) =>
        installedPlugins.value.some(
          (ip) => ip.agent === agt && ip.uninstallName === entry.sourcePlugins[agt]?.installName,
        ),
      );
    }
    return Array.from(map.values());
  });

  const mergedPluginsByMarketplace = computed<Record<string, MergedPlugin[]>>(() => {
    const grouped: Record<string, MergedPlugin[]> = {};
    for (const p of mergedPlugins.value) {
      if (!grouped[p.marketplace]) grouped[p.marketplace] = [];
      grouped[p.marketplace]!.push(p);
    }
    return grouped;
  });

  async function fetchAll() {
    loading.value = true;
    error.value = null;
    fetchErrors.value = [];
    try {
      const result = await FetchAll();
      plugins.value = result.plugins ?? [];
      fetchErrors.value = result.errors ?? [];
    } catch (e) {
      error.value = String(e);
    } finally {
      loading.value = false;
    }
  }

  async function refreshAll() {
    refreshing.value = true;
    refreshWarnings.value = {};
    try {
      const result = await RefreshMarketplaces();
      const warnings: Record<string, string> = {};
      const sectionPattern = /^\[([^\]]+)\]\s*(.*)$/;
      for (const entry of result.errors ?? []) {
        const match = sectionPattern.exec(entry);
        if (match) {
          const [, marketplaceID = '', message = ''] = match;
          warnings[marketplaceID] = message;
        }
      }
      refreshWarnings.value = warnings;
    } catch (e) {
      error.value = String(e);
    } finally {
      refreshing.value = false;
    }
  }

  async function fetchInstalled(agentID: string) {
    loading.value = true;
    error.value = null;
    try {
      const result = await ListInstalled(agentID);
      installedPlugins.value = [
        ...installedPlugins.value.filter((p) => p.agent !== agentID),
        ...result,
      ];
    } catch (e) {
      error.value = String(e);
    } finally {
      loading.value = false;
    }
  }

  function toggleSelect(installName: string) {
    const next = new Set(selected.value);
    if (next.has(installName)) {
      next.delete(installName);
    } else {
      next.add(installName);
    }
    selected.value = next;
  }

  function clearSelection() {
    selected.value = new Set();
  }

  async function silentRefreshInstalled(agentID: string) {
    try {
      const result = await ListInstalled(agentID);
      installedPlugins.value = [
        ...installedPlugins.value.filter((p) => p.agent !== agentID),
        ...result,
      ];
    } catch {
      // best-effort; don't interrupt the install flow
    }
  }

  async function installSelected(agentID: string, selectedPlugins: plugin.MarketplacePlugin[]) {
    installing.value = true;
    installLog.value = [];
    error.value = null;
    try {
      await Install(agentID, selectedPlugins);
      clearSelection();
      await silentRefreshInstalled(agentID);
    } catch (e) {
      error.value = String(e);
      throw e;
    } finally {
      installing.value = false;
    }
  }

  async function uninstall(agentID: string, pluginID: string) {
    await Uninstall(agentID, pluginID);
    installedPlugins.value = installedPlugins.value.filter((p) => p.uninstallName !== pluginID);
  }

  function isPluginBusy(installName: string) {
    return updatingPlugins.value.has(installName);
  }

  async function update(agentID: string, installName: string) {
    const next = new Set(updatingPlugins.value);
    next.add(installName);
    updatingPlugins.value = next;
    try {
      await UpdatePlugin(agentID, installName);
      await silentRefreshInstalled(agentID);
    } finally {
      const cleared = new Set(updatingPlugins.value);
      cleared.delete(installName);
      updatingPlugins.value = cleared;
    }
  }

  return {
    plugins,
    installedPlugins,
    selected,
    loading,
    refreshing,
    installing,
    installLog,
    fetchErrors,
    refreshWarnings,
    error,
    pluginsByMarketplace,
    mergedPlugins,
    mergedPluginsByMarketplace,
    errorsByMarketplace,
    fetchAll,
    refreshAll,
    fetchInstalled,
    toggleSelect,
    clearSelection,
    installSelected,
    uninstall,
    update,
    isPluginBusy,
    updatingPlugins,
  };
});
