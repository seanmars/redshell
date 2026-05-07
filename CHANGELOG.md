# Changelog

本專案所有重大變更皆記錄於此檔案.

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-TW/1.1.0/),
版本號採用 [Semantic Versioning](https://semver.org/lang/zh-TW/).

## [0.4.0] - 2026-05-07

### Added

- Marketplaces 分頁新增 "Update" 按鈕: 並行呼叫已啟用 agent 的
  `<agent> plugin marketplace update` 指令, 更新 agent 自身的
  marketplace 註冊清單. 與既有的 "Refresh" (重新整理 RedShell 端
  `~/.redshell/.cache/` 的 git clone) 各司其職, 互不影響.
  - 後端新增 `plugin.UpdateAgentMarketplace` (單一 agent) 與
    `UpdateAgentMarketplaces` (批次包裝) service API, 對應
    Wails wrapper `app/plugin.go`.
  - 前端透過 `Promise.all` 同時觸發 claude 與 copilot 的呼叫,
    牆上時間取 `max(claude, copilot)` 而非 `sum(...)`.
  - 每個 agent 顯示一個 sticky info toast (`Updating <agent>...`),
    完成後置換為 success / error toast; CLI stdout 透過
    `plugin:install-log` Wails event 串流, 以 `[claude] ...` /
    `[copilot] ...` prefix 區分來源.
- Windows 系統匣 (system tray) 圖示: 開啟後 RedShell 於系統匣常駐,
  支援透過匣選單顯示 / 隱藏視窗, 並可勾選 "Close button minimizes
  to tray" 切換主視窗關閉按鈕的行為.
  - 首次按下關閉按鈕時彈出一次性對話框讓使用者選擇預設行為
    (`exit` 或 `minimize-to-tray`), 結果寫入
    `~/.redshell/preferences.json`. 此檔案與 agent 設定的
    `~/.redshell/settings.json` 分離, 兩者各自演進.
  - 非 Windows 平台以 `internal/tray` build tag 為 no-op,
    行為與先前一致.
- 後端工具 `runAgentCmdStreaming` (`internal/plugin/service.go`):
  將 agent CLI 子程序的 stdout 以 line-splitting writer 透傳給
  呼叫端 callback, 使 long-running CLI 的進度可即時送回前端;
  既有的 `runAgentCmd` (僅捕捉 stderr) 維持不變, `Install` /
  `Uninstall` 行為一致.
- 新增 `internal/preferences` 模組: 管理 `~/.redshell/preferences.json`
  shell 層級偏好, 目前僅含 `closeBehavior` 一欄.

## [0.3.0] - 2026-05-06

### Added

- Hooks 檢視頁 (`/hooks`): 唯讀檢視 Claude Code 與 Copilot CLI 各 agent
  目前生效的 hook 設定. 與 Sessions / Installed 同層, 採 per-agent tab +
  左 source / event 折疊樹 + 右 detail 兩 pane 配置.
  - Claude 來源: `~/.claude/settings.json` (User), `~/.claude/settings.local.json`
    (Local), 以及每個已安裝 plugin 的 `<installPath>/hooks/hooks.json` (Plugin),
    經由 `~/.claude/plugins/installed_plugins.json` v2 schema 列舉,
    `cache/` 與 `.git/hooks/` 不會被掃入.
  - Copilot 來源: v1 因 Copilot CLI hooks 為 per-project, 顯示 empty state
    說明 workspace 選擇為日後擴充, service 介面已預留 `ListOpts.Workspace`.
  - 詳細面板顯示完整絕對路徑 (不截斷), 依 handler type 分區的 resolved fields,
    以及 read-only pretty-printed raw JSON.
  - `disableAllHooks: true` 時於 agent tab 上方以 banner 標示來源檔案路徑,
    清單仍會完整顯示.
  - 跨 source 出現相同 command 字串時, detail 面板顯示
    `appears in N sources` 標籤; v1 不做去重.
  - 「Open settings file」按鈕透過既有 `os-path-opener` 在 OS 檔案管理員開啟.
- 後端模組 `internal/hooks/` (types, paths, parser_claude, parser_copilot,
  plugin_scan, service), 對應 Wails wrapper `app/hooks.go`.

## [0.2.0] - 2026-04-28

### Fixed

- 修正 `wails dev` 與 `wails build` 兩種模式下主題顏色不一致的問題.
  原因為 daisyUI 5 同時註冊內建 `light` / `dark` 與自訂 `@plugin "daisyui/theme"`
  覆蓋時, build 會額外輸出高特定性 selector
  (`:is(:root:has(input.theme-controller[value=light]:checked),...)`),
  使內建主題在 cascade 中蓋過自訂值. 改以 `themes: false` 停用內建主題,
  讓自訂 light / dark 成為唯一定義, 並將 `prefers-color-scheme: dark` 的
  fallback 透過 dark 主題的 `prefersdark: true` 補回.
- `index.html` 加入同步 inline script, 在 CSS 評估前先依 localStorage 與
  `prefers-color-scheme` 設定 `data-theme`, 避免 build 模式下首次 paint
  使用預設主題造成的閃爍.

## [0.1.0] - 2026-04-27

首個公開版本. RedShell 是一個以 Wails v2 打造的桌面應用,
提供 Claude Code (`claude` CLI) 與 GitHub Copilot (`copilot` CLI)
的 plugin marketplace 管理與 session history 檢視.

### Added

#### 使用者功能

- 首次啟動引導流程 (`/setup/agents`), 完成代理 (agent) 設定前會強制導向設定頁.
- Marketplace 管理 (設定頁 marketplaces 分頁):
  - 支援以 SSH, HTTPS 或 `host/owner/repo` 形式新增 marketplace.
  - 自動以 `git clone --depth=1` 建立快取於 `~/.redshell/.cache/`,
    註冊資料儲存於 `~/.redshell/marketplace.json`.
  - 支援逐一 marketplace 重新整理 (`git fetch --depth=1` + reset).
- Plugin 瀏覽 (`/browse`): 跨所有 marketplace 聚合 plugin,
  以單一卡片顯示, 並標註支援的 agent badge.
- Plugin 安裝 / 解除安裝 (`/installed`):
  - 安裝前自動執行 `EnsureMarketplace` (依 agent 自身設定判斷是否需註冊).
  - 安裝過程的 stdout / stderr 透過 Wails event (`plugin:install-log`)
    即時串流到前端.
  - 已安裝清單直接讀取各 agent 自有設定:
    - Claude: `~/.claude/plugins/installed_plugins.json`
    - Copilot: `~/.copilot/config.json`
- Session History 檢視 (`/sessions`): 唯讀模式, 不寫入或刪除任何 session 檔.
  - Claude Code: 依專案分組 (grouped) 呈現.
  - Copilot: 平鋪 (flat) 呈現.
  - 支援分頁載入事件 (events).
- 設定頁 (`/settings`): agents 與 marketplaces 兩個分頁.
- 主題切換: daisyUI light / dark, 設定持久化於 localStorage.
- 應用程式版本顯示 (讀取自 `version.json`).
- 在 OS 檔案總管中開啟指定路徑.

#### 開發者功能

- Wails v2 + Vue 3 (Composition API) + Tailwind 4 + daisyUI 5 技術棧.
- 後端分層: `internal/<domain>/service.go` 與 `app/<domain>.go` 切分,
  service 層可獨立單元測試, app 層僅負責 Wails context 與 event emit.
- 已支援的 internal services:
  `agent`, `marketplace`, `plugin`, `sessionhistory`, `osopen`, `sysproc`.
- 前端 UI primitive (`AppButton`, `AppCard`, `AppModal` 等)
  封裝 daisyUI class, 業務元件不直接使用 daisyUI component class.
- 前端組合式函式 (composables): `useConfirm`, `usePageTitle`,
  `useToast`, `usePluginInstaller`.
- Wails 綁定型別自動產生於 `frontend/wailsjs/go/app/`,
  以 `@wailsjs/*` alias 引用.
- 提供 PowerShell 發佈腳本 `scripts/publish-wails.ps1`.
- `cmd/cli` 除錯工具: 將 `~/.redshell/marketplace.json` 以 JSON 印出.
