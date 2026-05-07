<script setup lang="ts">
interface Props {
  /** Tailwind height utility, e.g. 'h-4'. */
  height?: string;
  /** Tailwind width utility, e.g. 'w-full', 'w-1/3'. */
  width?: string;
  shape?: 'rect' | 'circle';
}

const props = withDefaults(defineProps<Props>(), {
  height: 'h-4',
  width: 'w-full',
  shape: 'rect',
});
</script>

<template>
  <div
    class="bg-base-content/10 motion-safe:animate-pulse"
    :class="[props.height, props.width, props.shape === 'circle' ? 'rounded-full' : 'rounded-md']"
    aria-hidden="true"
  />
</template>
