<script setup lang="ts">
import type { agent as AgentTypes } from '@wailsjs/go/models';
import AgentCard from '@/components/agent/AgentCard.vue';
import AppCheckbox from '@/components/ui/AppCheckbox.vue';

interface Props {
  agents: AgentTypes.Agent[];
  modelValue: string[];
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
});

const emit = defineEmits<{
  'update:modelValue': [value: string[]];
}>();

function updateValue(value: boolean | unknown[]) {
  if (!Array.isArray(value)) {
    emit('update:modelValue', []);
    return;
  }
  emit(
    'update:modelValue',
    value.filter((item): item is string => typeof item === 'string'),
  );
}
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <AgentCard v-for="agent in props.agents" :key="agent.id" :agent="agent">
      <template #extra>
        <div class="mt-4 space-y-2">
          <AppCheckbox
            :model-value="props.modelValue"
            :value="agent.id"
            :disabled="props.disabled"
            @update:modelValue="updateValue"
          >
            <span class="font-medium">Enable {{ agent.label }}</span>
          </AppCheckbox>
        </div>
      </template>
    </AgentCard>
  </div>
</template>
