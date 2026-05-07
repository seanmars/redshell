<script setup lang="ts">
import { computed } from 'vue';
import HookList from '@/components/hooks/HookList.vue';
import HookDetail from '@/components/hooks/HookDetail.vue';
import AppEmptyState from '@/components/ui/AppEmptyState.vue';
import AppAlert from '@/components/ui/AppAlert.vue';
import type { hooks } from '@wailsjs/go/models';

interface Props {
  agentId: string;
  listing: hooks.Listing | null;
  loading: boolean;
  error: string;
  selectedHookId: string;
  selectedHook: hooks.Hook | null;
  selectedSource: hooks.Source | null;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  select: [hookID: string];
}>();

const showCopilotEmpty = computed(() => props.listing?.emptyReason === 'copilot-project-scoped');

const banners = computed(() => props.listing?.disableAll ?? []);

function handleSelect(hookID: string) {
  emit('select', hookID);
}
</script>

<template>
  <AppEmptyState
    v-if="showCopilotEmpty"
    icon="hooks"
    title="Copilot CLI hooks are project-scoped"
    description="Copilot CLI reads hooks from each project's .github/hooks/ folder. Workspace selection is coming in a future release."
  />

  <div v-else class="flex flex-col h-full min-h-0">
    <div v-if="banners.length > 0" class="space-y-2 mb-3 shrink-0">
      <AppAlert
        v-for="flag in banners"
        :key="flag.path"
        type="warning"
        title="Hooks globally disabled"
      >
        Set in {{ flag.path }}
      </AppAlert>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-[24rem_1fr] gap-3 flex-1 min-h-0">
      <div
        class="bg-base-200/40 rounded-md border border-base-300/60 overflow-hidden h-full min-h-0"
      >
        <HookList
          :listing="props.listing"
          :loading="props.loading"
          :error="props.error"
          :selected-hook-id="props.selectedHookId"
          @select="handleSelect"
        />
      </div>
      <div class="bg-base-100 rounded-md border border-base-300/60 overflow-hidden h-full min-h-0">
        <HookDetail
          v-if="props.selectedHook && props.selectedSource"
          :hook="props.selectedHook"
          :source="props.selectedSource"
        />
        <AppEmptyState
          v-else
          icon="file"
          title="Select a hook"
          description="Choose a hook from the list to see its details."
        />
      </div>
    </div>
  </div>
</template>
