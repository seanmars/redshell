<script setup lang="ts">
import { computed } from 'vue';
import AppBadge from '@/components/ui/AppBadge.vue';
import type { DaisyColor } from '@/types';

interface Props {
  kind: string;
  subtype?: string;
}

const props = defineProps<Props>();

const variantByKind: Record<string, DaisyColor | 'outline'> = {
  user: 'primary',
  assistant: 'accent',
  tool_use: 'info',
  tool_result: 'success',
  system: 'neutral',
  attachment: 'secondary',
  meta: 'outline',
};

const variant = computed(() => variantByKind[props.kind] ?? 'outline');
</script>

<template>
  <span class="inline-flex items-center gap-1.5">
    <AppBadge :variant="variant" size="sm">{{ props.kind }}</AppBadge>
    <span v-if="props.subtype && props.subtype !== props.kind" class="text-xs opacity-60">
      {{ props.subtype }}
    </span>
  </span>
</template>
