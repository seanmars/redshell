## Why

使用者在 Marketplaces 頁面新增 marketplace 後, 切換到 Browse Plugins 仍會看到 "No plugins found. Add a marketplace." 的訊息, 讓人誤以為沒有任何 marketplace 被註冊. 目前 Browse Plugins 畫面只反映彙整後的 plugin 清單, 完全沒有讀取 marketplace registry, 所以只要抓取失敗, 遠端倉庫沒有 `plugins/` 目錄, 或狀態過期, 都會被壓縮成同一個誤導性的 empty state. 此外, 上方的 provider 分頁 (All / Claude Code / GitHub Copilot) 只是增加操作成本, 並未帶來對應的資訊價值, 因為每張 plugin card 本身已經標示了目標 provider.

## What Changes

- Browse Plugins view 會讀取 marketplace registry, 並為每個已註冊的 marketplace 渲染一個區塊, 區塊內列出該 marketplace 的 plugin cards.
- 拆分兩種 empty state: 當沒有任何 marketplace 被註冊時, 顯示導向 Marketplaces 頁面的提示; 當 marketplace 存在但某一個沒有回傳 plugin 時, 顯示該區塊內部的空白或錯誤訊息, 讓使用者能辨識是哪個來源出狀況.
- **BREAKING (UX)**: 移除 Browse Plugins 上方的 All / Claude Code / GitHub Copilot provider 分頁, 所有 plugin 直接依 marketplace 分組一次呈現.
- `usePluginStore` 在現有的 `FetchAllResult` 上新增以 marketplace 分組的 `pluginsByMarketplace` getter, 不需要變更後端 API.
- `FetchAllResult.Errors` 中屬於各別 marketplace 的錯誤, 改成顯示在對應的 marketplace 區塊內, 不再堆在頁面底部的警告列表.

## Capabilities

### New Capabilities
<!-- 無. 此次變更只修改既有 capability. -->

### Modified Capabilities
- `plugin-browser`: 以 marketplace 分組瀏覽取代 provider-filter 式瀏覽, 並重新定義 empty state 規則, 讓 "marketplace 是否存在" 與 "plugin 是否存在" 成為兩個獨立狀態.

## Impact

- Frontend: `frontend/src/views/BrowsePluginsView.vue`, `frontend/src/stores/plugin.ts`, 以及可能新增的小型元件 (例如 `MarketplaceSection.vue`) 或在 view 裡內嵌分組邏輯.
- 狀態: Browse Plugins 除了 `usePluginStore` 之外還會依賴 `useMarketplaceStore`, 兩者都在 mount 時抓取.
- Backend (`internal/plugin/service.go`, `app/plugin.go`): 無 API 變動. `FetchAllResult` 的每一筆 plugin 已經帶有 `marketplace` 與 `marketplaceName` 欄位, 前端分組即可完成.
- Specs: `openspec/specs/plugin-browser/spec.md` 中關於 empty state 與 provider filter 的 requirement 會被取代.
- 沒有資料搬移或持久化資料格式的變動.
