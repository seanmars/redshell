<script setup lang="ts">
import { ref } from 'vue';
import { ResumeSession } from '@wailsjs/go/app/SessionHistoryApp';
import { useToast } from '@/composables/useToast';
import AppIcon from '@/components/ui/AppIcon.vue';
import type { DaisySize } from '@/types';

interface Props {
  agentId: string;
  sessionId: string;
  cwd?: string;
  size?: DaisySize;
  tooltip?: string;
}

const props = withDefaults(defineProps<Props>(), {
  cwd: '',
  size: 'sm',
  tooltip: 'Resume session in terminal',
});

const sizeClass: Record<DaisySize, string> = {
  xs: 'btn-xs',
  sm: 'btn-sm',
  md: '',
  lg: 'btn-lg',
  xl: 'btn-xl',
};

const launching = ref(false);
const { push } = useToast();

async function handleClick() {
  if (launching.value) return;
  launching.value = true;
  try {
    await ResumeSession(props.agentId, props.sessionId, props.cwd);
    push({ type: 'success', message: 'Resuming session in new terminal' });
  } catch (e) {
    push({ type: 'error', message: `Failed to resume: ${String(e)}` });
  } finally {
    launching.value = false;
  }
}
</script>

<template>
  <button
    type="button"
    class="btn btn-ghost btn-circle"
    :class="sizeClass[props.size]"
    :title="props.tooltip"
    :aria-label="props.tooltip"
    :disabled="launching || !props.sessionId"
    @click="handleClick"
  >
    <AppIcon name="terminal" :size="props.size" />
  </button>
</template>
