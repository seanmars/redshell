<script setup lang="ts">
import { ref } from 'vue';
import { ClipboardSetText } from '@wailsjs/runtime/runtime';
import { useToast } from '@/composables/useToast';
import AppIcon from '@/components/ui/AppIcon.vue';
import type { DaisySize } from '@/types';

interface Props {
  text: string;
  size?: DaisySize;
  tooltip?: string;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm',
  tooltip: 'Copy',
});

const sizeClass: Record<DaisySize, string> = {
  xs: 'btn-xs',
  sm: 'btn-sm',
  md: '',
  lg: 'btn-lg',
  xl: 'btn-xl',
};

const copied = ref(false);
let resetTimer: ReturnType<typeof setTimeout> | null = null;
const { push } = useToast();

async function handleClick() {
  try {
    const ok = await ClipboardSetText(props.text);
    if (ok === false) {
      push({ type: 'error', message: 'Failed to copy' });
      return;
    }
    copied.value = true;
    if (resetTimer) clearTimeout(resetTimer);
    resetTimer = setTimeout(() => {
      copied.value = false;
      resetTimer = null;
    }, 1200);
    push({ type: 'success', message: 'Copied' });
  } catch {
    push({ type: 'error', message: 'Failed to copy' });
  }
}
</script>

<template>
  <button
    type="button"
    class="btn btn-outline btn-square"
    :class="sizeClass[props.size]"
    :title="props.tooltip"
    :aria-label="props.tooltip"
    @click="handleClick"
  >
    <AppIcon :name="copied ? 'check' : 'copy'" :size="props.size" />
  </button>
</template>
