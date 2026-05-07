<script setup lang="ts">
import type { DaisyColor } from '@/types';

interface Props {
  type?: Extract<DaisyColor, 'info' | 'success' | 'warning' | 'error'>;
  title?: string;
  dismissible?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: 'info',
});

const emit = defineEmits<{ dismiss: [] }>();

const typeClass: Record<string, string> = {
  info: 'alert-info',
  success: 'alert-success',
  warning: 'alert-warning',
  error: 'alert-error',
};

const iconName: Record<string, string> = {
  info: 'mdi:information-outline',
  success: 'mdi:check-circle-outline',
  warning: 'mdi:alert-outline',
  error: 'mdi:close-circle-outline',
};
</script>

<template>
  <div role="alert" class="alert" :class="typeClass[props.type]">
    <iconify-icon :icon="iconName[props.type]" aria-hidden="true" class="h-6 w-6 shrink-0" />
    <div>
      <h3 v-if="props.title" class="font-bold">{{ props.title }}</h3>
      <slot />
    </div>
    <button v-if="props.dismissible" class="btn btn-sm btn-ghost" @click="emit('dismiss')">
      ✕
    </button>
  </div>
</template>
