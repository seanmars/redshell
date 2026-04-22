<script setup lang="ts">
import type { DaisyColor, DaisySize } from '@/types'

interface Props {
  variant?: DaisyColor | 'ghost' | 'outline' | 'link'
  size?: DaisySize
  loading?: boolean
  disabled?: boolean
  wide?: boolean
  block?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
})

const variantClass = {
  primary: 'btn-primary',
  secondary: 'btn-secondary',
  accent: 'btn-accent',
  neutral: 'btn-neutral',
  info: 'btn-info',
  success: 'btn-success',
  warning: 'btn-warning',
  error: 'btn-error',
  ghost: 'btn-ghost',
  outline: 'btn-outline',
  link: 'btn-link',
}

const sizeClass = {
  xs: 'btn-xs',
  sm: 'btn-sm',
  md: '',
  lg: 'btn-lg',
  xl: 'btn-xl',
}
</script>

<template>
  <button
    class="btn"
    :class="[
      variantClass[props.variant ?? 'primary'],
      sizeClass[props.size ?? 'md'],
      { 'btn-wide': props.wide, 'btn-block': props.block },
    ]"
    :disabled="props.disabled || props.loading"
  >
    <span v-if="props.loading" class="loading loading-spinner" />
    <slot />
  </button>
</template>
