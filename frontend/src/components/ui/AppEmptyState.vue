<script setup lang="ts">
import AppIcon, { type IconName } from '@/components/ui/AppIcon.vue';

interface Props {
  icon?: IconName;
  title: string;
  description?: string;
}

defineProps<Props>();
</script>

<template>
  <div class="flex flex-col items-center text-center py-16 px-6 gap-4">
    <div
      v-if="icon"
      class="grid place-items-center size-14 rounded-2xl bg-base-200 ring-1 ring-base-content/5 text-base-content/45"
    >
      <AppIcon :name="icon" size="lg" />
    </div>
    <div class="space-y-1.5 max-w-md">
      <h3 class="text-lg font-semibold tracking-tight">{{ title }}</h3>
      <p v-if="description" class="text-sm text-base-content/60 leading-relaxed">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.default" class="mt-2">
      <slot />
    </div>
  </div>
</template>
