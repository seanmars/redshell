<script setup lang="ts">
interface Props {
  title?: string
  /** 緊湊模式（card-compact） */
  compact?: boolean
  /** 顯示陰影 */
  shadow?: boolean
  /** 置中內容 */
  center?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  shadow: true,
})
</script>

<template>
  <div
    class="card bg-base-200"
    :class="{
      'card-compact': props.compact,
      'shadow-md': props.shadow,
    }"
  >
    <div class="card-body" :class="{ 'items-center text-center': props.center }">
      <h2 v-if="props.title" class="card-title">{{ props.title }}</h2>
      <slot />
      <div v-if="$slots.actions" class="card-actions justify-end">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>
