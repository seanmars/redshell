<script setup lang="ts">
import type { agent as AgentTypes } from '@wailsjs/go/models';
import { OpenPath } from '@wailsjs/go/app/SystemApp';
import AppCard from '@/components/ui/AppCard.vue';
import AppBadge from '@/components/ui/AppBadge.vue';
import AppIcon from '@/components/ui/AppIcon.vue';
import { useToast } from '@/composables/useToast';

defineProps<{
  agent: AgentTypes.Agent;
}>();

const { push: pushToast } = useToast();

async function openPath(path: string) {
  try {
    await OpenPath(path);
  } catch (e) {
    pushToast({ type: 'error', message: `Failed to open ${path}: ${String(e)}` });
  }
}
</script>

<template>
  <AppCard :title="agent.label" shadow>
    <div class="flex items-center gap-2 mb-3">
      <AppBadge v-if="agent.version" variant="info" size="sm">
        {{ agent.version }}
      </AppBadge>
      <span v-else class="inline-flex items-center text-warning" aria-label="Not installed">
        <AppIcon name="warning" size="sm" />
      </span>
    </div>
    <div class="text-sm space-y-1">
      <button
        type="button"
        class="group relative flex items-center gap-2 w-full text-left rounded-md px-2 py-1.5 cursor-pointer transition-colors hover:bg-base-300/60 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        :aria-label="`Open directory ${agent.configDir}`"
        @click="openPath(agent.configDir)"
      >
        <AppIcon name="folder" size="sm" />
        <span class="font-medium">Directory:</span>
        <span class="opacity-80 truncate">{{ agent.configDir }}</span>
        <span
          role="tooltip"
          class="pointer-events-none absolute left-2 bottom-full mb-1 hidden group-hover:block group-focus-visible:block whitespace-nowrap rounded-md bg-neutral text-neutral-content text-xs px-2 py-1 shadow-lg z-20"
        >
          Click to open directory: {{ agent.configDir }}
        </span>
      </button>
      <button
        type="button"
        class="group relative flex items-center gap-2 w-full text-left rounded-md px-2 py-1.5 cursor-pointer transition-colors hover:bg-base-300/60 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        :aria-label="`Open file ${agent.settingsFile}`"
        @click="openPath(agent.settingsFile)"
      >
        <AppIcon name="file" size="sm" />
        <span class="font-medium">Configuration:</span>
        <span class="opacity-80 truncate">{{ agent.settingsFile }}</span>
        <span
          role="tooltip"
          class="pointer-events-none absolute left-2 bottom-full mb-1 hidden group-hover:block group-focus-visible:block whitespace-nowrap rounded-md bg-neutral text-neutral-content text-xs px-2 py-1 shadow-lg z-20"
        >
          Click to open file: {{ agent.settingsFile }}
        </span>
      </button>
    </div>
    <p v-if="!agent.configured" class="text-xs mt-2 text-warning">
      Install {{ agent.label }} to enable this agent.
    </p>
    <slot name="extra" />
  </AppCard>
</template>
