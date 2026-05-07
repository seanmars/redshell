<script setup lang="ts">
interface Props {
  modelValue?: string | number;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  size?: 'sm' | 'md' | 'lg';
  label?: string;
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  size: 'md',
});

const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

const sizeClass: Record<NonNullable<Props['size']>, string> = {
  sm: 'input-sm',
  md: '',
  lg: 'input-lg',
};

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).value);
}
</script>

<template>
  <label v-if="props.label" class="form-control">
    <div class="label">
      <span class="label-text">{{ props.label }}</span>
    </div>
    <input
      class="input input-bordered w-full"
      :class="sizeClass[props.size ?? 'md']"
      :type="props.type"
      :value="props.modelValue ?? ''"
      :placeholder="props.placeholder"
      :disabled="props.disabled"
      @input="onInput"
    />
  </label>
  <input
    v-else
    class="input input-bordered w-full"
    :class="sizeClass[props.size ?? 'md']"
    :type="props.type"
    :value="props.modelValue ?? ''"
    :placeholder="props.placeholder"
    :disabled="props.disabled"
    @input="onInput"
  />
</template>
