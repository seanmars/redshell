<script setup lang="ts">
import { useToast } from '@/composables/useToast';

const { toasts } = useToast();

const alertClass: Record<string, string> = {
  success: 'alert-success',
  error: 'alert-error',
  info: 'alert-info',
  warning: 'alert-warning',
};
</script>

<template>
  <div v-if="toasts.length > 0" class="toast toast-top toast-end z-50">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="alert shadow-lg"
      :class="alertClass[t.type]"
      role="status"
    >
      <span class="text-sm">{{ t.message }}</span>
    </div>
  </div>
</template>
