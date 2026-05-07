<script setup lang="ts">
import { computed, inject, onBeforeUnmount, watchEffect, type ComputedRef } from 'vue';

interface Props {
  id: string;
  label: string;
}

const props = defineProps<Props>();

interface TabsApi {
  register(reg: { id: string; label: string }): void;
  unregister(id: string): void;
  activeId: ComputedRef<string>;
}

const tabsApi = inject<TabsApi>('app-tabs');

if (!tabsApi) {
  throw new Error('AppTab must be used inside <AppTabs>');
}

watchEffect(() => {
  tabsApi.register({ id: props.id, label: props.label });
});

onBeforeUnmount(() => {
  tabsApi.unregister(props.id);
});

const isActive = computed(() => tabsApi.activeId.value === props.id);
</script>

<template>
  <div v-if="isActive" role="tabpanel">
    <slot />
  </div>
</template>
