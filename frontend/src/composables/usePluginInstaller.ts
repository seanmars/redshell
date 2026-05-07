import { computed, ref, watch, type ComputedRef } from 'vue';
import type { plugin } from '@wailsjs/go/models';
import type { usePluginStore } from '@/stores/plugin';

type PluginStore = ReturnType<typeof usePluginStore>;

export function usePluginInstaller(store: PluginStore, allowedAgents: ComputedRef<string[]>) {
  const showInstallModal = ref(false);
  const targetAgents = ref<string[]>([]);
  const installError = ref<string | null>(null);

  const selectedMergedPlugins = computed(() =>
    store.mergedPlugins.filter((mp) => store.selected.has(`${mp.name}@${mp.marketplace}`)),
  );

  const availableAgents = computed(() => {
    const set = new Set<string>();
    for (const mp of selectedMergedPlugins.value) {
      for (const agt of mp.agents) {
        if (allowedAgents.value.includes(agt)) {
          set.add(agt);
        }
      }
    }
    return set;
  });

  watch(showInstallModal, (open) => {
    if (open) {
      targetAgents.value =
        availableAgents.value.size === 1 ? Array.from(availableAgents.value) : [];
    }
  });

  async function handleInstall() {
    installError.value = null;
    try {
      for (const agt of targetAgents.value) {
        const pluginsToInstall = selectedMergedPlugins.value
          .filter((mp) => mp.agents.includes(agt))
          .map((mp) => mp.sourcePlugins[agt])
          .filter((p): p is plugin.MarketplacePlugin => p !== undefined);
        if (pluginsToInstall.length > 0) {
          await store.installSelected(agt, pluginsToInstall);
        }
      }
      showInstallModal.value = false;
    } catch (e) {
      installError.value = String(e);
    }
  }

  function closeModal() {
    showInstallModal.value = false;
    installError.value = null;
  }

  return {
    showInstallModal,
    targetAgents,
    installError,
    selectedMergedPlugins,
    availableAgents,
    handleInstall,
    closeModal,
  };
}
