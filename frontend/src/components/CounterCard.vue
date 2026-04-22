<script setup lang="ts">
import { useCounterStore } from '@/stores/counter'

const counter = useCounterStore()
</script>

<template>
  <div class="card bg-base-200 shadow-md w-72">
    <div class="card-body items-center text-center gap-3">
      <h2 class="card-title">計數器</h2>
      <div class="stats stats-vertical shadow w-full">
        <div class="stat">
          <div class="stat-title">目前數值</div>
          <div class="stat-value text-primary">{{ counter.count }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">雙倍數值</div>
          <div class="stat-value text-secondary">{{ counter.doubleCount }}</div>
        </div>
      </div>
      <div class="card-actions">
        <button class="btn btn-primary btn-wide" @click="counter.increment">+1 增加</button>
      </div>
    </div>
  </div>
</template>
