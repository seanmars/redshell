<script setup lang="ts">
import ThemeToggle from '@/components/ThemeToggle.vue';
import CloseBehaviorPromptModal from '@/components/system/CloseBehaviorPromptModal.vue';
import { useAppVersion } from '@/composables/useAppVersion';

const { version } = useAppVersion();
</script>

<template>
  <div class="h-screen bg-base-100 flex flex-col">
    <nav class="navbar bg-base-200 shadow-sm px-4 shrink-0 z-50">
      <div class="flex-1">
        <span class="text-xl font-bold px-2">RedShell</span>
        <span v-if="version" class="text-sm opacity-80 -ml-1">v{{ version }}</span>
      </div>
      <div class="flex-none">
        <ThemeToggle />
      </div>
    </nav>

    <main class="flex-1 overflow-hidden">
      <RouterView />
    </main>

    <CloseBehaviorPromptModal />
  </div>
</template>
