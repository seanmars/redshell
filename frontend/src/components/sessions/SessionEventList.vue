<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue';
import SessionEventItem from './SessionEventItem.vue';
import AppSpinner from '@/components/ui/AppSpinner.vue';
import AppCheckbox from '@/components/ui/AppCheckbox.vue';
import type { sessionhistory } from '@wailsjs/go/models';

interface Props {
  events: sessionhistory.Event[];
  hasMore: boolean;
  loading: boolean;
  total: number;
  skippedLines: number;
}

const props = defineProps<Props>();
const emit = defineEmits<{ loadMore: [] }>();

const wrap = ref(true);

const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function setupObserver() {
  if (observer) observer.disconnect();
  if (!sentinel.value) return;
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting && props.hasMore && !props.loading) {
          emit('loadMore');
        }
      }
    },
    { rootMargin: '200px' },
  );
  observer.observe(sentinel.value);
}

onMounted(setupObserver);
onBeforeUnmount(() => observer?.disconnect());

// Re-attach on prop change in case the sentinel re-mounts.
watch(
  () => [props.hasMore, sentinel.value],
  () => setupObserver(),
);
</script>

<template>
  <div class="flex flex-col h-full">
    <div
      class="flex items-center justify-end gap-3 px-3 py-1.5 border-b border-base-300/60 shrink-0"
    >
      <AppCheckbox v-model="wrap" size="sm">
        <span class="text-xs opacity-80">Wrap long content</span>
      </AppCheckbox>
    </div>

    <div
      v-if="props.skippedLines > 0"
      class="text-xs text-warning px-3 py-2 border-b border-base-300/60 shrink-0"
    >
      {{ props.skippedLines }} unparseable line{{ props.skippedLines === 1 ? '' : 's' }} skipped
    </div>

    <div
      class="flex-1 overflow-y-auto px-2 py-2 space-y-1.5"
      :class="wrap ? 'overflow-x-hidden' : 'overflow-x-auto'"
    >
      <SessionEventItem v-for="ev in props.events" :key="ev.index" :event="ev" :wrap="wrap" />

      <div v-if="props.hasMore" ref="sentinel" class="py-4 flex justify-center">
        <AppSpinner v-if="props.loading" size="sm" />
        <span v-else class="text-xs opacity-60">Scroll for more…</span>
      </div>
      <div v-else-if="props.events.length > 0" class="py-4 text-center text-xs opacity-50">
        — end of session ({{ props.total }} event{{ props.total === 1 ? '' : 's' }}) —
      </div>
    </div>
  </div>
</template>
