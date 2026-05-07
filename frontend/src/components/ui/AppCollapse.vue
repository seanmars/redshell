<script setup lang="ts">
interface Props {
  title?: string;
  defaultOpen?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  defaultOpen: true,
});
</script>

<template>
  <details :open="props.defaultOpen" class="collapse collapse-arrow bg-base-200">
    <summary class="collapse-title text-lg font-semibold cursor-pointer">
      <slot name="title">{{ props.title }}</slot>
    </summary>
    <div class="collapse-content">
      <slot />
    </div>
  </details>
</template>
