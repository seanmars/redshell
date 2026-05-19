<script setup lang="ts">
import { computed } from 'vue';
import type { plugin as PluginTypes } from '@wailsjs/go/models';
import AppCard from '@/components/ui/AppCard.vue';
import AppButton from '@/components/ui/AppButton.vue';

const props = withDefaults(
  defineProps<{
    plugin: PluginTypes.InstalledPlugin;
    busy?: boolean;
  }>(),
  { busy: false },
);

const emit = defineEmits<{
  uninstall: [id: string];
  update: [id: string];
}>();

const pluginName = computed(() => {
  const { displayName, marketplaceName } = props.plugin;
  if (!marketplaceName) return displayName;
  const suffix = `@${marketplaceName}`;
  return displayName.endsWith(suffix) ? displayName.slice(0, -suffix.length) : displayName;
});

const subtitle = computed(() => {
  const base = props.plugin.marketplaceName || props.plugin.agent;
  const version = props.plugin.version?.trim();
  if (!version) return base;
  return `${base} · v${version}`;
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
          {{ subtitle }}
        </p>
      </div>
      <div class="flex flex-row items-center gap-2">
        <AppButton
          variant="outline"
          size="md"
          :loading="busy"
          :disabled="busy"
          @click="emit('update', plugin.uninstallName)"
        >
          Update
        </AppButton>
        <AppButton
          variant="outline"
          size="md"
          :disabled="busy"
          @click="emit('uninstall', plugin.uninstallName)"
        >
          Uninstall
        </AppButton>
      </div>
    </div>
  </AppCard>
</template>
