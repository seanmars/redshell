<script setup lang="ts">
import SessionListItem from './SessionListItem.vue';
import AppCollapse from '@/components/ui/AppCollapse.vue';
import AppEmptyState from '@/components/ui/AppEmptyState.vue';
import AppSpinner from '@/components/ui/AppSpinner.vue';
import type { sessionhistory } from '@wailsjs/go/models';

interface Props {
  listing: sessionhistory.Listing | undefined;
  loading: boolean;
  error: string;
  selectedSessionId: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{ select: [agentID: string, sessionID: string] }>();

function handleSelect(agentID: string, sessionID: string) {
  emit('select', agentID, sessionID);
}

function shortPath(input: string | undefined): { parent: string; root: string } {
  if (!input) return { parent: '', root: '' };
  const parts = input.split(/[\\/]/).filter((p) => p.length > 0);
  if (parts.length === 0) return { parent: '', root: input };
  if (parts.length === 1) return { parent: '', root: parts[0]! };
  return { parent: parts[parts.length - 2]!, root: parts[parts.length - 1]! };
}
</script>

<template>
  <div class="h-full flex flex-col">
    <div v-if="props.loading" class="flex-1 flex items-center justify-center">
      <AppSpinner size="md" />
    </div>

    <div v-else-if="props.error" class="p-4">
      <p class="text-sm text-error">{{ props.error }}</p>
    </div>

    <AppEmptyState
      v-else-if="!props.listing"
      icon="folder"
      title="No sessions"
      description="No session history available for this agent."
    />

    <div v-else-if="props.listing.kind === 'flat'" class="flex-1 overflow-auto p-2 space-y-1">
      <AppEmptyState
        v-if="!props.listing.flat || props.listing.flat.length === 0"
        icon="folder"
        title="No sessions"
        description="No session history available for this agent."
      />
      <SessionListItem
        v-else
        v-for="s in props.listing.flat"
        :key="s.sessionID"
        :session="s"
        :selected="s.sessionID === props.selectedSessionId"
        @select="handleSelect"
      />
    </div>

    <div v-else-if="props.listing.kind === 'groups'" class="flex-1 overflow-auto p-2 space-y-2">
      <AppEmptyState
        v-if="!props.listing.groups || props.listing.groups.length === 0"
        icon="folder"
        title="No sessions"
        description="No session history available for this agent."
      />
      <AppCollapse
        v-else
        v-for="g in props.listing.groups"
        :key="g.encodedDir"
        :default-open="false"
      >
        <template #title>
          <span
            class="font-mono text-sm truncate inline-block max-w-full"
            :title="g.cwd || g.encodedDir"
          >
            <template v-if="shortPath(g.cwd).parent">
              <span class="opacity-50">{{ shortPath(g.cwd).parent }}</span>
              <span class="opacity-50">/</span>
            </template>
            <span>{{ shortPath(g.cwd).root || g.encodedDir }}</span>
          </span>
        </template>
        <div class="space-y-1">
          <SessionListItem
            v-for="s in g.sessions"
            :key="s.sessionID"
            :session="s"
            :selected="s.sessionID === props.selectedSessionId"
            @select="handleSelect"
          />
        </div>
      </AppCollapse>
    </div>
  </div>
</template>
