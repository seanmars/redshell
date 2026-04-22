<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import ThemeToggle from '@/components/ThemeToggle.vue'
</script>

<template>
  <div class="min-h-screen bg-base-100 flex flex-col">
    <nav class="navbar bg-base-200 shadow-sm px-4">
      <div class="flex-1">
        <RouterLink to="/" class="btn btn-ghost text-xl font-bold">MyApp</RouterLink>
      </div>
      <div class="flex-none gap-1">
        <ul class="menu menu-horizontal px-1">
          <li><RouterLink to="/">Home</RouterLink></li>
          <li><RouterLink to="/about">About</RouterLink></li>
        </ul>
        <ThemeToggle />
      </div>
    </nav>

    <main class="flex-1">
      <RouterView />
    </main>

    <footer class="footer footer-center p-4 bg-base-200 text-base-content">
      <p class="text-sm opacity-60">© 2026 MyApp. Built with Vue + daisyUI.</p>
    </footer>
  </div>
</template>
