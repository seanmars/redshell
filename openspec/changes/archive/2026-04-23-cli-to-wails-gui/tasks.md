## 1. Wails App Scaffold

- [x] 1.1 確認 `go.mod` 已包含 Wails v2 dependency
- [x] 1.2 設定 `wails.json`，包含 app name "RedShell"、minimum window size 1024x700
- [x] 1.3 在 `src/router/index.ts` 新增四個路由：`/providers`、`/marketplaces`、`/browse`、`/installed`
- [x] 1.4 更新 `src/layouts/DefaultLayout.vue`，加入左側 sidebar 導覽列，包含四個導覽項目

## 2. internal/ - Provider Service (pure Go, no Wails imports)

- [x] 2.1 建立 `internal/provider/service.go`，定義 `Service` struct 和 `ListProviders()` method
- [x] 2.2 實作 Claude Code (`~/.claude`) 和 GitHub Copilot (`~/.copilot`) 設定路徑的存在狀態檢查
- [x] 2.3 實作 `GetTokens()` 和 `SaveToken(provider, token string)`，讀寫 `~/.redshell/config.json`
- [x] 2.4 確認 `internal/provider/` 不含任何 `github.com/wailsapp` import

## 3. internal/ - Marketplace Service (pure Go, no Wails imports)

- [x] 3.1 建立 `internal/marketplace/service.go`，定義 `Service` struct
- [x] 3.2 實作 `List()` — 讀取 `~/.redshell/marketplace.json`
- [x] 3.3 實作 `Add(url string)` — 驗證 URL、生成 ID (`hostname::owner@repo`)、寫入 JSON
- [x] 3.4 實作 `Remove(id string)` — 從 JSON 移除指定 marketplace
- [x] 3.5 實作 `GenerateID(url string) string` 工具函數
- [x] 3.6 確認 `internal/marketplace/` 不含任何 `github.com/wailsapp` import

## 4. internal/ - Plugin Service (pure Go, no Wails imports)

- [x] 4.1 建立 `internal/plugin/service.go`，定義 `Service` struct
- [x] 4.2 實作 `FetchFromGitHub(repoURL, token string)` — 呼叫 GitHub REST API 取得 marketplace.json
- [x] 4.3 實作 `FetchFromGitLab(repoURL, token string)` — 呼叫 GitLab REST API
- [x] 4.4 實作 `FetchAll()` — 整合所有已登錄 marketplaces，回傳 `[]MarketplacePlugin`
- [x] 4.5 實作 `runProviderCmd(provider string, args []string) error` — 以 `os/exec` 呼叫 provider CLI 指令，capture stderr 回傳錯誤，若 CLI 未安裝 (ENOENT) 回傳明確訊息
- [x] 4.6 實作 `EnsureMarketplace(marketplaceURL, provider string)` — 先讀取 provider 設定檔檢查是否已登錄，若否則呼叫 `<provider> plugin marketplace add <url>` (對應 ai-tools 的 `ensureMarketplace`)
- [x] 4.7 實作 `Install(provider string, plugins []MarketplacePlugin, logFn func(string)) error` — 依序呼叫 `EnsureMarketplace` 再執行 `<provider> plugin install <installName>`，透過 `logFn` callback 回報每步進度
- [x] 4.8 實作 `ListInstalled(provider string)` — 唯讀方式讀取 `~/.claude/plugins/installed_plugins.json` 或 `~/.copilot/config.json`
- [x] 4.9 實作 `Uninstall(provider, pluginID string)` — 呼叫 `<provider> plugin uninstall <pluginID>`
- [x] 4.10 確認 `internal/plugin/` 不含任何 `github.com/wailsapp` import

## 5. app/ - Wails Binding Layer (thin wrappers over internal/)

