<script setup lang="ts">
import AppButton from './AppButton.vue';
import AppModal from './AppModal.vue';

defineProps<{
  isOpen: boolean;
  title?: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
}>();

const emit = defineEmits<{
  confirm: [];
  cancel: [];
}>();
</script>

<template>
  <AppModal :is-open="isOpen" size="sm" @close="emit('cancel')">
    <template #header>{{ title ?? 'Confirm' }}</template>
    <p class="py-2">{{ message }}</p>
    <template #actions>
      <AppButton variant="ghost" @click="emit('cancel')">{{ cancelLabel ?? 'Cancel' }}</AppButton>
      <AppButton variant="error" @click="emit('confirm')">{{
        confirmLabel ?? 'Confirm'
      }}</AppButton>
    </template>
  </AppModal>
</template>
