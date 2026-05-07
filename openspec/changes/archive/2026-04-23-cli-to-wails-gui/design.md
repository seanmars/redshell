## Context

ai-tools 是一個使用 TypeScript + ink + meow 構建的 CLI 工具，用於管理 AI provider (Claude Code, GitHub Copilot) 的 marketplace 和 plugin。現有的資料層邏輯分布在 `src/data/` 和 `src/utils/` 中，配置儲存在 `~/.redshell/`、`~/.claude/`、`~/.copilot/`。

目標是在 `redshell` repo 中新建一個獨立的 Wails GUI 應用程式，複製並延伸 ai-tools 的核心功能，提供更直覺的操作介面。

## Goals / Non-Goals

**Goals:**
- 建立一個可在 macOS/Windows 運行的 Wails v2 桌面應用
- 延續 ai-tools CLI 的安裝邏輯，**透過呼叫各 AI provider CLI 指令** (`claude plugin marketplace add`, `claude plugin install` 等) 執行 marketplace 和 plugin 安裝，不直接修改 provider 檔案
- Marketplace 登錄管理 (`~/.redshell/marketplace.json` 的 CRUD) 和 plugin 清單抓取 (GitHub/GitLab API) 以 Go 原生實作
- **所有業務邏輯放在與 Wails 完全無關的 `internal/` 套件中，Wails 只是薄薄的 binding 層**，讓相同的 service 套件日後可直接被 Go CLI 使用
- 以 Vue 3 + daisyUI 實作前端介面，提供 marketplace、plugin、provider 三個主要管理頁面
- 與現有 ai-tools CLI 共用相同的設定檔路徑 (`~/.redshell/`, `~/.claude/`, `~/.copilot/`)

**Non-Goals:**
- 不修改 ai-tools 的 CLI 程式碼
- **不直接複製檔案至 provider 目錄，不直接修改 `~/.claude/` 或 `~/.copilot/` 內的設定** — 這些操作完全委由 provider CLI 負責
- 不實作 CI/CD 或自動更新機制
- 不支援自訂 provider 類型 (只支援 claude 和 copilot)
- 不提供 plugin 的 create/edit 功能，只支援 install/uninstall

## Decisions

### 1. Wails v2 over Electron
**選擇:** Wails v2
**理由:** Go backend 天然適合檔案系統操作和 HTTP API 呼叫；產出的 binary 較小；無需 Node.js runtime；Wails 的 binding 機制直接從 Go struct 生成 TypeScript types。
**替代方案:** Electron — 可直接復用 TypeScript 邏輯但 bundle size 大且記憶體消耗高。

### 2. Frontend: Vue 3 + Vite + daisyUI (已建構完成)
**選擇:** Vue 3.5 + Vite 8 + TailwindCSS 4 + daisyUI 5 + Pinia + Vue Router 5 + VueUse
**現況:** frontend 已完成 scaffold，技術棧固定如下：

| 層面 | 技術 | 版本 |
|------|------|------|
| 框架 | Vue 3 | 3.5.33 |
| 狀態管理 | Pinia | 3.0.4 |
| 路由 | Vue Router | 5.0.6 |
| CSS | TailwindCSS + daisyUI | 4.2.2 / 5.5.19 |
| 工具函數 | VueUse | 14.2.1 |
| 建置 | Vite | 8.0.9 |
| 測試 | Vitest | 4.1.5 |
| Linting | ESLint + Oxlint | 10.2.1 / 1.61.0 |

**已建立的共用基礎結構：**
- `src/components/ui/` — `AppAlert.vue`, `AppButton.vue`, `AppCard.vue`
- `src/composables/` — `useConfirm.ts`, `usePageTitle.ts`
- `src/layouts/DefaultLayout.vue` — navbar + footer 佈局
- `src/stores/theme.ts` — dark/light 主題管理 (含 localStorage 持久化)
- `wailsjs/` — Wails Go binding 自動生成目錄

