<script setup lang="ts">
import { computed } from 'vue';
import AppCollapse from '@/components/ui/AppCollapse.vue';
import AppBadge from '@/components/ui/AppBadge.vue';
import HookSourceBadge from '@/components/hooks/HookSourceBadge.vue';
import type { hooks } from '@wailsjs/go/models';

interface Props {
  listing: hooks.Listing | null;
  loading: boolean;
  error: string;
  selectedHookId: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  select: [hookID: string];
}>();

interface SourceBlock {
  source: hooks.Source;
  byEvent: Map<string, hooks.Hook[]>;
  error: hooks.SourceError | null;
}

const sourceBlocks = computed<SourceBlock[]>(() => {
  if (!props.listing) return [];
  const list = props.listing;
  const sources = list.sources ?? [];
  const allHooks = list.hooks ?? [];
  const allErrors = list.errors ?? [];
  return sources.map((source) => {
    const byEvent = new Map<string, hooks.Hook[]>();
    for (const hook of allHooks) {
      if (hook.sourceID !== source.id) continue;
      const arr = byEvent.get(hook.event) ?? [];
      arr.push(hook);
      byEvent.set(hook.event, arr);
    }
    const error = allErrors.find((e) => e.sourceID === source.id) ?? null;
    return { source, byEvent, error };
  });
});

function shortSummary(hook: hooks.Hook): string {
  const matcher = hook.matcher ? hook.matcher : '*';
  const parts = [matcher, hook.type, hook.summary].filter((x) => x !== '');
  return parts.join(' | ');
}

function copilotSummary(hook: hooks.Hook): string {
  return [hook.type, hook.summary].filter((x) => x !== '').join(' | ');
}

function rowSummary(hook: hooks.Hook): string {
  if (props.listing?.agentID === 'copilot') return copilotSummary(hook);
  return shortSummary(hook);
}

function isSelected(hook: hooks.Hook): boolean {
  return hook.id === props.selectedHookId;
}

function handleSelect(hook: hooks.Hook): void {
  emit('select', hook.id);
}

const sortedEvents = (m: Map<string, hooks.Hook[]>): string[] => {
  return Array.from(m.keys()).sort();
};
</script>

<template>
  <div class="h-full overflow-y-auto p-3 space-y-3">
    <div v-if="props.loading" class="text-sm text-base-content/60">Loading hooks...</div>
    <div v-else-if="props.error" class="text-sm text-error">{{ props.error }}</div>
    <div
      v-else-if="!props.listing || sourceBlocks.length === 0"
      class="text-sm text-base-content/60"
    >
      No hooks configured.
    </div>

    <AppCollapse
      v-for="block in sourceBlocks"
      :key="`${block.source.kind}::${block.source.path}`"
      :default-open="block.source.kind === 'user'"
    >
      <template #title>
        <div class="flex items-center gap-2 min-w-0">
          <HookSourceBadge :source="block.source" size="sm" />
          <span class="text-xs text-base-content/55 truncate">
            {{ block.source.path }}
          </span>
        </div>
      </template>

      <div v-if="block.error" class="text-sm text-error px-2 py-1">
        {{ block.error.message }}
      </div>

      <div v-else class="space-y-2">
        <AppCollapse
          v-for="event in sortedEvents(block.byEvent)"
          :key="event"
          :default-open="false"
        >
          <template #title>
            <div class="flex items-center justify-between gap-2">
              <span class="text-base font-medium tracking-tight">{{ event }}</span>
              <AppBadge size="xs" variant="neutral">
                {{ block.byEvent.get(event)?.length ?? 0 }}
              </AppBadge>
            </div>
          </template>

          <ul class="space-y-1">
            <li
              v-for="hook in block.byEvent.get(event)"
              :key="hook.id"
              class="px-3 py-2 rounded-md cursor-pointer transition-colors text-sm font-mono leading-snug truncate"
              :class="
                isSelected(hook) ? 'bg-primary/15 ring-1 ring-primary/40' : 'hover:bg-base-300/40'
              "
              :title="rowSummary(hook)"
              @click="handleSelect(hook)"
            >
              {{ rowSummary(hook) }}
            </li>
          </ul>
        </AppCollapse>
      </div>
    </AppCollapse>
  </div>
</template>
