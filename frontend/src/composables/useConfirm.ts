import { ref } from 'vue'

interface ConfirmOptions {
  title?: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
}

/**
 * 提供 imperative 的確認對話框狀態管理
 * 需搭配 AppConfirmModal 元件使用
 */
export function useConfirm() {
  const isOpen = ref(false)
  const options = ref<ConfirmOptions>({ message: '' })
  let resolveCallback: ((value: boolean) => void) | null = null

  function confirm(opts: ConfirmOptions): Promise<boolean> {
    options.value = opts
    isOpen.value = true
    return new Promise((resolve) => {
      resolveCallback = resolve
    })
  }

  function onConfirm() {
    isOpen.value = false
    resolveCallback?.(true)
  }

  function onCancel() {
    isOpen.value = false
    resolveCallback?.(false)
  }

  return { isOpen, options, confirm, onConfirm, onCancel }
}