**頁面開發慣例：**
- 新頁面放在 `src/views/` 目錄，使用 `.vue` SFC 格式
- Pinia store 放在 `src/stores/`
- 可復用 UI 元件放在 `src/components/ui/`

### 3. Go Backend 資料層架構 (Wails-agnostic service layer)
**選擇:** 採用兩層架構：
- **`internal/` 套件 (pure Go):** `marketplace.Service`、`plugin.Service`、`provider.Service` 完全不 import 任何 Wails 套件，只依賴 Go 標準庫 (`net/http`, `os`, `io`, `encoding/json`, `os/exec`)。這一層可被任何 Go 程式使用。
  - `plugin.Service.Install()` 和 `plugin.Service.Uninstall()` 透過 `os/exec` 呼叫 provider CLI 指令，不直接操作 provider 目錄
  - `marketplace.Service.Add()` 的 marketplace 登錄僅寫入 `~/.redshell/marketplace.json`；provider 端的 marketplace 登錄在安裝 plugin 前由 `plugin.Service` 透過 CLI 完成 (對應 ai-tools 的 `ensureMarketplace`)
- **`app/` 套件 (Wails binding layer):** 薄薄的 struct，embed 或持有 `internal/` 的 service，exported methods 僅做參數轉換後委派給 service，此層才 import Wails。

```
internal/
  marketplace/service.go   ← no wails import
  plugin/service.go        ← no wails import
  provider/service.go      ← no wails import
app/
  marketplace_app.go       ← wails binding wrapper
  plugin_app.go            ← wails binding wrapper
  provider_app.go          ← wails binding wrapper
main.go                    ← wails.Run, bind app/* structs
```

**理由:** 當日後需要製作 Go CLI 時，只需新建 `cmd/cli/` entry point，直接引用 `internal/` 套件，無需修改任何業務邏輯。Wails GUI 和 Go CLI 共用同一份 service 實作，確保行為一致。
**替代方案 (已排除):** 將業務邏輯直接寫在 Wails binding struct 內 — 會導致 CLI 無法復用，**明確不採用**。

### 4. 共用設定檔路徑
**選擇:**
- Go backend 直接讀寫 `~/.redshell/marketplace.json` (marketplace 登錄)
- `~/.claude/` 和 `~/.copilot/` 僅以唯讀方式讀取 (檢查 installed plugins 狀態)；寫入操作完全委由 provider CLI 負責
**理由:** 與 ai-tools CLI 的資料互通，同時避免繞過 provider CLI 的驗證邏輯直接修改其設定，確保安裝行為與 CLI 一致。

## Risks / Trade-offs

- **GitHub/GitLab API rate limit** → 加入 token 設定頁面，讀取 `GITHUB_TOKEN` / `GITLAB_TOKEN` 環境變數作為 fallback
- **跨平台路徑差異 (Windows `~` 展開)** → 使用 Go `os.UserHomeDir()` 統一處理
- **Provider CLI 未安裝** → `os/exec` 呼叫失敗時回傳明確錯誤訊息，提示使用者先安裝對應 CLI
- **Provider CLI 執行結果串流回前端** → 使用 Wails Event (EventsEmit) 將 stdout/stderr 即時推送到 Vue 前端顯示安裝進度
- **shadcn/ui 需要 Node.js 環境建構** → Wails build 已整合 Node 前端建構步驟，無額外問題

## Migration Plan

1. 建立 Wails 專案 scaffold (`wails init`)
2. 實作 Go backend services
3. 實作 React frontend pages
4. 本地測試，確認與 `ai-tools` CLI 資料互通
5. Wails build 產出獨立 binary

## Open Questions

- 是否需要 app icon 和 code signing (macOS)?
- Plugin 安裝是否保留 CLI 的互動確認步驟，或直接在 GUI 中以 dialog 取代?
