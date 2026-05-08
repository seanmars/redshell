<script setup lang="ts">
import { computed } from 'vue';
import AppBadge from '@/components/ui/AppBadge.vue';
import type { hooks } from '@wailsjs/go/models';

interface Props {
  source: hooks.Source;
  size?: 'xs' | 'sm' | 'md';
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm',
});

const variant = computed(() => {
  switch (props.source.kind) {
    case 'user':
      return 'primary' as const;
    case 'local':
      return 'secondary' as const;
    case 'plugin':
      return 'accent' as const;
    default:
      return 'neutral' as const;
  }
});
</script>

<template>
  <AppBadge
    :variant="variant"
    :size="props.size"
    class="min-w-0 max-w-full whitespace-nowrap truncate"
    :title="props.source.label"
  >
    {{ props.source.label }}
  </AppBadge>
</template>
