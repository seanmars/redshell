<script setup lang="ts">
import { computed, provide, reactive, ref } from 'vue';

interface Props {
  active: string;
  variant?: 'bordered' | 'boxed' | 'lift';
  /**
   * When true, the tabpanel fills its parent's height via flex column. The
   * caller must give the AppTabs root an explicit height (e.g. h-full).
   * Children inside the active AppTab should also opt in to the height
   * (h-full / flex-1) — leaving this off keeps the natural content-height
   * behaviour used by most views.
   */
  fill?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'bordered',
  fill: false,
});

const emit = defineEmits<{ 'update:active': [id: string] }>();

interface TabRegistration {
  id: string;
  label: string;
}

const tabs = reactive<Map<string, TabRegistration>>(new Map());
const order = ref<string[]>([]);

const tabsApi = {
  register(reg: TabRegistration) {
    if (!tabs.has(reg.id)) order.value.push(reg.id);
    tabs.set(reg.id, reg);
  },
  unregister(id: string) {
    tabs.delete(id);
    order.value = order.value.filter((x) => x !== id);
  },
  activeId: computed(() => props.active),
};

provide('app-tabs', tabsApi);

const variantClass: Record<NonNullable<Props['variant']>, string> = {
  bordered: 'tabs-border',
  boxed: 'tabs-box',
  lift: 'tabs-lift',
};

const orderedTabs = computed(() =>
  order.value.map((id) => tabs.get(id)).filter((t): t is TabRegistration => t !== undefined),
);

function selectTab(id: string) {
  if (id !== props.active) emit('update:active', id);
}
</script>

<template>
  <div :class="props.fill ? 'flex flex-col min-h-0' : ''">
    <div
      role="tablist"
      class="tabs"
      :class="[variantClass[props.variant ?? 'bordered'], props.fill ? 'shrink-0' : '']"
    >
      <button
        v-for="tab in orderedTabs"
        :key="tab.id"
        role="tab"
        type="button"
        class="tab"
        :class="{ 'tab-active': tab.id === props.active }"
        :aria-selected="tab.id === props.active"
        @click="selectTab(tab.id)"
      >
        {{ tab.label }}
      </button>
    </div>
    <div
      role="tabpanel"
      :class="[
        props.variant === 'lift'
          ? 'bg-base-100 border border-base-content/10 rounded-b-box p-5 pt-6'
          : '',
        props.fill ? 'flex-1 min-h-0 flex flex-col' : '',
      ]"
    >
      <slot />
    </div>
  </div>
</template>
