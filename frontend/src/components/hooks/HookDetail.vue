<script setup lang="ts">
import { computed } from 'vue';
import AppButton from '@/components/ui/AppButton.vue';
import AppBadge from '@/components/ui/AppBadge.vue';
import AppIcon from '@/components/ui/AppIcon.vue';
import HookSourceBadge from '@/components/hooks/HookSourceBadge.vue';
import { OpenPath } from '@wailsjs/go/app/SystemApp';
import { useToast } from '@/composables/useToast';
import type { hooks } from '@wailsjs/go/models';

interface Props {
  hook: hooks.Hook;
  source: hooks.Source;
}

const props = defineProps<Props>();
const toast = useToast();

const rawPretty = computed(() => {
  try {
    return JSON.stringify(props.hook.raw, null, 2);
  } catch {
    return '';
  }
});

interface Field {
  label: string;
  value: string;
}

const resolvedFields = computed<Field[]>(() => {
  const raw = props.hook.raw ?? {};
  const list: Field[] = [];

  const push = (label: string, key: string) => {
    const value = raw[key];
    if (value === undefined || value === null || value === '') return;
    if (typeof value === 'object') {
      list.push({ label, value: JSON.stringify(value) });
    } else {
      list.push({ label, value: String(value) });
    }
  };

  switch (props.hook.type) {
    case 'command':
      push('command', 'command');
      push('if', 'if');
      push('shell', 'shell');
      push('async', 'async');
      push('timeout', 'timeout');
      push('bash', 'bash');
      push('powershell', 'powershell');
      push('cwd', 'cwd');
      push('timeoutSec', 'timeoutSec');
      push('comment', 'comment');
      break;
    case 'http':
      push('url', 'url');
      push('headers', 'headers');
      push('allowedEnvVars', 'allowedEnvVars');
      push('if', 'if');
      push('timeout', 'timeout');
      break;
    case 'mcp_tool':
      push('server', 'server');
      push('tool', 'tool');
      push('input', 'input');
      push('if', 'if');
      push('timeout', 'timeout');
      break;
    case 'prompt':
    case 'agent':
      push('prompt', 'prompt');
      push('model', 'model');
      push('if', 'if');
      push('timeout', 'timeout');
      break;
    default:
      // Unknown type: detail pane shows raw JSON only.
      break;
  }
  return list;
});

const matcherText = computed(() => {
  if (!props.hook.matcher) return '';
  return props.hook.matcher;
});

async function openInOS() {
  try {
    await OpenPath(props.source.path);
  } catch (e) {
    toast.push({ type: 'error', message: `Failed to open file: ${e}` });
  }
}
</script>

<template>
  <div class="h-full overflow-y-auto p-4 space-y-4">
    <header class="space-y-2">
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-2 min-w-0">
          <HookSourceBadge :source="props.source" size="md" />
          <AppBadge size="sm" variant="info">{{ props.hook.event }}</AppBadge>
          <AppBadge v-if="matcherText" size="sm" variant="outline">
            {{ matcherText }}
          </AppBadge>
          <AppBadge size="sm" variant="neutral">{{ props.hook.type }}</AppBadge>
        </div>
        <AppButton variant="ghost" size="sm" @click="openInOS">
          <AppIcon name="folder" size="sm" />
          <span class="ml-1">Open settings file</span>
        </AppButton>
      </div>

      <p class="text-xs text-base-content/60 break-all font-mono">
        {{ props.source.path }}
      </p>

      <div v-if="props.hook.dupCount > 1" class="flex">
        <AppBadge size="sm" variant="warning">
          appears in {{ props.hook.dupCount }} sources
        </AppBadge>
      </div>
    </header>

    <section v-if="resolvedFields.length > 0" class="space-y-1">
      <h3 class="text-sm font-semibold tracking-tight text-base-content/70">Resolved fields</h3>
      <dl class="grid grid-cols-[8rem_1fr] gap-x-3 gap-y-1 text-sm">
        <template v-for="field in resolvedFields" :key="field.label">
          <dt class="text-base-content/55 font-mono">{{ field.label }}</dt>
          <dd class="font-mono break-all">{{ field.value }}</dd>
        </template>
      </dl>
    </section>

    <section class="space-y-1">
      <h3 class="text-sm font-semibold tracking-tight text-base-content/70">Raw JSON</h3>
      <pre
        class="text-xs font-mono bg-base-200/60 ring-1 ring-base-content/5 rounded-md p-3 overflow-auto whitespace-pre"
      ><code>{{ rawPretty }}</code></pre>
    </section>
  </div>
</template>
