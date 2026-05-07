<script setup lang="ts">
import { computed, ref } from 'vue';
import SessionEventBadge from './SessionEventBadge.vue';
import type { sessionhistory } from '@wailsjs/go/models';

interface Props {
  event: sessionhistory.Event;
  /**
   * When true (default), long text wraps and `min-w-0` is applied so flex
   * children can shrink. When false, text uses `whitespace-nowrap` and the
   * caller is expected to provide a horizontal scroll container.
   */
  wrap?: boolean;
}

const props = withDefaults(defineProps<Props>(), { wrap: true });

const expanded = ref(false);

const summary = computed(() => props.event.summary || props.event.subtype || props.event.kind);

const rowAccent: Record<string, string> = {
  user: 'border-l-primary',
  assistant: 'border-l-accent',
  tool_use: 'border-l-info',
  tool_result: 'border-l-success',
  system: 'border-l-base-content/30',
  attachment: 'border-l-secondary',
  meta: 'border-l-base-content/20',
};

const rowClass = computed(() => rowAccent[props.event.kind] ?? 'border-l-base-content/20');

const prettyJSON = computed(() => JSON.stringify(props.event.raw, null, 2));

function toggle() {
  expanded.value = !expanded.value;
}
</script>

<template>
  <div
    class="border-l-4 pl-3 py-2 bg-base-100 hover:bg-base-200/60 rounded-r-md transition-colors"
    :class="rowClass"
  >
    <button
      type="button"
      class="w-full text-left flex items-start gap-3 cursor-pointer"
      :class="props.wrap ? 'min-w-0' : ''"
      :aria-expanded="expanded"
      @click="toggle"
    >
      <SessionEventBadge :kind="props.event.kind" :subtype="props.event.subtype" />
      <span
        class="flex-1 text-sm leading-relaxed"
        :class="props.wrap ? 'min-w-0 break-words' : 'whitespace-nowrap'"
        >{{ summary }}</span
      >
      <span class="text-xs opacity-50 shrink-0">#{{ props.event.index }}</span>
    </button>

    <div v-if="props.event.children && props.event.children.length > 0" class="mt-2 ml-6 space-y-1">
      <div
        v-for="(child, i) in props.event.children"
        :key="i"
        class="text-xs flex items-center gap-2 opacity-80"
        :class="props.wrap ? 'min-w-0' : ''"
      >
        <SessionEventBadge :kind="child.kind" :subtype="child.subtype" />
        <span :class="props.wrap ? 'min-w-0 break-words' : 'whitespace-nowrap'">{{
          child.summary
        }}</span>
      </div>
    </div>

    <div v-if="expanded" class="mt-3">
      <pre
        class="text-xs bg-base-200 rounded-md p-3 overflow-x-auto"
        :class="props.wrap ? 'whitespace-pre-wrap break-words' : 'whitespace-pre'"
        >{{ prettyJSON }}</pre
      >
    </div>
  </div>
</template>
