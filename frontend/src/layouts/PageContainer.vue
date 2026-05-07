<script setup lang="ts">
interface Props {
  title: string;
  /** Optional suffix appended after an em dash; reactive in callers. */
  titleSuffix?: string;
  /** Tailwind max-width utility for the centered content column. */
  maxWidth?: string;
  /**
   * When true, the content wrapper becomes a flex column that fills the
   * available viewport height instead of sizing to its children. Use for
   * views whose body must fit-and-fill (e.g. master/detail panes that scroll
   * internally). Children should declare their own height (h-full / flex-1)
   * to take advantage of the fill.
   */
  fill?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  maxWidth: 'max-w-5xl',
  fill: false,
});
</script>

<template>
  <div class="h-full min-h-0 flex flex-col">
    <div class="shrink-0 bg-base-100/85 backdrop-blur-md border-b border-base-300/60">
      <div
        :class="[
          props.maxWidth,
          'mx-auto px-6 h-20 pt-6 flex items-start justify-between gap-4 overflow-hidden',
        ]"
      >
        <h1
          class="text-3xl font-semibold tracking-tight leading-none flex flex-col gap-y-1 min-w-0"
        >
          <span class="whitespace-nowrap">{{ props.title }}</span>
          <span
            v-if="props.titleSuffix"
            class="text-xl font-normal opacity-60 break-words leading-tight"
          >
            {{ props.titleSuffix }}
          </span>
        </h1>
        <div v-if="$slots.actions" class="flex items-center gap-2 shrink-0">
          <slot name="actions" />
        </div>
      </div>
    </div>

    <div
      class="flex-1 min-h-0"
      :class="props.fill ? 'overflow-hidden flex flex-col' : 'overflow-auto'"
    >
      <div
        :class="[
          props.maxWidth,
          'mx-auto px-6 pt-6 pb-8',
          props.fill ? 'flex-1 min-h-0 w-full flex flex-col' : '',
        ]"
      >
        <slot />
      </div>
    </div>
  </div>
</template>
