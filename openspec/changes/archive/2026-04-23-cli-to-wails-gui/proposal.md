## Why

ai-tools 目前只提供 CLI 介面，對於管理 AI provider (Claude Code, GitHub Copilot) 的 marketplace 和 plugin 來說操作門檻偏高，需要記憶指令且缺乏視覺化回饋。透過 Wails GUI app，可以讓使用者以更直覺的方式瀏覽、安裝、管理 plugins，並統一管理多個 AI provider 的設定。

## What Changes

- 新建一個 Wails (Go + React) 桌面 GUI 應用程式，取代現有的 ink/meow CLI 互動流程
- 延續 ai-tools CLI 的核心邏輯，透過呼叫各 AI provider CLI 指令 (`claude plugin marketplace add`, `claude plugin install`, `copilot plugin marketplace add`, `copilot plugin install`) 執行安裝與 marketplace 管理，不直接修改 provider 的設定檔或複製檔案至 provider 目錄
- Marketplace 瀏覽 (讀取 `~/.redshell/marketplace.json`) 和 plugin 清單抓取 (GitHub/GitLab API) 以 Go 原生實作
- 所有業務邏輯封裝於與 Wails 完全無關的 Go 套件中，使 Wails app 與未來的 Go CLI 可共用相同的 service 層
- 提供視覺化的 marketplace 管理介面 (新增/移除/列表)
- 提供 plugin 瀏覽器介面，支援依 provider 過濾和多選安裝
- 提供已安裝 plugin 管理介面 (依 provider 分類展示、移除)
- 提供 AI provider 設定管理介面

## Capabilities

### New Capabilities

- `wails-app-shell`: Wails 應用程式框架，包含 Go main entry、React 前端 scaffold、window 設定與 build pipeline
- `provider-management`: 管理 AI provider 設定 (Claude Code `~/.claude`, GitHub Copilot `~/.copilot`)，顯示各 provider 的設定路徑與狀態
- `marketplace-management`: CRUD 操作 `~/.redshell/marketplace.json`，支援新增 git repo URL、移除已登錄 marketplace、列表展示
- `plugin-browser`: 從已登錄的 marketplaces 抓取並展示可安裝的 plugins，支援依 provider 過濾、多選、確認安裝流程
- `installed-plugins-view`: 讀取各 provider 的已安裝 plugin 清單 (`installed_plugins.json`)，支援依 provider 分頁展示與移除操作

### Modified Capabilities

## Impact

- **新增依賴:** Wails v2 (Go), Vue 3, Pinia, Vue Router 5, TailwindCSS 4, daisyUI 5
- **新增 Go module:** `github.com/wailsapp/wails/v2`
- **受影響資料路徑:** `~/.redshell/marketplace.json` (讀寫), `~/.claude/plugins/installed_plugins.json`, `~/.copilot/config.json` (僅讀取，寫入由 provider CLI 負責)
- **與現有 CLI 並行:** Wails GUI app 作為獨立專案存在於 `redshell` repo，不修改 `ai-tools` 的 CLI 邏輯
- **外部 API:** GitHub REST API, GitLab REST API (需要 GITHUB_TOKEN / GITLAB_TOKEN)
