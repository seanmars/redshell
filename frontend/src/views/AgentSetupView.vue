<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import AppAlert from '@/components/ui/AppAlert.vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppCheckbox from '@/components/ui/AppCheckbox.vue';
import AppSkeleton from '@/components/ui/AppSkeleton.vue';
import { usePageTitle } from '@/composables/usePageTitle';
import { useAgentSetupStore } from '@/stores/agentSetup';

usePageTitle('Set Up Agents');

const setupAgents = [
  {
    id: 'claude',
    label: 'Claude Code',
    description: 'Enable Claude Code plugin browsing, installation, and installed plugin views.',
  },
  {
    id: 'copilot',
    label: 'GitHub Copilot',
    description: 'Enable GitHub Copilot plugin browsing, installation, and installed plugin views.',
  },
] as const;

const router = useRouter();
const setupStore = useAgentSetupStore();

const selectedAgents = ref<string[]>([]);
const saveError = ref<string | null>(null);

const isReady = computed(() => setupStore.loaded);

onMounted(async () => {
  await setupStore.ensureLoaded();
  selectedAgents.value = [...setupStore.enabledAgents];
});

async function handleContinue() {
  saveError.value = null;
  try {
    await setupStore.saveEnabledAgents(selectedAgents.value);
    await router.replace('/browse');
  } catch (e) {
    saveError.value = String(e);
  }
}
</script>

<template>
  <div class="h-full overflow-auto bg-base-100">
    <div class="mx-auto max-w-5xl px-6 py-10 space-y-6">
      <div class="space-y-3">
        <h1 class="text-4xl font-semibold tracking-tight">Choose your agents</h1>
        <p class="text-base text-base-content/70 max-w-3xl">
          RedShell only shows browsing, installation, and installed plugin workflows for the agents
          you enable here.
        </p>

        <AppAlert type="info"> You can change enabled agents later in Settings → Agents. </AppAlert>
      </div>

      <div v-if="!isReady" class="rounded-2xl border border-base-content/10" aria-busy="true">
        <div
          v-for="i in 2"
          :key="i"
          class="px-5 py-4 space-y-3"
          :class="i === 1 ? 'border-b border-base-content/10' : ''"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="flex-1 space-y-2">
              <AppSkeleton height="h-5" width="w-1/3" />
              <AppSkeleton height="h-4" width="w-1/4" />
              <AppSkeleton height="h-3" width="w-3/4" />
            </div>
            <AppSkeleton height="h-5" width="w-24" />
          </div>
          <div class="space-y-2 pt-2">
            <AppSkeleton height="h-3" width="w-2/3" />
            <AppSkeleton height="h-3" width="w-1/2" />
          </div>
        </div>
      </div>

      <div v-else class="rounded-2xl border border-base-content/10 overflow-hidden">
        <AppCheckbox
          v-for="(agent, index) in setupAgents"
          :key="agent.id"
          v-model="selectedAgents"
          :value="agent.id"
          :disabled="setupStore.saving"
          class="w-full items-start gap-3 px-5 py-4"
          :class="{ 'border-t border-base-content/10': index > 0 }"
        >
          <div class="min-w-0 flex-1 space-y-1.5">
            <p class="font-semibold tracking-tight">{{ agent.label }}</p>
            <p class="text-sm text-base-content/65">{{ agent.description }}</p>
          </div>
        </AppCheckbox>
      </div>

      <AppAlert v-if="saveError" type="error">
        {{ saveError }}
      </AppAlert>

      <div class="flex justify-end">
        <AppButton
          :loading="setupStore.saving"
          :disabled="selectedAgents.length === 0"
          @click="handleContinue"
        >
          Continue
        </AppButton>
      </div>
    </div>
  </div>
</template>
