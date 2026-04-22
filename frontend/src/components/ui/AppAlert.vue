<script setup lang="ts">
import type { DaisyColor } from '@/types'

interface Props {
  type?: Extract<DaisyColor, 'info' | 'success' | 'warning' | 'error'>
  title?: string
  dismissible?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'info',
})

const emit = defineEmits<{ dismiss: [] }>()

const typeClass: Record<string, string> = {
  info: 'alert-info',
  success: 'alert-success',
  warning: 'alert-warning',
  error: 'alert-error',
}

const iconPath: Record<string, string> = {
  info: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
  success: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
  warning:
    'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z',
  error: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z',
}
</script>

<template>
  <div role="alert" class="alert" :class="typeClass[props.type]">
    <svg
      xmlns="http://www.w3.org/2000/svg"
      class="h-6 w-6 shrink-0 stroke-current"
      fill="none"
      viewBox="0 0 24 24"
    >
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        :d="iconPath[props.type]"
      />
    </svg>
    <div>
      <h3 v-if="props.title" class="font-bold">{{ props.title }}</h3>
      <slot />
    </div>
    <button v-if="props.dismissible" class="btn btn-sm btn-ghost" @click="emit('dismiss')">
      ✕
    </button>
  </div>
</template>
