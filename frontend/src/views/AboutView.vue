<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="container mx-auto px-4 py-16 max-w-2xl">
    <div class="card bg-base-200 shadow-md">
      <div class="card-body gap-4">
        <h1 class="card-title text-3xl">關於此專案</h1>
        <p class="text-base-content/70">
          這個專案是以 <strong>Vue 3</strong> + <strong>daisyUI</strong> 為基礎的前端範本，
          適合快速啟動新的 Web 應用程式。
        </p>
        <div class="divider"></div>
        <ul class="list-none space-y-2">
          <li class="flex items-center gap-2">
            <span class="badge badge-primary">Vue 3</span>
            Composition API + TypeScript
          </li>
          <li class="flex items-center gap-2">
            <span class="badge badge-secondary">daisyUI</span>
            基於 Tailwind CSS 的 UI 元件庫
          </li>
          <li class="flex items-center gap-2">
            <span class="badge badge-accent">Vite</span>
            快速的開發與建置工具
          </li>
        </ul>
        <div class="card-actions justify-start mt-2">
          <RouterLink to="/" class="btn btn-primary btn-sm">回到首頁</RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>
