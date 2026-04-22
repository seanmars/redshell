import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

export const useThemeStore = defineStore('theme', () => {
  const STORAGE_KEY = 'theme'
  const saved = localStorage.getItem(STORAGE_KEY)
  const isDark = ref(
    saved !== null ? saved === 'dark' : window.matchMedia('(prefers-color-scheme: dark)').matches,
  )

  function applyTheme() {
    const theme = isDark.value ? 'dark' : 'light'
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem(STORAGE_KEY, theme)
  }

  function toggle() {
    isDark.value = !isDark.value
  }

  watch(isDark, applyTheme, { immediate: true })

  return { isDark, toggle }
})
