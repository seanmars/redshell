<script setup lang="ts">
import { RouterLink } from 'vue-router'
import CounterCard from '@/components/CounterCard.vue'
</script>

<template>
  <div class="hero min-h-[calc(100vh-8rem)]">
    <div class="hero-content flex-col gap-10">
      <div class="text-center">
        <h1 class="text-5xl font-bold">Hello World!</h1>
        <p class="py-6 text-base-content/70 max-w-md">
          這是一個使用 Vue 3 + daisyUI 建立的範本專案。 開始修改這個頁面，打造你自己的應用程式吧！
        </p>
        <div class="flex gap-2 justify-center">
          <RouterLink to="/about" class="btn btn-primary">了解更多</RouterLink>
          <a
            href="https://daisyui.com"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-outline"
          >
            daisyUI 文件
          </a>
        </div>
      </div>

      <CounterCard />
    </div>
  </div>
</template>
