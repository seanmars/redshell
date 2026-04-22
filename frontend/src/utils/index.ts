// ──────────────────────────────────────────────
// 字串工具
// ──────────────────────────────────────────────

/** 截斷字串並附加省略號 */
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str
  return str.slice(0, maxLength - 1) + '…'
}

/** 將 camelCase 轉為顯示用的 Title Case */
export function camelToTitle(str: string): string {
  return str.replace(/([A-Z])/g, ' $1').replace(/^./, (s) => s.toUpperCase())
}

// ──────────────────────────────────────────────
// 時間工具
// ──────────────────────────────────────────────

/** 格式化日期為本地字串（台灣格式） */
export function formatDate(date: Date | string | number, locale = 'zh-TW'): string {
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(date))
}

/** 延遲指定毫秒（可用於測試或動畫） */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// ──────────────────────────────────────────────
// 物件工具
// ──────────────────────────────────────────────

/** 從物件中移除值為 undefined 或 null 的 key */
export function cleanObject<T extends Record<string, unknown>>(obj: T): Partial<T> {
  return Object.fromEntries(
    Object.entries(obj).filter(([, v]) => v !== undefined && v !== null),
  ) as Partial<T>
}
