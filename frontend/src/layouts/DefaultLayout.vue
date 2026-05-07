<script setup lang="ts">
import { RouterLink } from 'vue-router';
import AppToast from '@/components/ui/AppToast.vue';
import AppIcon, { type IconName } from '@/components/ui/AppIcon.vue';
import UpdateAvailableBanner from '@/components/system/UpdateAvailableBanner.vue';

const navItems: Array<{ to: string; label: string; icon: IconName }> = [
  { to: '/browse', label: 'Plugins', icon: 'browse' },
  { to: '/sessions', label: 'Sessions', icon: 'sessions' },
  { to: '/hooks', label: 'Hooks', icon: 'hooks' },
  { to: '/installed', label: 'Installed', icon: 'installed' },
];
</script>

<template>
  <div class="h-full flex">
    <!-- Sidebar -->
    <aside
      class="w-56 bg-base-200 border-r border-base-content/5 flex flex-col shrink-0 overflow-y-auto"
    >
      <nav class="flex-1 p-2">
        <ul class="menu menu-md w-full gap-0.5">
          <li v-for="item in navItems" :key="item.to">
            <RouterLink :to="item.to" active-class="menu-active" class="gap-3">
              <AppIcon :name="item.icon" size="md" />
              <span class="font-medium tracking-tight">{{ item.label }}</span>
            </RouterLink>
          </li>
        </ul>
      </nav>

      <div class="border-t border-base-content/5 p-2 flex">
        <RouterLink
          to="/settings"
          class="btn btn-ghost btn-circle"
          title="Settings"
          aria-label="Settings"
        >
          <AppIcon name="settings" size="md" />
        </RouterLink>
      </div>
    </aside>

    <!-- Main content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <UpdateAvailableBanner />
      <main class="flex-1 min-h-0 overflow-hidden flex flex-col">
        <slot />
      </main>
    </div>

    <AppToast />
  </div>
</template>
