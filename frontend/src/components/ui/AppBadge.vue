<script setup lang="ts">
import type { DaisyColor } from '@/types';

interface Props {
  variant?: DaisyColor | 'outline';
  size?: 'xs' | 'sm' | 'md' | 'lg';
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm',
});

const variantClass: Record<NonNullable<Props['variant']>, string> = {
  primary: 'badge-primary',
  secondary: 'badge-secondary',
  accent: 'badge-accent',
  neutral: 'badge-neutral',
  info: 'badge-info',
  success: 'badge-success',
  warning: 'badge-warning',
  error: 'badge-error',
  outline: 'badge-outline',
};

const sizeClass: Record<NonNullable<Props['size']>, string> = {
  xs: 'badge-xs',
  sm: 'badge-sm',
  md: '',
  lg: 'badge-lg',
};
</script>

<template>
  <span
    class="badge"
    :class="[props.variant ? variantClass[props.variant] : '', sizeClass[props.size ?? 'sm']]"
  >
    <slot />
  </span>
</template>
