<script setup lang="ts" generic="T extends string | number">
import type { DaisySize } from '@/types';

interface Option {
  value: T;
  label: string;
}

interface Props {
  modelValue: T;
  options: Option[];
  size?: DaisySize;
  disabled?: boolean;
  bordered?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  bordered: true,
});

const emit = defineEmits<{ 'update:modelValue': [value: T] }>();

const sizeClass: Record<DaisySize, string> = {
  xs: 'select-xs',
  sm: 'select-sm',
  md: '',
  lg: 'select-lg',
  xl: 'select-lg',
};

function onChange(event: Event) {
  const target = event.target as HTMLSelectElement;
  const raw = target.value;
  const matched = props.options.find((o) => String(o.value) === raw);
  if (matched) {
    emit('update:modelValue', matched.value);
  }
}
</script>

<template>
  <select
    class="select"
    :class="[sizeClass[props.size ?? 'md'], { 'select-bordered': props.bordered }]"
    :value="String(props.modelValue)"
    :disabled="props.disabled"
    @change="onChange"
  >
    <option v-for="o in props.options" :key="String(o.value)" :value="String(o.value)">
      {{ o.label }}
    </option>
  </select>
</template>
