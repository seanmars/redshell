<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import EnabledAgentList from '@/components/agent/EnabledAgentList.vue';
import AppAlert from '@/components/ui/AppAlert.vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppSkeleton from '@/components/ui/AppSkeleton.vue';
import { useToast } from '@/composables/useToast';
import { useAgentStore } from '@/stores/agent';
import { useAgentSetupStore } from '@/stores/agentSetup';

const agentStore = useAgentStore();
const setupStore = useAgentSetupStore();
const toast = useToast();

const selectedAgents = ref<string[]>([]);
const saveError = ref<string | null>(null);

const hasChanges = computed(
  () => JSON.stringify(selectedAgents.value) !== JSON.stringify(setupStore.enabledAgents),
);

onMounted(async () => {
  await setupStore.ensureLoaded();
  await agentStore.fetchAgents();
  selectedAgents.value = [...setupStore.enabledAgents];
});

async function handleSave() {
  saveError.value = null;
  try {
    await setupStore.saveEnabledAgents(selectedAgents.value);
    selectedAgents.value = [...setupStore.enabledAgents];
    toast.push({ type: 'success', message: 'Enabled agents updated.' });
  } catch (e) {
    saveError.value = String(e);
  }
}
</script>

<template>
  <div class="space-y-4">
    <AppAlert type="info">
      RedShell only shows plugin browsing, installation, and installed states for enabled agents.
      Keep at least one agent enabled to save your changes.
    </AppAlert>

    <div
      v-if="agentStore.loading || !setupStore.loaded"
      class="grid grid-cols-1 md:grid-cols-2 gap-4"
      aria-busy="true"
      aria-live="polite"
    >
      <div
        v-for="i in 2"
        :key="i"
        class="rounded-2xl bg-base-200 ring-1 ring-base-content/5 px-6 py-5 space-y-3"
      >
        <AppSkeleton height="h-5" width="w-1/3" />
        <AppSkeleton height="h-4" width="w-1/4" />
        <div class="space-y-2 pt-2">
          <AppSkeleton height="h-3" width="w-3/4" />
          <AppSkeleton height="h-3" width="w-2/3" />
          <AppSkeleton height="h-3" width="w-1/2" />
        </div>
      </div>
    </div>

    <EnabledAgentList
      v-else
      v-model="selectedAgents"
      :agents="agentStore.agents"
      :disabled="setupStore.saving"
    />

    <AppAlert v-if="saveError" type="error">
      {{ saveError }}
    </AppAlert>

    <div class="flex justify-end">
      <AppButton
        :loading="setupStore.saving"
        :disabled="selectedAgents.length === 0 || !hasChanges"
        @click="handleSave"
      >
        Save Enabled Agents
      </AppButton>
    </div>
  </div>
</template>
