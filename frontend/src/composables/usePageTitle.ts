import { useTitle } from '@vueuse/core'
import { computed } from 'vue'

const APP_NAME = 'MyApp'

/**
 * 管理頁面標題，自動附加 App 名稱
 * @example
 * const { setTitle } = usePageTitle('首頁')
 * // => document.title === "首頁 | MyApp"
 */
export function usePageTitle(pageTitle?: string) {
  const title = useTitle()

  function setTitle(name: string) {
    title.value = `${name} | ${APP_NAME}`
  }

  if (pageTitle) {
    setTitle(pageTitle)
  }

  const currentTitle = computed(() => title.value ?? APP_NAME)

  return { currentTitle, setTitle }
}
