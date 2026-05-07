<script lang="ts">
export type IconName =
  | 'browse'
  | 'installed'
  | 'settings'
  | 'sessions'
  | 'hooks'
  | 'check'
  | 'sun'
  | 'moon'
  | 'plus'
  | 'refresh'
  | 'trash'
  | 'warning'
  | 'folder'
  | 'file';

const mdiByName: Record<IconName, string> = {
  browse: 'mdi:magnify',
  installed: 'mdi:package-variant-closed',
  settings: 'mdi:cog-outline',
  sessions: 'mdi:history',
  hooks: 'mdi:hook',
  check: 'mdi:check-bold',
  sun: 'mdi:white-balance-sunny',
  moon: 'mdi:weather-night',
  plus: 'mdi:plus',
  refresh: 'mdi:refresh',
  trash: 'mdi:trash-can-outline',
  warning: 'mdi:alert-outline',
  folder: 'mdi:folder-outline',
  file: 'mdi:file-outline',
};
</script>

<script setup lang="ts">
import { computed } from 'vue';
import { Icon } from '@iconify/vue';
import type { DaisySize } from '@/types';

interface Props {
  name: IconName;
  size?: DaisySize;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
});

const sizeMap: Record<DaisySize, string> = {
  xs: 'w-3 h-3',
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-6 h-6',
  xl: 'w-8 h-8',
};

const iconAttr = computed(() => mdiByName[props.name]);
</script>

<template>
  <Icon :icon="iconAttr" aria-hidden="true" :class="sizeMap[props.size ?? 'md']" />
</template>
