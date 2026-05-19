# Changelog

本專案所有重大變更皆記錄於此檔案.

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-TW/1.1.0/),
版本號採用 [Semantic Versioning](https://semver.org/lang/zh-TW/).

## [0.9.0] - 2026-05-19

### Added

- Installed Plugins 頁面新增 "Update" 按鈕: 放在每張卡片
  "Uninstall" 按鈕左側, 點擊後呼叫對應 agent 的
  `<agent> plugin update <name>@<marketplace>` 指令更新該 plugin,
  完成後重新讀取 agent 的 installed-plugins 檔案以同步卡片 metadata.
  - 後端新增 `plugin.Service.UpdatePlugin(agentID, installName, logFn)`,
    沿用既有 `runAgentCmdStreaming` 將 CLI stdout 以 `[<agent>] ` prefix
    串流回前端, 並重用既有 `plugin:install-log` Wails event (不新增 channel).
    空字串 `installName` 與 disabled agent 都在 shell out 前被擋下.
  - Wails wrapper `app/plugin.go` 對應新增 `PluginApp.UpdatePlugin`,
    `frontend/wailsjs/go/app/PluginApp.{js,d.ts}` 與 `models.ts`
    一併更新.
  - 前端 `usePluginStore` 新增 `update(agentID, installName)` action
    與 per-plugin 的 `updatingPlugins: Set<string>` busy set,
    透過 `isPluginBusy(installName)` 暴露給卡片;
    `InstalledPluginCard.vue` 在進行中時同時 disable Update / Uninstall
    兩顆按鈕避免重複觸發, Update 按鈕額外顯示 spinner.
  - Update 為冪等操作, 不彈出 confirm modal; 成功 / 失敗皆以 toast 回報,
    錯誤訊息直接帶上 CLI stderr 內容.
- Installed Plugin 卡片副標題顯示安裝版本 (例如
  `claude-plugins-official · v1.0.0`):
  - `internal/plugin/service.go` 的 `InstalledPlugin` struct 新增
    `Version string` 欄位 (JSON `omitempty`).
  - `readClaudeInstalled` 改為解析
    `~/.claude/plugins/installed_plugins.json` v2 schema 每個 key 對應的
    install 陣列, 取 `scope: "user"` 的 `version` 為主, 其他 scope 為
    fallback; 字面值 `"unknown"` 視為無版本而忽略.
  - Copilot 的 `~/.copilot/config.json` schema 不記錄 plugin 版本,
    `Version` 留空, 副標題沿用既有「marketplace name」格式.

### Changed

- NSIS installer template `build/windows/installer/wails_tools.nsh`
  `INFO_PRODUCTVERSION` 由 `0.6.0` 補上至 `0.8.0`, 配合 0.8.0 installer
  release pipeline; 0.9.0 由 build pipeline 透過 ldflags 注入,
  template 預設值僅作為 fallback.

## [0.8.0] - 2026-05-09

### Added

- Installer-kind 自動更新路徑: 為 NSIS 安裝版本新增獨立的 in-app 更新流程,
  與 portable 版本共用同一個 release pipeline 與 `checksums.txt` 驗證,
  但安裝步驟改為 ShellExecute `runas` + NSIS silent flag (`/S`).
  - 新增 build-time discriminator `internal/updater.BuildKind`
    (`portable` 為預設, `installer` 透過
    `-ldflags "-X 'redshell/internal/updater.BuildKind=installer'"` 注入).
    `Updater.GetState()` 暴露 `buildKind` 欄位給前端.
  - 新增 `internal/updater/installer_install_windows.go`
    (build-tag `windows`) 與 `internal/updater/installer_install_other.go`
    (no-op stub), 透過 `ShellExecuteW` verb=`runas` 觸發單一 UAC 提示後執行
    silent install. 使用者按下取消 (`ERROR_CANCELLED` / errno 1223) 時清除
    in-progress flag 並 emit `updater:error` 事件.
  - `internal/updater/service.go` 在 `Start()` 與 `install()` 依 `BuildKind`
    分流: installer build 跳過 `IsWritable` 探測, 一律走 silent installer
    路徑; portable build 維持原本的 rename-trick swap.
  - Tray "Check for Updates" 選單與 `UpdatesTab.vue` 的 manual-required
    banner 改以 `BuildKind` 而非寫入權判斷; installer 版本一律顯示更新動作,
    並提示「更新會觸發 Windows UAC 視窗」.
- NSIS 安裝程式調整 (`build/windows/installer/project.nsi`):
  install section 開頭加入 `Sleep 2000` 給剛離開的 RedShell 釋放檔案鎖,
  silent 模式 (`/S`) 結束後**不會**自動重啟 RedShell, 以避免
  `RequestExecutionLevel admin` 把新程序也以 elevated token 啟動.
- `scripts/publish-wails.ps1` 新增 `-Kind portable|installer` 參數,
  installer 模式自動帶入 ldflags 並產生
  `RedShell-amd64-installer.exe`, 與 portable 資產共用同一份
  `checksums.txt`.

### Changed

- `internal/updater/cleanup.go` 一併清掉 installer build 殘留的
  `redshell-installer.new` / `*.partial` 檔案.

## [0.7.0] - 2026-05-08

### Added

- Session History 頁面 header 改成「session id + display name」雙列結構:
  - 主 suffix 改為**完整 session id** (UUID 不截斷), 緊鄰一個
    `AppCopyButton` (icon-only ghost button, 點擊後 icon 暫時切換並彈出
    「Copied」toast).
  - 當後端解析出與 session id 不同的 `displayName` 時, 於下一行以小字顯示
    rich title; 若 displayName 為空或等於 session id (fallback case)
    則整列省略.
  - 新增 `AppResumeButton` 配合後端 `SessionHistoryApp.ResumeSession`,
    透過 `cmd /c start "" pwsh -NoExit -NoProfile -Command "<agent>
    --resume <id>"` 在**該 session 的 project cwd** 開啟新 pwsh 視窗,
    並以 `cmd.Dir` (而非字串拼接) 傳入路徑避免 quoting 問題.
    session id 以 `^[A-Za-z0-9_-]+$` 嚴格驗證, cwd 若不存在或非目錄則回傳
    `ErrProjectCwdMissing`, 不開啟 terminal.
  - `internal/sessionhistory/terminal_other.go` 在非 Windows 平台回傳
    `ErrTerminalUnsupported` no-op.
- `frontend/src/stores/sessionViewPrefs.ts`: 新增 `wrap` 偏好
  (預設 `true`), 持久化於 `localStorage` `sessionView.wrap`,
  控制 `SessionEventList` 與 Hooks 詳細面板長字串是否換行.
- `internal/preferences` 新增 `AutoUpdate` 區塊
  (`enabled`, `intervalHours`, `source`, `githubRepo`, `gitlabHost`,
  `gitlabProject`, `skipVersion`, `lastCheckedAt`), 並接上 observer
  通知讓 updater service 在偏好變動時即時 reschedule.
- Windows 系統匣選單新增 "Check for Updates" 項目, 點擊開啟
  Settings -> Updates tab.
- 新增 `AppCopyButton`, `AppResumeButton`, `AppIcon` UI primitives,
  維持 daisyUI `btn`/`btn-circle` 邊界規則.

### Changed

- `OnBeforeClose` 在 `Updater.InProgress()` 為 `true` 時跳過 close-behavior
  prompt 與 minimize-to-tray 判斷, 讓 rename swap 期間可以乾淨退出.

## [0.6.1] - 2026-05-08

### Changed

- 暫時於 `UpdatesTab.vue` 中關閉 GitLab provider 的 UI 入口
  (radio button 與 side-by-side peek), 後端 `provider_gitlab` 程式碼保留
  以利後續恢復. Reason: 在已知 host 設定流程下 GitLab API auth / asset
  permalink 行為仍需更多打磨, 不阻擋 portable 版本 release.

## [0.6.0] - 2026-05-08

### Added

- Portable Windows 自動更新器 (`internal/updater/`,
  `app/updater.go`, `frontend/src/components/settings/UpdatesTab.vue`,
  `frontend/src/composables/useUpdater.ts`): 背景輪詢使用者選擇的
  release source (GitHub 或 GitLab), 以 semver 比對運行版本, 下載並驗證
  asset 後以 rename-trick 取代執行檔.
  - 排程: 啟動後 5 秒內若 `now - lastCheckedAt >= intervalHours` 即觸發
    第一次檢查, 之後依 1 / 6 / 12 / 24 / 168 小時 ticker 再檢; 偏好停用
    時完全不打 ticker. 手動 "Check for updates" 忽略 debounce.
  - GitHub provider: `GET https://api.github.com/repos/<owner>/<repo>/releases/latest`,
    Accept header 採 `application/octet-stream` 取資產二進位, 解出
    OS/arch 對應 asset 與 `checksums.txt` URL.
  - GitLab provider: `GET <host>/api/v4/projects/<URL-encoded(project)>/releases/permalink/latest`,
    解出與 GitHub 一致的 `Release` 結構.
  - 安全驗證: 強制下載 `checksums.txt` (`sha256sum` 格式), 以 streaming
    SHA-256 比對下載檔案; mismatch / sidecar 缺失 / asset 未列入 sidecar
    皆 abort 並 emit `updater:error`, **不會**回退到無驗證安裝.
  - Rename-trick swap (`internal/updater/rename_windows.go`): 下載到
    `redshell.exe.partial` -> verify -> 改名 `redshell.exe.new` -> 將執行中
    `redshell.exe` 改為 `*.old` -> `*.new` 改為 `redshell.exe` ->
    `exec.Command(...).Start()` detached -> `quitApp()`.
    Process 啟動時 `cleanup.go` 會嘗試刪掉殘留的 `*.old` / `*.partial`,
    失敗忽略.
  - 寫入權探測: portable build 若所在目錄不可寫 (例如放在
    `Program Files`), updater **不會**註冊 ticker, 改 emit
    `updater:manual-required` 由 UI 顯示「請改用 portable 下載」.
- Settings -> Updates tab: 切換來源 (radio), 選擇 interval, 顯示雙來源
  side-by-side 最新版本 (peek 不會變更 active source 或 `lastCheckedAt`),
  支援手動檢查與「Skipped」revoke. 偏好變更會立即 reschedule ticker.
- 可用更新行為: toast / banner 提供 `Update Now` / `Skip This Version`
  (寫入 `prefs.autoUpdate.skipVersion`) / `Later` (僅 dismiss) 三選擇.
- 前端事件 (Wails runtime emit): `updater:check-started`,
  `updater:available`, `updater:up-to-date`, `updater:download-progress`
  (節流 250ms), `updater:installed`, `updater:error`,
  `updater:manual-required`. `useUpdater` composable 統一訂閱.
- Provider 抽象: `internal/updater/types.go` 定義 `Provider` interface
  與 `InstallerAssetNameFor` helper, 測試以 `httptest.Server` + 注入
  base directory 完整覆蓋, 不依賴實際 `api.github.com` 或 `gitlab.com`.
- `golang.org/x/mod/semver` 加入 `go.mod`, 用於 prerelease-aware
  版本比對.

### Changed

- GitHub provider 下載 asset 時改採 `Accept: application/octet-stream`
  並改用 `assets[].url` 而非 `browser_download_url`, 避免 HTML redirect
  路徑造成 streaming hash mismatch.

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
