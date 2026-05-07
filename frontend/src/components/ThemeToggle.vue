<script setup lang="ts">
import { useThemeStore } from '@/stores/theme';
import AppIcon from '@/components/ui/AppIcon.vue';

const themeStore = useThemeStore();
</script>

<template>
  <label class="swap swap-rotate btn btn-ghost btn-circle">
    <input
      type="checkbox"
      :checked="themeStore.isDark"
      :aria-label="themeStore.isDark ? 'Switch to light theme' : 'Switch to dark theme'"
      @change="themeStore.toggle"
    />
    <AppIcon name="sun" size="md" class="swap-on" />
    <AppIcon name="moon" size="md" class="swap-off" />
  </label>
</template>
