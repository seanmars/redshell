<script setup lang="ts">
import { computed } from 'vue';
import type { DaisyColor, DaisySize } from '@/types';

interface Props {
  modelValue?: boolean | unknown[];
  value?: unknown;
  variant?: DaisyColor;
  size?: DaisySize;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
});

const emit = defineEmits<{ 'update:modelValue': [value: boolean | unknown[]] }>();

const variantClass: Record<DaisyColor, string> = {
  primary: 'checkbox-primary',
  secondary: 'checkbox-secondary',
  accent: 'checkbox-accent',
  neutral: 'checkbox-neutral',
  info: 'checkbox-info',
  success: 'checkbox-success',
  warning: 'checkbox-warning',
  error: 'checkbox-error',
};

const sizeClass: Record<DaisySize, string> = {
  xs: 'checkbox-xs',
  sm: 'checkbox-sm',
  md: '',
  lg: 'checkbox-lg',
  xl: 'checkbox-lg',
};

const isArrayMode = computed(() => Array.isArray(props.modelValue));

const checked = computed(() => {
  if (isArrayMode.value) {
    return (props.modelValue as unknown[]).includes(props.value);
  }
  return Boolean(props.modelValue);
});

function onChange(e: Event) {
  const target = e.target as HTMLInputElement;
  if (isArrayMode.value) {
    const current = props.modelValue as unknown[];
    const next = target.checked
      ? [...current, props.value]
      : current.filter((v) => v !== props.value);
    emit('update:modelValue', next);
  } else {
    emit('update:modelValue', target.checked);
  }
}
</script>

<template>
  <label
    class="flex items-center gap-2"
    :class="props.disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'"
  >
    <input
      type="checkbox"
      class="checkbox"
      :class="[variantClass[props.variant ?? 'primary'], sizeClass[props.size ?? 'md']]"
      :checked="checked"
      :disabled="props.disabled"
      @change="onChange"
    />
    <span v-if="$slots.default" class="select-none"><slot /></span>
  </label>
</template>
