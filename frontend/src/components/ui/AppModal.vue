<script setup lang="ts">
import { onBeforeUnmount, watch } from 'vue';

interface Props {
  isOpen: boolean;
  size?: 'sm' | 'md' | 'lg';
  dismissable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  dismissable: true,
});

const emit = defineEmits<{ close: [] }>();

const sizeClass: Record<NonNullable<Props['size']>, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-2xl',
};

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.isOpen && props.dismissable) {
    emit('close');
  }
}

watch(
  () => props.isOpen,
  (open) => {
    if (open) {
      window.addEventListener('keydown', onKeydown);
    } else {
      window.removeEventListener('keydown', onKeydown);
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown);
});

function onBackdropSubmit() {
  if (props.dismissable) {
    emit('close');
  }
}
</script>

<template>
  <dialog :open="isOpen" class="modal modal-bottom sm:modal-middle">
    <div class="modal-box" :class="sizeClass[props.size ?? 'md']">
      <header v-if="$slots.header" class="font-bold text-lg mb-3">
        <slot name="header" />
      </header>
      <div>
        <slot />
      </div>
      <div v-if="$slots.actions" class="modal-action">
        <slot name="actions" />
      </div>
    </div>
    <form
      v-if="dismissable"
      method="dialog"
      class="modal-backdrop"
      @submit.prevent="onBackdropSubmit"
    >
      <button>close</button>
    </form>
  </dialog>
</template>