- [x] 5.1 建立 `app/provider.go`，定義 `ProviderApp` struct，embed `*provider.Service`，exposed methods 委派給 service
- [x] 5.2 建立 `app/marketplace.go`，定義 `MarketplaceApp` struct，embed `*marketplace.Service`
- [x] 5.3 建立 `app/plugin.go`，定義 `PluginApp` struct，embed `*plugin.Service`；`Install` method 使用 `wails.EventsEmit` 將 `logFn` 的每行輸出即時推送到前端
- [x] 5.4 在 `main.go` 初始化各 `internal/` service，注入至對應 `app/` struct，透過 `wails.Bind()` 暴露給前端

## 6. Vue Frontend - Providers Page

- [x] 6.1 建立 `src/stores/provider.ts` Pinia store，管理 provider 狀態與 API token
- [x] 6.2 建立 `src/views/ProvidersView.vue`
- [x] 6.3 實作 provider 卡片元件 (`src/components/ui/ProviderCard.vue`)，顯示名稱、config 路徑、Configured/Not Configured daisyUI badge
- [x] 6.4 加入 API token 設定表單 (GitHub Token、GitLab Token 輸入欄)，使用 `AppButton` 儲存
- [x] 6.5 串接 `ProviderApp` Go bindings (透過 Pinia store action 呼叫)

## 7. Vue Frontend - Marketplaces Page

- [x] 7.1 建立 `src/stores/marketplace.ts` Pinia store，管理 marketplace 清單
- [x] 7.2 建立 `src/views/MarketplacesView.vue`
- [x] 7.3 實作 marketplace 卡片元件 (`src/components/ui/MarketplaceCard.vue`)，使用 `AppCard` 包裝
- [x] 7.4 實作新增 marketplace daisyUI modal (URL 輸入、submit)
- [x] 7.5 使用 `useConfirm` composable 實作 Remove 確認
- [x] 7.6 串接 `MarketplaceApp` Go bindings (透過 Pinia store action 呼叫)

## 8. Vue Frontend - Browse Plugins Page

- [x] 8.1 建立 `src/stores/plugin.ts` Pinia store，管理 plugin 清單與選取狀態
- [x] 8.2 建立 `src/views/BrowsePluginsView.vue`
- [x] 8.3 實作 plugin 卡片元件 (`src/components/ui/PluginCard.vue`)，顯示名稱、描述、作者、category badge、已安裝 badge
- [x] 8.4 實作 provider 過濾器 (daisyUI tabs 或 select)
- [x] 8.5 實作多選邏輯，選取時卡片顯示 daisyUI `ring` 選取框
- [x] 8.6 實作安裝確認 daisyUI modal (列出選取的 plugins)
- [x] 8.7 串接 `PluginApp.FetchAll` 和 `Install` bindings (透過 Pinia store action 呼叫)

## 9. Vue Frontend - Installed Plugins Page

- [x] 9.1 建立 `src/views/InstalledPluginsView.vue`
- [x] 9.2 實作 provider 切換使用 daisyUI tabs 元件
- [x] 9.3 實作已安裝 plugin 卡片元件 (`src/components/ui/InstalledPluginCard.vue`)，顯示名稱、版本
- [x] 9.4 使用 `useConfirm` composable 實作 Uninstall 確認
- [x] 9.5 串接 `PluginApp.ListInstalled` 和 `Uninstall` bindings (透過 Pinia store action 呼叫)

## 10. Build & Validation

- [x] 10.1 執行 `wails dev` 確認開發模式正常啟動
- [x] 10.2 測試 Marketplace CRUD 操作與 `~/.redshell/marketplace.json` 的資料互通
- [x] 10.3 測試 Plugin 安裝流程，確認透過 provider CLI 指令完成，安裝 log 即時顯示於 GUI
- [x] 10.4 測試 Uninstall 流程，確認呼叫 `<provider> plugin uninstall` 後 installed plugins 清單正確更新
- [x] 10.5 測試 provider CLI 未安裝情境，確認錯誤訊息明確提示使用者
- [x] 10.6 執行 `wails build` 確認可產出獨立 binary
- [x] 10.7 確認 `internal/` 套件可被獨立引用 (可寫一個最小 `cmd/cli/main.go` 呼叫 `internal/marketplace.Service.List()` 驗證無 Wails 依賴)
