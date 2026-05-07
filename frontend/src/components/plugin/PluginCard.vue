<script lang="ts">
const agentLabels: Record<string, string> = {
  claude: 'Claude',
  copilot: 'Copilot',
};

const agentClasses: Record<string, { border: string; filled: string; outlined: string }> = {
  claude: {
    border: 'border-agent-claude',
    filled: 'bg-agent-claude text-white',
    outlined: 'text-agent-claude',
  },
  copilot: {
    border: 'border-agent-copilot',
    filled: 'bg-agent-copilot text-white',
    outlined: 'text-agent-copilot',
  },
};
</script>

<script setup lang="ts">
import type { MergedPlugin } from '@/stores/plugin';
import AppBadge from '@/components/ui/AppBadge.vue';
import AppIcon from '@/components/ui/AppIcon.vue';

const props = defineProps<{
  plugin: MergedPlugin;
  selected: boolean;
}>();

const emit = defineEmits<{
  toggle: [key: string];
}>();

function handleClick() {
  emit('toggle', `${props.plugin.name}@${props.plugin.marketplace}`);
}

function badgeClass(agt: string, installed: boolean): string {
  const cls = agentClasses[agt];
  if (!cls) return '';
  return `${cls.border} ${installed ? cls.filled : cls.outlined}`;
}
</script>

<template>
  <div
    class="group relative flex items-center gap-3 px-4 py-3 rounded-lg bg-base-200 ring-1 ring-base-content/5 cursor-pointer transition-[background-color,box-shadow,transform] duration-200 hover:bg-base-300 hover:ring-base-content/10 motion-safe:active:scale-[0.995]"
    :class="{
      'bg-primary/10 ring-primary/40 hover:bg-primary/15': selected,
    }"
    @click="handleClick"
  >
    <div class="shrink-0">
      <div v-if="selected" class="w-5 h-5 rounded-full bg-primary flex items-center justify-center">
        <AppIcon name="check" size="xs" class="text-primary-content" />
      </div>
      <div
        v-else
        class="w-5 h-5 rounded-full border-2 border-base-content/25 transition-colors group-hover:border-base-content/45"
      />
    </div>

    <div class="flex-1 min-w-0">
      <h3 class="font-semibold text-base tracking-tight">{{ plugin.name }}</h3>
      <div class="flex gap-1 mt-1">
        <AppBadge
          v-for="agt in plugin.agents"
          :key="agt"
          size="sm"
          class="gap-1 border"
          :class="badgeClass(agt, plugin.installedAgents.includes(agt))"
        >
          <AppIcon v-if="plugin.installedAgents.includes(agt)" name="check" size="xs" />
          {{ agentLabels[agt] ?? agt }}
        </AppBadge>
      </div>
      <p v-if="plugin.description" class="text-sm text-base-content/70 mt-1 leading-snug">
        {{ plugin.description }}
      </p>
    </div>
  </div>
</template>
