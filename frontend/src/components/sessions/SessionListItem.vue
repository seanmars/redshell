<script setup lang="ts">
import { computed } from 'vue';
import type { sessionhistory } from '@wailsjs/go/models';

interface Props {
  session: sessionhistory.SessionMeta;
  selected: boolean;
}

const props = defineProps<Props>();
const emit = defineEmits<{ select: [agentID: string, sessionID: string] }>();

const primaryLine = computed(() => {
  const s = props.session;
  if (s.summary && s.summary.trim() !== '') return s.summary;
  if (s.displayName && s.displayName.trim() !== '') return s.displayName;
  if (s.repository) return s.repository;
  if (s.cwd) return s.cwd;
  return basename(s.sessionID);
});

function basename(id: string): string {
  return id.includes('/') ? id.slice(id.lastIndexOf('/') + 1) : id;
}

const secondaryLine = computed(() => {
  const s = props.session;
  const parts: string[] = [];
  if (s.cwd && s.cwd !== primaryLine.value) parts.push(s.cwd);
  if (s.branch) parts.push(s.branch);
  return parts.join(' · ');
});

const timestamp = computed(() => {
  const s = props.session;
  const t = s.modifiedAt || s.updatedAt || s.createdAt || null;
  if (!t) {
    return '';
  }
  // to local time, without seconds
  const date = new Date(t);
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
});

function handleClick() {
  emit('select', props.session.agentID, props.session.sessionID);
}
</script>

<template>
  <button
    type="button"
    class="w-full text-left rounded-md px-3 py-2 hover:bg-base-300/40 transition-colors cursor-pointer"
    :class="props.selected ? 'bg-base-300/60 ring-1 ring-primary/40' : 'bg-base-100'"
    @click="handleClick"
  >
    <div class="flex items-start gap-2">
      <div class="flex-1 min-w-0">
        <div class="text-sm font-medium leading-tight truncate">{{ primaryLine }}</div>
        <div v-if="secondaryLine" class="text-xs opacity-60 leading-tight truncate mt-0.5">
          {{ secondaryLine }}
        </div>
        <div class="text-[11px] opacity-40 mt-1 flex items-center gap-2">
          <!-- <span class="font-mono">{{ tail }}</span> -->
          <span v-if="timestamp">{{ timestamp }}</span>
        </div>
      </div>
    </div>
  </button>
</template>
