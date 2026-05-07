<script setup lang="ts">
import type { DaisySize } from '@/types';

interface Props {
  variant?: 'spinner' | 'dots' | 'ring' | 'ball' | 'bars';
  size?: DaisySize;
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'spinner',
  size: 'md',
});

const variantClass: Record<NonNullable<Props['variant']>, string> = {
  spinner: 'loading-spinner',
  dots: 'loading-dots',
  ring: 'loading-ring',
  ball: 'loading-ball',
  bars: 'loading-bars',
};

const sizeClass: Record<DaisySize, string> = {
  xs: 'loading-xs',
  sm: 'loading-sm',
  md: 'loading-md',
  lg: 'loading-lg',
  xl: 'loading-lg',
};
</script>

<template>
  <span
    class="loading"
    :class="[variantClass[props.variant ?? 'spinner'], sizeClass[props.size ?? 'md']]"
  />
</template>
