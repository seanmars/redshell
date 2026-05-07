# template-vue-daisyui

一個以 **Vue 3 + daisyUI** 為核心的前端專案範本，整合了現代前端工具鏈，適合快速啟動新的 Web 應用程式。

## 技術棧

| 分類 | 套件 | 版本 |
|---|---|---|
| UI Framework | Vue 3 + TypeScript | ^3.5 |
| CSS / UI 元件 | Tailwind CSS 4 + daisyUI 5 | ^4 / ^5 |
| 狀態管理 | Pinia | ^3 |
| 路由 | Vue Router | ^5 |
| 工具 Composables | @vueuse/core | ^14 |
| 建置工具 | Vite | ^8 |
| 測試 | Vitest + @vue/test-utils | ^4 |
| Linter | ESLint + oxlint + Prettier | — |

---

## 快速開始

```sh
# 安裝相依套件
pnpm install

# 啟動開發伺服器（http://localhost:5173）
pnpm dev

# 型別檢查 + 建置 production bundle
pnpm build

# 預覽 production build
pnpm preview
```

### 其他指令

```sh
# 執行單元測試（watch 模式）
pnpm test:unit

# Lint 並自動修正（oxlint → eslint 依序執行）
pnpm lint

# Prettier 格式化 src/ 目錄
pnpm format
```

---

## 目錄架構

```
src/
├── assets/                  # 靜態資源（CSS、圖片、字型）
│   └── main.css             # 全域樣式，引入 Tailwind / daisyUI
│
├── components/              # Vue 元件
│   ├── ui/                  # 通用 UI Primitives（與業務邏輯無關）
│   │   ├── AppButton.vue    # 帶型別的 daisyUI button 封裝
│   │   ├── AppCard.vue      # 帶 title / actions slot 的 card
│   │   └── AppAlert.vue     # 帶 icon、可 dismiss 的 alert
│   ├── CounterCard.vue      # 業務元件範例
│   └── ThemeToggle.vue      # 深色/淺色主題切換按鈕
│
├── composables/             # 自定義 Composables（use 前綴）
│   ├── usePageTitle.ts      # 管理頁面 <title>，基於 @vueuse/core
│   └── useConfirm.ts        # Imperative 確認對話框狀態管理
│
├── layouts/                 # 頁面 Layout 元件
│   └── DefaultLayout.vue    # 含 navbar + footer 的主 layout
│
├── router/                  # Vue Router 設定
│   └── index.ts             # 路由定義（含 lazy-loading 範例）
│
├── stores/                  # Pinia 狀態管理
│   ├── counter.ts           # 計數器 store（範例）
│   └── theme.ts             # 深色/淺色主題 store
│
├── types/                   # TypeScript 型別定義
│   └── index.ts             # 共用 interface / type（DaisyColor、ApiResponse 等）
│
├── utils/                   # 純工具函式（非 reactive，無 Vue 依賴）
│   └── index.ts             # truncate、formatDate、cleanObject 等
│
├── views/                   # 頁面元件（對應路由）
│   ├── HomeView.vue
│   └── AboutView.vue
│
├── App.vue                  # 根元件
└── main.ts                  # 應用程式進入點
```

---

## 各目錄使用說明

### `components/ui/` — UI Primitives

封裝 daisyUI 元件，提供 TypeScript props 支援。**不應包含業務邏輯**，只處理樣式與互動。

```vue
<AppButton variant="primary" size="lg" :loading="isSubmitting">
  送出
</AppButton>

<AppCard title="我的卡片" :shadow="true">
  卡片內容
  <template #actions>
    <AppButton variant="ghost" size="sm">取消</AppButton>
  </template>
</AppCard>

<AppAlert type="success" title="操作成功" dismissible @dismiss="showAlert = false">
  資料已儲存。
</AppAlert>
```

### `composables/` — 自定義 Composables

存放封裝**可重用 reactive 邏輯**的 `use*` 函式。優先利用 `@vueuse/core` 提供的底層工具。

```ts
// 在任何 <script setup> 中使用
import { usePageTitle } from '@/composables/usePageTitle'

const { setTitle } = usePageTitle('首頁')
// => document.title === "首頁 | MyApp"
```

```ts
import { useConfirm } from '@/composables/useConfirm'

const { confirm } = useConfirm()

async function handleDelete() {
  const ok = await confirm({ message: '確定要刪除嗎？', confirmLabel: '刪除' })
  if (ok) { /* 執行刪除 */ }
}
```

### `layouts/` — 頁面 Layout

當應用程式有多種 layout（例如：有 sidebar 的後台、無 navbar 的登入頁）時，在此建立不同的 layout 元件，再於 router 中透過 `meta` 切換。

```vue
<!-- views/SomePage.vue -->
<template>
  <DefaultLayout>
    <div class="container mx-auto py-8">
      <!-- 頁面內容 -->
    </div>
  </DefaultLayout>
</template>
```

### `stores/` — Pinia Store

使用 **Composition API 風格**（`defineStore` + `setup function`）定義 store，與 Vue 3 的開發體驗保持一致。

```ts
// stores/user.ts
import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', () => {
  const name = ref('')
  const isLoggedIn = computed(() => name.value !== '')
  return { name, isLoggedIn }
})
```

### `types/` — TypeScript 型別

存放**跨檔案共用**的 interface、type alias、enum。避免把型別定義散落在各個元件中。

```ts
// 引入共用型別
import type { ApiResponse, DaisyColor, NavItem } from '@/types'
```

### `utils/` — 純工具函式

存放**不依賴 Vue reactive 系統**的純函式。可以獨立測試，不應 import Vue 相關 API。

```ts
import { truncate, formatDate, cleanObject } from '@/utils'

truncate('這是一段很長的文字', 8)   // => "這是一段很長…"
formatDate(new Date())             // => "2026/03/31"
cleanObject({ a: 1, b: null })     // => { a: 1 }
```

---

## 主題切換

本範本已整合 daisyUI 的 **light / dark** 主題切換，並持久化至 `localStorage`。

- 切換邏輯在 `stores/theme.ts`
- UI 元件在 `components/ThemeToggle.vue`
- 偵測系統偏好（`prefers-color-scheme`）作為預設值

如需新增更多 daisyUI 主題，在 `src/assets/main.css` 中設定：

```css
@plugin "daisyui" {
  themes: light, dark, cupcake, dracula;
}
```

---

## 新增頁面流程

1. 在 `src/views/` 建立 `XxxView.vue`
2. 在 `src/router/index.ts` 新增路由（建議使用 lazy-loading）
3. 如有需要，在 `src/stores/` 建立對應的 store

```ts
// router/index.ts
{
  path: '/dashboard',
  name: 'dashboard',
  component: () => import('@/views/DashboardView.vue'),
}
```

---

## 開發環境建議

- **IDE**: [VS Code](https://code.visualstudio.com/) + [Vue - Official](https://marketplace.visualstudio.com/items?itemName=Vue.volar) 擴充套件（請停用 Vetur）
- **瀏覽器**: Chromium 系列 + [Vue.js devtools](https://chromewebstore.google.com/detail/vuejs-devtools/nhdogjmejiglipccpnnnanhbledajbpd)
- **Node.js**: `^20.19.0` 或 `>=22.12.0`（見 `package.json` engines）
