<script setup lang="ts">
import { computed } from 'vue';
import type { plugin as PluginTypes } from '@wailsjs/go/models';
import AppCard from '@/components/ui/AppCard.vue';
import AppButton from '@/components/ui/AppButton.vue';

const props = defineProps<{
  plugin: PluginTypes.InstalledPlugin;
}>();

const emit = defineEmits<{
  uninstall: [id: string];
}>();

const pluginName = computed(() => {
  const { displayName, marketplaceName } = props.plugin;
  if (!marketplaceName) return displayName;
  const suffix = `@${marketplaceName}`;
  return displayName.endsWith(suffix) ? displayName.slice(0, -suffix.length) : displayName;
});
</script>

<template>
  <AppCard compact>
    <div class="flex flex-row items-center justify-between gap-3">
      <div class="flex-1 min-w-0">
        <p class="text-base truncate tracking-tight">
          <span class="font-semibold">{{ pluginName }}</span>
          <span
            v-if="plugin.marketplaceName"
            class="font-normal text-base-content/45 font-mono text-sm"
            >@{{ plugin.marketplaceName }}</span
          >
        </p>
        <p class="text-sm text-base-content/55 mt-0.5">
          {{ plugin.marketplaceName || plugin.agent }}
        </p>
      </div>
      <AppButton variant="ghost" size="md" @click="emit('uninstall', plugin.uninstallName)">
        Uninstall
      </AppButton>
    </div>
  </AppCard>
</template>
