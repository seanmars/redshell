// ──────────────────────────────────────────────
// 通用 API 回應型別
// ──────────────────────────────────────────────

export interface ApiResponse<T> {
  data: T
  message: string
  success: boolean
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
  total: number
  page: number
  pageSize: number
}

// ──────────────────────────────────────────────
// 通用 UI 型別
// ──────────────────────────────────────────────

/** daisyUI 顏色語義 */
export type DaisyColor =
  | 'primary'
  | 'secondary'
  | 'accent'
  | 'neutral'
  | 'info'
  | 'success'
  | 'warning'
  | 'error'

/** daisyUI 尺寸 */
export type DaisySize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

// ──────────────────────────────────────────────
// 路由相關
// ──────────────────────────────────────────────

export interface NavItem {
  label: string
  to: string
  icon?: string
}
