# RedShell Updater 設計文件

本文件記錄 RedShell 的「自動更新」(Auto-Updater) 全部流程、規則與架構, 對應 `internal/updater/`, `app/updater.go`, 以及前端 `useUpdater` / `UpdatesTab` / `UpdateAvailableBanner`.

> 適用範圍: Windows 可攜版 (portable) 與 NSIS installer 安裝版二進位. Portable 走 rename trick, installer 走 elevated silent-install (UAC). 由建置時 `BuildKind` ldflag 區分, 詳見 §23. macOS / Linux 在 v1 不支援, 由 build tag 提供 no-op stub.

---

## 1. 整體架構

三層分離, 由下往上呼叫, 上層不會被下層感知:

```
+--------------------------------------------------+
|  Frontend (Vue 3)                                |
|   - composables/useUpdater.ts                    |
|   - components/system/UpdateAvailableBanner.vue  |
|   - components/settings/UpdatesTab.vue           |
+----------------------+---------------------------+
                       | EventsOn / Wails bindings
+----------------------v---------------------------+
|  app/updater.go  (UpdaterApp)                    |
|   - 持有 Wails ctx                               |
|   - emit  -> runtime.EventsEmit                  |
|   - quit  -> runtime.Quit                        |
+----------------------+---------------------------+
                       | EventEmitter / SwapFunc / SpawnFunc
+----------------------v---------------------------+
|  internal/updater  (Service)                     |
|   - prefs / providers / runLoop / install        |
|   - 完全不含 Wails 依賴, 可單元測試              |
+--------------------------------------------------+
```

* **`internal/updater/service.go`**: 排程、狀態、安裝管線. 透過 `EventEmitter` (函式) 與 `SwapFunc` / `SpawnFunc` 與外界互動, 測試可全部注入假實作.
* **`app/updater.go`**: 把 Wails `runtime.EventsEmit` / `runtime.Quit` 接給 service, 並把 service 的 public API 包成 Wails bindings.
* **前端**: 透過 `EventsOn` 訂閱 `updater:*` 事件, 不主動輪詢 backend.

---

## 2. 偏好設定 (Preferences) Schema

存放於 `~/.redshell/preferences.json`, 由 `internal/preferences/service.go` 管理. 與 `closeBehavior` 同檔案, 但兩者各自獨立演進.

```jsonc
{
  "closeBehavior": "minimize-to-tray",
  "autoUpdate": {
    "enabled": true,
    "intervalHours": 6,
    "source": "github",
    "githubRepo": "seanmars/redshell",
    "gitlabHost": "https://gitlab.com",
    "gitlabProject": "seanmars/redshell",
    "skipVersion": "",
    "lastCheckedAt": "2026-05-07T00:00:00Z"
  }
}
```

### 欄位規則

| 欄位 | 預設值 | 規則 |
|---|---|---|
| `enabled` | `true` | 為 `false` 時暫停所有 ticker 排程, 不消耗 polling 配額. |
| `intervalHours` | `6` | 僅允許 `{1, 6, 12, 24, 168}`, 違反則 setter 拒絕. 防止使用者設 `0` 而 DoS 自家 release host. |
| `source` | `"github"` | 僅允許 `"github"` 或 `"gitlab"`, 其餘拒絕. |
| `githubRepo` | `seanmars/redshell` | `<owner>/<repo>`, 不可為空、不可含空白. v1 不在 UI 暴露, 由 fork / 自架 GitLab 使用者直接編輯 JSON. |
| `gitlabHost` | `https://gitlab.com` | 必須是合法 URL 且 scheme=`https`. |
| `gitlabProject` | `seanmars/redshell` | 同 `githubRepo` 規則. |
| `skipVersion` | `""` | 使用者「跳過」的版本字串. 為空表示沒有跳過任何版本. |
| `lastCheckedAt` | `""` | RFC 3339 字串, 由 service 在每次檢查 (成功或失敗) 後寫入. |

### Observer 通知策略 (重要)

`preferences.Service.OnChange` 觀察者會在以下變更發生時觸發, 用以重啟 ticker:

* `Enabled`, `IntervalHours`, `Source`, `SkipVersion` 變更 -> **通知**
* `LastCheckedAt`, `GithubRepo`, `GitlabHost`, `GitlabProject` 變更 -> **不通知** (避免每次檢查完成觸發 loop reset).

實作位置: `autoUpdateObservableChange` 函式比對前後值決定是否通知.

### 預設值補完

讀檔時 (`readLocked`) 若 `autoUpdate` 不存在或欄位缺失, 會用 `applyAutoUpdateDefaults` 補上預設值; 若值仍違反 validation, 直接返回 error 讓使用者自行修正 JSON, 不會「靜默修復」.

---

## 3. 核心型別

`internal/updater/types.go`:

```go
type Release struct {
    Version      string    // 例 "v0.5.0", 與 git tag 一致, 保留前綴 v
    PublishedAt  time.Time
    Notes        string    // raw markdown, 前端負責渲染
    AssetURL     string    // 二進位資產 URL
    AssetName    string    // 例 "redshell-windows-amd64.exe"
    AssetSize    int64
    ChecksumsURL string    // checksums.txt URL
}

type Provider interface {
    Name() string                                     // "github" | "gitlab"
    LatestRelease(ctx context.Context) (Release, error)
}
```

### 資產檔名規則

```go
func AssetNameFor(goos, goarch string) string {
    suffix := ""
    if goos == "windows" {
        suffix = ".exe"
    }
    return "redshell-" + goos + "-" + goarch + suffix
}
```

* Windows AMD64 -> `redshell-windows-amd64.exe`
* `checksums.txt` 是固定常數 `ChecksumsAssetName = "checksums.txt"`.

兩個 provider 都會在 release assets 裡尋找上述兩個檔名, 缺一就回傳 `ErrAssetNotFound` 或 `ErrChecksumsNotFound`, 該 release 視為不可用.

---

## 4. 版本比較

`internal/updater/version.go` 包裝 `golang.org/x/mod/semver`:

* `canonicalSemver` 自動補上前綴 `v`.
* `Compare(a, b)` 傳回 `-1 / 0 / 1`, 兩邊皆非 semver 則視為相等; 一邊有效一邊無效, 無效那邊較小.
* 採用 semver 排序, 因此 prerelease 順序正確: `v0.5.0-rc1 < v0.5.0`.

「有更新可用」的判定: `Compare(latest, running) > 0` 且 `latest != skipVersion`.

---

## 5. Provider

### 5.1 GitHub Provider (`provider_github.go`)

* Endpoint: `GET https://api.github.com/repos/<owner>/<repo>/releases/latest`
* Headers:
  * `Accept: application/vnd.github+json`
  * `X-GitHub-Api-Version: 2022-11-28`
  * `If-None-Match: <last ETag>` (有快取時)
* Response 304 Not Modified -> 回傳上次快取的 `Release`, 不消耗 rate-limit 配額.
* Response 2xx -> 解析 `assets[]`, 取 `Name == AssetNameFor(GOOS,GOARCH)` 與 `Name == "checksums.txt"`, 兩者 URL 取 `assets[i].URL` (即 API URL); 配合 `Accept: application/octet-stream` 下載.
* 任一資產缺少 -> 包裝 `ErrAssetNotFound` / `ErrChecksumsNotFound`.

### 5.2 GitLab Provider (`provider_gitlab.go`)

* Endpoint: `GET <host>/api/v4/projects/<URL-encoded(project)>/releases/permalink/latest`
  * project 必須包含 `/` (例 `group/project`); host 必須是 http/https 合法 URL, 否則建構失敗.
* Headers:
  * `Accept: application/json`
  * `If-None-Match` 與 GitHub 相同邏輯.
* Response 解析 `assets.links[]`, 比對 `Name`. 下載 URL 偏好 `direct_asset_url`, 缺則 fallback `URL`.

### 5.3 共通行為

兩個 provider 結構相同:

* 內含 `lastETag` + `lastRelease` + `sync.Mutex`, 304 時直接回傳快取.
* `LatestRelease` 失敗只回 error, 不會自動重試也不會跨來源 fallback.

---

## 6. 服務生命週期

`internal/updater/service.go`:

### 6.1 建構

```go
NewService(prefs, runningVersion, exePath, Options{}) (*Service, error)
```

* 必填: `prefs`, `exePath`. 預設 HTTP timeout 30s.
* `runningVersion` 由 `main.go` 的 `GetAppVersion()` 提供, 來源是 `//go:embed wails.json` 後解析其 `info.productVersion` 欄位. 因此 **發版時要改版本號, 改 `wails.json` 即可**, 不需要動 Go 程式碼.
* `exePath` 由 `os.Executable()` 取得, 給 swap / cleanup / install_dir 三處共用.
* `Options` 可注入 `HTTPClient`, `Now`, `Swap`, `Spawn` (測試時用).
* 同時提供 `NewServiceWithProviders` 讓測試直接注入由 `httptest.Server` 構成的 provider map, 不接觸真實 GitHub/GitLab.

### 6.2 Start

由 `app/updater.go` 在 Wails `OnStartup` 呼叫:

1. **Cleanup stale**: 刪除 `<exe>.old`, `<exe>.new`, `<exe>.new.partial`, `<exe>.partial` 任何前次中斷殘留. 失敗會 emit `updater:error` (stage=cleanup), 但不阻擋啟動.
2. **可寫性偵測**: 在 `filepath.Dir(exePath)` 嘗試建立 + 刪除一個 `redshell-update-probe-*` 暫存檔. 失敗 -> emit `updater:manual-required` 並 **直接 return**, 不啟動 ticker. (常見原因: NSIS 裝在 `Program Files`.)
3. **註冊 prefs observer** (見 §6.4).
4. **啟動 runLoop goroutine**, 帶 cancellable context.

### 6.3 runLoop 排程

主迴圈處理三個訊號:

* **Timer 觸發 (週期 ticker)**: 若 `enabled` 為 `true`, 跑一次 `runCheck("ticker")`, 之後 reset timer 為 `intervalHours`.
* **`manualCheckCh`**: 來自前端 `CheckNow()` 或 tray 的「Check for Updates」. 立即跑 `runCheck("manual")`, 然後 reset timer (避免「剛 manual 完又 ticker 跑一次」).
* **`prefsChangedCh`**: 偏好變動 (僅含 observable 欄位). 比對 `source` 是否變更, 若 source 改變且 `enabled` 為 true, 立即跑 `runCheck("source-change")`. 不論 source 改不改, 都會用新的 interval reset timer.

#### Startup-time check

`maybeFireStartupCheck` 在 runLoop 開始 timer 之前先判斷:

* `enabled` 為 false -> 跳過.
* `lastCheckedAt` 為空, 或解析失敗, 或 `now - lastCheckedAt >= intervalHours` -> 立即跑 `runCheck("startup")`.
* 否則跳過, 等下一個 timer.

> 這與 spec 一致: 應用程式重啟不會把使用者的「冷卻時間」歸零.

### 6.4 偏好變動橋接

```go
s.prefs.OnChange(s.onPrefsChange)
```

`onPrefsChange` 只把訊號丟進 `prefsChangedCh` (non-blocking), 真正處理在 runLoop. 這樣可避免在 prefs callback 裡持有 lock 的情況下又呼叫 service 內部, 造成死結.

### 6.5 Stop

`Stop()` 取消 runLoop ctx 並等待 `loopDone`. 由 `OnShutdown` 呼叫. 多次呼叫安全.

---

## 7. 檢查流程 `runCheck`

每次執行步驟:

1. emit `updater:check-started` `{ source, trigger }`. `trigger` 為 `"startup" | "ticker" | "manual" | "source-change"`.
2. 從 `s.providers[au.Source]` 找 active provider, 找不到 emit `updater:error` (stage=check).
3. 呼叫 `provider.LatestRelease(ctx)`.
4. **無論成功失敗** 都更新 `prefs.SetAutoUpdateLastCheckedAt(now)`. 這個 setter 不觸發 observer.
5. 失敗 -> emit `updater:error` (stage=check, source, message).
6. 成功且 `Compare(latest, running) <= 0` -> 清空 `lastResult`, emit `updater:up-to-date` `{ source, latestVersion, runningVersion }`.
7. 成功且 `Compare(latest, running) > 0` -> 快取 `lastResult = &release`.
   * 若 `latest == skipVersion` -> **不 emit** `updater:available`, 但 `lastResult` 仍保留 (供 `GetState` 顯示「Skipped」標籤).
   * 否則 emit `updater:available` 並把整個 `Release` payload 帶過去.

### 跨來源策略

失敗時 **不會** 自動 fallback 到另一個來源. 設計理由: 靜默切換會掩蓋連線問題或 publisher 漏發. 使用者透過 Settings 自行切換.

---

## 8. Peek (兩來源並行查詢)

`PeekBothSources(ctx)` 是給 Settings UI 用的 read-only 操作:

* 對所有 provider 並行呼叫 `LatestRelease`.
* 任一成功 -> 填入 `PeekResult.GitHub` / `PeekResult.GitLab`.
* 任一失敗 -> 填入 `PeekResult.Errors[name]`.
* **不修改** `lastCheckedAt` 也不更動 active source 狀態, 因此可任意呼叫.

---

## 9. 安裝流程 `install`

由前端按下「Update Now」(banner / UpdatesTab) 或對應的 Wails binding `InstallAvailable()` 觸發. 一進入即:

1. **互斥檢查**: 入口處只做 `inProgress.Load()`, 為 true 則回 error `update already in progress`. 真正的 `Store(true)` 發生在 verify 通過、即將 swap 前 (見步驟 8), 因此理論上 Load 與 Store 之間存在 TOCTOU window. 同時觸發兩個 install 的 race 主要靠前端 button loading state 防護, backend 沒有原子 CAS.
2. **並行下載**:
   * `Download(ctx, httpClient, rel.AssetURL, exePath+".new", rel.AssetSize, progressFn)` 串流寫入 `<exe>.new.partial`, 同步計算 SHA-256.
   * `downloadBytes(ctx, httpClient, rel.ChecksumsURL)` 一次讀進記憶體 (上限 1 MiB).
3. **進度事件**: `Download` 內 progress writer 至多每 250 ms emit `updater:download-progress` `{ bytesDownloaded, totalBytes }`. `totalBytes` 優先用 `rel.AssetSize`, fallback 為 `Content-Length`.
4. 任一下載失敗 -> 清掉 `<exe>.new` 與 `.partial`, emit `updater:error` (stage=download).
5. **解析 checksums**: `ParseChecksums` 接受 `<hex>  <filename>` 格式 (相容 `sha256sum`):
   * 跳過空行與 `#` 開頭註解.
   * `hash` 必須為 64 字元小寫 hex. 大寫會被 `ToLower`.
   * `name` 接受 `*` 前綴並去除 (相容 `sha256sum -b`).
   * 整個檔案 zero entries -> error.
6. **Lookup**: `expected = checksums[rel.AssetName]`. 不存在 -> emit `updater:error` (stage=verify), 中止.
7. **比對**: 串流 hash 不等於 expected -> 刪除 `.new`, emit `updater:error` (stage=verify, message=`checksum mismatch...`).
8. **進入 in-progress**: `inProgress.Store(true)`. 這個 flag 給 `OnBeforeClose` 看, 用以略過 close behavior prompt (見 §11).
9. **Swap** (Windows: rename trick):
   * `<exe>` -> `<exe>.old`
   * `<exe>.new` -> `<exe>`
   * 第二步失敗會嘗試 rollback 第一步; rollback 也失敗則回傳合併 error.
   * 失敗 -> 把 `inProgress` 設回 `false`, 刪除 `<exe>.new`, emit `updater:error` (stage=rename).
10. **Spawn**: `exec.Command(<exe>).Start()`, **stdin/stdout/stderr 全部 nil**, 然後 `cmd.Process.Release()` 讓子 process detach. 失敗 emit `updater:error` (stage=spawn). (此時 `inProgress` 維持 true, swap 已完成.)
11. emit `updater:installed` `{ version }`.
12. 呼叫 `quitApp()` -> Wails `runtime.Quit(ctx)`. 子 process 在數百毫秒內接手.

`Download` 細節:

* 寫入 `destPath + ".partial"`, copy 完成後 `os.Rename(partialPath, destPath)`. 若 copy 過程出錯, defer 會自動關檔並刪除 `.partial`, 不會留下半成品命名為 `.new`.
* 採 `io.MultiWriter(file, hasher, progressWriter)`, 一次掃過資料.

---

## 10. Windows Rename Trick

`internal/updater/rename_windows.go` (build tag `//go:build windows`):

```go
func Swap(currentPath, newPath string) error {
    oldPath := currentPath + ".old"
    _ = os.Remove(oldPath)                          // 清舊 .old
    if err := os.Rename(currentPath, oldPath); err != nil {
        return ...
    }
    if err := os.Rename(newPath, currentPath); err != nil {
        if rb := os.Rename(oldPath, currentPath); rb != nil {
            return wrapped error
        }
        return ...
    }
    return nil
}
```

關鍵事實:

* Windows 自 Vista 起允許 rename **正在執行** 的 `.exe`, 但禁止 delete / overwrite. 因此整個 rename trick 完全可行, 且不需 admin 權限.
* `rename_other.go` (`//go:build !windows`) 直接回 `ErrPlatformUnsupported`. 服務層在這個情境下會把它表面化為 UI 訊息.

---

## 11. 殘留檔清理

`internal/updater/cleanup.go` 會在每次 Start 時呼叫 `CleanupStale(exePath)`, 刪除以下相鄰檔案:

| 檔名 | 來源 |
|---|---|
| `<exe>.old`                            | 上一次 swap 留下的舊版二進位. 新 process 啟動時舊 process 已退出, 因此可刪. |
| `<exe>.new`                            | 已驗證但未 swap 的下載 (例: post-verify crash). |
| `<exe>.new.partial`                    | 中斷中的 portable 下載. |
| `<exe>.partial`                        | 防禦性: 舊版本可能直接寫到這個檔名. |
| `%TEMP%\redshell-installer.exe`         | 已驗證或已 spawn 但仍留下的 installer payload (UAC 拒絕後 / installer 完成後). |
| `%TEMP%\redshell-installer.exe.partial` | 中斷中的 installer 下載. |
| `%TEMP%\redshell-installer.new`         | 舊版本 (副檔名 bug 修掉之前) 留下的, 防禦性清掉. |
| `%TEMP%\redshell-installer.new.partial` | 同上. |

`os.ErrNotExist` 一律靜默忽略; 其他 error 則 bubble 出來, 由 service 包成 `updater:error` (stage=cleanup) emit, 但 **不阻擋啟動**.

---

## 12. 安裝目錄可寫性偵測

`internal/updater/install_dir.go` 的 `IsWritable(dir)`:

* `os.Stat(dir)` 確認是 directory.
* 嘗試建立 `redshell-update-probe-<unix-nano>` 檔, 寫入後刪除, 任一失敗回 `false`.

服務層在 Start 時, 如果 `IsWritable(filepath.Dir(s.exePath)) == false`:

* **不註冊 ticker**, runLoop 也不啟動.
* emit `updater:manual-required` `{ reason, exePath }`.
* `GetState().ManualRequired = true`, 前端據此把 UI 改為「This is an installed build...」.

`main.go` 在 tray 啟動前也會問 `updaterApp.ManualRequired()`, 為 true 則 **不註冊** tray「Check for Updates」項目.

> 用 write probe 而非讀 NSIS uninstall registry, 是為了一個泛用偵測, 不被將來換 installer 影響.

---

## 13. Wails 綁定 (`app/updater.go`)

`UpdaterApp` 暴露給前端的方法 (TypeScript bindings 在 `frontend/wailsjs/go/app/UpdaterApp.d.ts`):

| Binding | 行為 |
|---|---|
| `Startup(ctx)` (lifecycle) | 由 `main.go` 的 `OnStartup` 呼叫, 啟動 service. |
| `CheckNow()` | 對 `manualCheckCh` 送訊號. 已有未處理訊號則直接 return (non-blocking). |
| `PeekBothSources()` | 兩來源並行查詢, 不影響 active state. |
| `InstallAvailable()` | 對 `lastResult` 開始 install pipeline. `lastResult` 為 nil 時回 `no release available to install`. |
| `SkipVersion(version)` | 設定 `prefs.autoUpdate.skipVersion`. 觸發 observer, runLoop 會 reset. |
| `Unskip()` | 等同 `SkipVersion("")`. |
| `GetState()` | 取得 `State` 快照給 Settings UI 渲染. |
| `InProgress()` | 給 `OnBeforeClose` 用, 不對前端開放成「正常」UI 路徑. |
| `HandleTrayOpen()` | tray「Check for Updates」按下時觸發: emit `tray:open-updates` (前端跳到 `/settings?tab=updates`) + `CheckNow`. |
| `ManualRequired()` | 給 `main.go` 用以決定是否註冊 tray 項目. |

`State` 結構:

```go
type State struct {
    Enabled         bool
    Source          string
    IntervalHours   int
    RunningVersion  string
    LastCheckedAt   string
    LatestAvailable *Release
    SkipVersion     string
    InProgress      bool
    ManualRequired  bool
}
```

---

## 14. 前端事件清單

由 `app/updater.go` (透過 service) emit, 前端在 `useUpdater` 註冊 `EventsOn`:

| 事件 | Payload | 觸發時機 |
|---|---|---|
| `updater:check-started`     | `{ source, trigger }` | 每次 runCheck 入口 |
| `updater:available`         | `Release` | 有非 skip 的較新版本 |
| `updater:up-to-date`        | `{ source, latestVersion, runningVersion }` | latest <= running |
| `updater:download-progress` | `{ bytesDownloaded, totalBytes }` | 至多 250ms 一次 |
| `updater:installed`         | `{ version }` | swap + spawn 完成, 即將 quit |
| `updater:error`             | `{ stage, message }` | 任一階段失敗; `stage` 為 `check` / `download` / `verify` / `rename` / `spawn` / `cleanup` |
| `updater:manual-required`   | `{ reason, exePath }` | 安裝目錄不可寫 |
| `tray:open-updates`         | (no payload) | tray「Check for Updates」按下, 前端應跳到 `/settings?tab=updates` |

> Skipped-version 情境下不 emit `updater:available`, 但 `GetState()` 依然回傳 `LatestAvailable`, 因此 Settings UI 仍會顯示版本號加上「Skipped」徽章.

---

## 15. 前端 UI

### 15.1 `useUpdater` composable

* **單例 ref 設計**: 模組層級 `status / state / peek / error / progress / manualRequired` 為共享 ref, 所有 component 看到同一份狀態; 第一次呼叫時才註冊 `EventsOn` 訂閱 (`bootstrapped` flag 防重複).
* `status` 為 `'idle' | 'checking' | 'available' | 'up-to-date' | 'downloading' | 'installing' | 'installed' | 'error'`. 狀態轉換由 `updater:*` 事件驅動.
* 對外提供 `refreshState`, `checkNow`, `peekBoth`, `install`, `skip`, `unskip`.
* `error` 訊息格式為 `[stage] message`, 跟 backend payload 對齊.

### 15.2 `UpdateAvailableBanner.vue`

掛在 `DefaultLayout` 頂部, 條件: `release != null && !dismissed && release.version !== skipVersion`. 提供四個動作:

* **Update now** -> 呼叫 `install()` 進入下載流程.
* **View details** -> `router.push({ path: '/settings', query: { tab: 'updates' } })`.
* **Skip** -> `skip(release.version)`, 持久化, 之後同版本不再顯示 (除非 unskip).
* **Later** -> 設 `dismissed = true`, 僅本次 session 隱藏; 下次 ticker 還會再 emit.

### 15.3 `UpdatesTab.vue` (Settings 內)

* 上半: enable toggle, interval `<select>` (1/6/12/24/168 hours), 顯示 running version 與 last checked time.
* 中段: 兩張卡片並排顯示 GitHub / GitLab 各自的 latest peek; 每張卡片有「Use this」按鈕切換 active source.
* 下半: 「Check now」 + 「Update to vX.Y.Z」, 顯示 latest version, 提供「Skip this version」/「Unskip」.
* manual-required 時最上方顯示警告 alert, 且把 toggle 與 interval 設為 disabled.

### 15.4 `frontend/src/stores/preferences.ts`

包裝 backend 的 `AppPreferencesApp` bindings, 提供 reactive `autoUpdate` 並暴露 setter (`setAutoUpdateEnabled`, `setAutoUpdateInterval`, `setAutoUpdateSource` 等). UI 透過 store 操作, 避免直接呼叫 Wails binding.

`AUTO_UPDATE_INTERVALS = [1, 6, 12, 24, 168]` 是前後端共識的允許值.

---

## 16. 系統匣整合 (Windows)

`internal/tray/tray_windows.go` 在 `onReady` 建構選單時, 若 `m.checkForUpdates != nil` 才會加入「Check for Updates」項目; nil 時整個項目不存在.

`main.go` 的接線:

```go
if updaterApp != nil && !updaterApp.ManualRequired() {
    trayMgr.SetCheckForUpdates(updaterApp.HandleTrayOpen)
}
```

* Manual-required 時跳過註冊, 所以 installer 版本不會看到這個項目.
* 點擊行為: `ShowWindow()` -> 呼叫 `HandleTrayOpen` -> emit `tray:open-updates` + `CheckNow`.

非 Windows: `internal/tray/tray_other.go` stub 提供同樣的介面但什麼都不做.

---

## 17. 與 `OnBeforeClose` / Close Behavior 的互斥

`main.go`:

```go
OnBeforeClose: func(ctx context.Context) bool {
    if updaterApp != nil && updaterApp.InProgress() {
        return false  // 允許關閉, 不問偏好
    }
    return preferencesApp.HandleBeforeClose(ctx)
}
```

* 一旦進入 `install()` 的 swap 階段, `inProgress` flag 會被設為 true.
* 之後 `runtime.Quit(ctx)` 觸發 `OnBeforeClose`, 第一個 if 立即 return false (= 允許關閉), 不會去問 close-behavior 偏好, 也不會 emit `tray:close-behavior-prompt` 事件. 子 process 接手.
* 這個機制與既有的 explicit-quit pattern 一致 (tray「Quit RedShell」也類似).

---

## 18. 可測試性

`internal/updater` 的測試策略 (參考 `service_test.go`, `provider_*_test.go`, `download_test.go` 等):

* **Provider**: 用 `httptest.Server` 餵 `testdata/github_latest.json` 等 fixture, 測 URL 構造、JSON 解析、ETag 流程、缺失資產.
* **Service**: `NewServiceWithProviders` 注入假 providers 與假 swap/spawn 函式, 測排程、startup debounce、source 切換、skip 抑制 emit、install 失敗階段對應的事件.
* **Download**: `httptest` 餵 byte stream, 確認 SHA, atomic rename, progress callback.
* **Rename**: `rename_windows_test.go` 用 build tag, 在 temp dir 模擬三個檔案做 swap 斷言.
* **Cleanup**: 表格驅動驗證 `<exe>.old/.new/.new.partial/.partial` 全部刪除.

**禁忌**: 測試不可呼叫真實 `os.Executable()` 或 `os.UserHomeDir()`, 不可走真實 GitHub/GitLab 端點. 所有 IO 都應透過注入點. 注意 `preferences.NewServiceWithPath` 而非 `NewService`, 才不會去碰真正的 home dir.

---

## 19. 發版流程 (Release Workflow)

> 來源於 `add-portable-auto-updater/tasks.md` §11.

每次發版必須產出三個檔案並 **同步上傳到 GitHub release 與 GitLab release** 同一個 tag:

```
redshell-windows-amd64.exe       # 可攜版二進位
RedShell-amd64-installer.exe     # NSIS installer
checksums.txt                    # sha256sum 相容格式
```

`checksums.txt` 製作:

```sh
sha256sum redshell-windows-amd64.exe RedShell-amd64-installer.exe > checksums.txt
```

格式必須能被 `ParseChecksums` 接受:

```
<64-char-lowercase-hex>  <filename>
<64-char-lowercase-hex>  *<filename>      # 也接受, * 會被去掉
# comment lines ignored
```

漏掉 `checksums.txt` 會導致所有 user 的更新流程在 verify 階段中止, 不會「best-effort 安裝」. 這是非協商規則.

---

## 20. 不在範圍 (v1)

* macOS / Linux 的 portable replacement (`rename_other.go` 直接回 `ErrPlatformUnsupported`).
* Differential / binary-patch 更新.
* Beta / nightly / canary release channels (僅 `latest`).
* 失敗自動 rollback (失敗時 `.old` 與原檔同時保留, 文件記載恢復方法為刪 `.old`).
* GitHub authenticated request / 私人 GitLab token (60 req/hr unauthenticated 已足).
* 程式碼簽章 (整體完整性僅靠 `checksums.txt`).
* Tray icon 上的「有更新」徽章 overlay.

---

## 21. 風險與已知緩解

| 風險 | 緩解 |
|---|---|
| 無簽章 -> SmartScreen / AV 警示 | 下載先寫 `.partial`, 通過 SHA-256 才 rename 為 `.new`, 縮短 AV 掃描視窗. |
| swap 期間兩個 process 同時存在 | 目前無 single-instance lock, 影響微小; 若日後加 lock, 應透過 spawn 時帶 `--from-update` argv 區分. |
| AV 鎖住新檔導致 rename 失敗 | 失敗時 emit error, 保留 `.partial` 與原檔, 由使用者重試. |
| Publisher 一邊發一邊不發 | Settings tab 並排顯示兩來源 latest, 使用者自行切換. |
| 帳號被攻陷, 同時換二進位與 checksum | 已知無法靠 updater 解決; 仰賴 token 範圍與 branch protection. |
| GitHub rate limit (60 req/hr 匿名) | interval >= 1h, 加上 `If-None-Match`/304 已遠低於配額. |
| 磁碟滿 / 中斷下載 | `.partial` 殘留, 下次啟動 `CleanupStale` 刪除. |

---

## 22. 快速對照表 (給維護者)

* 新增第三個 release 來源: 在 `internal/updater/` 加 `provider_<name>.go`, 在 `service.go` 的 `buildProviders` 註冊, `preferences` 加常數與 setter, 前端 `UpdatesTab.vue` 加並排卡片.
* 改變允許的 interval: `internal/preferences/service.go` 的 `allowedAutoUpdateIntervalHours` + 前端 `AUTO_UPDATE_INTERVALS`. 兩邊必須同步.
* 改變資產檔名規則: 只動 `internal/updater/types.go` 的 `AssetNameFor`. Provider 不需改, 因為它們吃 `AssetName` 字串.
* 改變 in-progress 行為 (例: 想加進度 modal): 在 `useUpdater` 把 `installing` / `downloading` 狀態接到 `AppModal`, 不需動 backend.
* 改變 OS 支援: 加 `rename_<os>.go` build-tag 檔, 實作 `Swap(currentPath, newPath) error`. 其他層不變.

---

## 23. Installer 安裝版自動更新

從 v0.7+ 開始, NSIS installer 安裝版也支援 in-app 自動更新, 走的不是 portable 的 rename trick, 而是「下載新 installer + UAC elevated silent install」.

### 23.1 BuildKind 識別

由 `internal/updater/buildkind.go` 的 package 變數 `BuildKind` 識別當前二進位是哪種版本:

* `"portable"` (預設, 不傳 ldflag) -> 走 rename swap.
* `"installer"` (建置時 `-ldflags "-X 'redshell/internal/updater.BuildKind=installer'"`) -> 走 elevated silent install.
* 其他值 -> 安全 fallback 為 `"portable"`.

`IsInstaller()` / `IsPortable()` helper 取代字串比對, 所有需要分流的地方 (Service.Start, install dispatch, GetState, tray gating) 都用 helper.

### 23.2 建置 installer 版本 (預設, installer-only)

從 v0.7.x 開始, Windows 只發 installer 版本, 不再附 portable binary. `scripts/publish-wails.ps1` 預設 `-Nsis:$true` 一次出:

* `RedShell-amd64-installer.exe` — NSIS 安裝檔 (內含 BuildKind=installer 的 binary).
* `checksums.txt` — SHA-256 sidecar.

```pwsh
pwsh ./scripts/publish-wails.ps1
```

build 流程:

1. 一次 `wails build -nsis -ldflags "-s -w -X redshell/internal/updater.BuildKind=installer"`.
2. dist staging 只複製 NSIS installer + 寫 checksums.txt. **不**輸出 `redshell.exe` (那顆已經被包進 installer, 而且 BuildKind 是 installer; 當作 portable 散布會誤導使用者, 而且他們的 in-app updater 還是會走 elevated install pathway).

**重要陷阱**: `-X` flag 不可加引號. `"-X 'redshell/internal/updater.BuildKind=installer'"` 在 PowerShell 會把單引號當字面字元傳給 linker, importpath parse 失敗, **靜默** fallback 為 `"portable"`. 必須寫成 `"-X redshell/internal/updater.BuildKind=installer"` (無內層引號).

驗證方法 (publish 完之後):

```pwsh
Select-String -Path build/bin/RedShell-amd64-installer.exe -Pattern 'installer' -SimpleMatch
```

如果 binary 內找不到字串 `installer` 對應 BuildKind 變數, 表示 ldflag 沒生效, installer 安裝後 in-app updater 會誤判為 portable, 在 `Program Files` 跑時會顯示「This is a portable build placed in a directory that is not writable」警告.

#### 例外: 開發或測試需要 portable binary

如果暫時要產 portable (本地測 dev 流程, 或為非 Windows 平台 build), 加 `-Nsis:$false`:

```pwsh
pwsh ./scripts/publish-wails.ps1 -Nsis:$false
```

這條路 BuildKind 留在 `"portable"`, 走 rename-trick swap. 不應該用於正式 release.

### 23.3 Provider 對 portable asset 的處理 (installer-only release)

Provider (`provider_github.go` / `provider_gitlab.go` 的 `toRelease`) 把 portable + installer 兩個 asset 都當 **optional**:

* `checksums.txt` 必有 (兩條 pathway 都要 SHA verify).
* `redshell-windows-amd64.exe` (portable asset) — optional. 缺了不會錯, 只是 `Release.AssetURL` 留空.
* `RedShell-amd64-installer.exe` (installer asset) — optional. 缺了不會錯, 只是 `Release.InstallerAssetURL` 留空.

Install dispatch (`service.install`) 在 `BuildKind` 對應的 pathway 開頭檢查需要的 asset 是否存在; 不存在就 emit `updater:error` (`stage="download"` for portable, `stage="installer-download"` for installer) 並中止. 這個設計讓:

* Installer-only release 對 installer client 完全可用.
* 還在用舊 portable build 的使用者觸發 update 時看到 clear error: 「portable asset not in release vX.Y.Z; this release ships installer-only — download the latest installer manually to migrate」, 知道要手動下載 installer 一次.

### 23.4 安裝流程

按下「Update Now」之後:

1. 從 active source 取得當前 `Release`. Release 結構額外帶 `InstallerAssetURL` / `InstallerAssetName` / `InstallerAssetSize` 三個欄位; 由 provider 在解析 release JSON 時順便填入. 缺少 -> emit `updater:error` (stage `installer-download`) 並中止.
2. 並行下載 installer 二進位至 `%TEMP%\redshell-installer.exe` 與 `checksums.txt`. 兩個重要規則:
   - **位置**: 必須寫到使用者可寫位置 (`os.TempDir()`), 不能寫到 `<exe-dir>` — installer 版本的 exe 位於 `Program Files`, running app 沒有 admin 權限, 寫到那邊會 `Access is denied`. UAC 提權發生在下載完成 *之後*; 提權後的 child 仍可讀 `%TEMP%` (同 user, 只是換 token).
   - **副檔名**: 必須是 `.exe`. `ShellExecute` 是 verb-driven + extension-driven, 它會去 registry 找「`.<ext>` + `<verb>` 的 handler」; `.exe` 一定有, 但其他副檔名 (例如 `.new`) 會回 `SE_ERR_NOASSOC` ("No application is associated with the specified file for this operation"), 即使檔案本身是合法 PE binary 也一樣. Portable 走 `os.Rename + exec.Command` -> `CreateProcess` 看 PE header 不看副檔名, 所以沒這個問題.
3. 對 `RedShell-amd64-installer.exe` 在 checksums 內查 SHA-256, 與下載的串流 hash 比對. 不符 -> 刪檔, emit `updater:error` (stage `verify`).
4. 設 `inProgress = true` (放在 spawn 之前, 因為 UAC 對話框會 block 同 goroutine, close intercept 必須在那段時間 short-circuit).
5. 呼叫 `installerSpawn(installerPath, []string{"/S"})`. 在 Windows 上是 `golang.org/x/sys/windows.ShellExecute(0, "runas", path, "/S", nil, SW_SHOW)`, 觸發 UAC 對話框.
   * 使用者按「是」 -> 系統建立 elevated process, ShellExecute 回 nil. 我們 emit `updater:installed` 後 `quitApp()` 釋放檔案 lock.
   * 使用者按「否」 -> 回 `syscall.Errno(1223)` (`ERROR_CANCELLED`), 包裝為 `ErrUACDeclined`. 清 `inProgress`, emit `updater:error` (stage `installer-spawn`, message `"user cancelled elevation"`), 不 quit.
6. NSIS installer 以 `/S` 旗標靜默執行, 在 install Section 開頭有 `Sleep 2000` (見 `build/windows/installer/project.nsi`), 給剛 quit 的 RedShell 時間釋放檔案 lock 後才覆寫 `RedShell.exe`.
7. installer 完成後**不會自動重啟 RedShell** (見 §23.5), 使用者從開始功能表 / 桌面捷徑重開即可.

### 23.5 與 portable 流程的差異對照

| 階段 | Portable | Installer |
|---|---|---|
| 寫入路徑可寫性 | 需要寫得進 `filepath.Dir(exePath)`, 否則 `manual-required` | 不需要; 由 UAC 帶 admin 權限 |
| 下載資產 | `redshell-windows-amd64.exe` | `RedShell-amd64-installer.exe` |
| 寫入位置 | `<exe>.new.partial` -> `<exe>.new` | `<exe-dir>\redshell-installer.new.partial` -> `<exe-dir>\redshell-installer.new` |
| 換檔機制 | rename trick (允許在執行中 rename `.exe`) | NSIS installer 覆寫 (需 admin) |
| 重啟機制 | `defaultSpawn(exePath)` 直接 `exec.Command().Start()` | 使用者手動從開始功能表重開 |
| 進入 in-progress 時機 | swap 之前 | spawn 之前 (UAC 對話框 block 期間) |
| 失敗 stage | `download` / `verify` / `rename` / `spawn` | `installer-download` / `verify` / `installer-spawn` |

### 23.6 為何不在 silent install 結尾自動重啟

`build/windows/installer/wails_tools.nsh` 已經宣告 `RequestExecutionLevel "admin"`, 因此 installer 永遠 elevated 執行. 如果在 install Section 結尾加 `Exec '"$INSTDIR\RedShell.exe"'`, NSIS 用 `CreateProcess` 帶**parent 的 elevated token**, 重新啟動的 RedShell 會繼承 admin 權限. 這會: WebView2 sandbox 行為改變、檔案寫入落到 admin-only 位置、使用者沒有任何視覺提示自己跑在 admin 模式. 所以 v1 直接放棄自動重啟; 使用者按一下開始功能表是可接受成本.

未來若想做自動重啟, 可考慮:
* 在 RedShell quit 之前先 spawn 一個非 elevated launcher (cmd.exe 或 tiny Go helper), 由它 poll 兩個 PID 結束後再啟動 RedShell.
* 用 NSIS `UAC` 或 `ShellExecAsUser` plugin 在 installer 內 de-elevate 後啟動.

### 23.7 Tray gating 變動

`main.go`:

```go
if updaterApp != nil && updaterApp.AutoUpdateAvailable() {
    trayMgr.SetCheckForUpdates(updaterApp.HandleTrayOpen)
}
```

`AutoUpdateAvailable() = IsInstaller() || !ManualRequired()`. 因此:

* Portable 在可寫目錄 -> 註冊 tray 項目.
* Portable 在不可寫目錄 -> 不註冊 (manual-required).
* Installer (永遠) -> 註冊.

### 23.8 `ManualRequired` 語意收緊

舊版: `ManualRequired = !IsWritable(exeDir)`.
新版: `ManualRequired = IsPortable() && !IsWritable(exeDir)`.

Installer 版本永遠回 `false`, 即使在 `Program Files`. 前端 `UpdatesTab.vue` 的警告 alert 也跟著改 `manualRequired && buildKind === 'portable'`, 並對 installer 版本顯示一行 info alert 提醒會有 UAC 提示.

### 23.9 Release-step 檢查

每次 release 出 installer 版本時, 建議的人工驗證步驟:

```pwsh
strings build\bin\RedShell-amd64-installer.exe | Select-String "BuildKind"
```

確認字串 `installer` 有被 linker 替換進二進位; 沒有則代表 ldflag 漏帶, installer 版本會 fallback 為 portable, 在 `Program Files` 跑時會誤判為 manual-required.

### 23.10 從舊版 installer / portable 升級到 installer-only

* 舊版 installer (尚未支援 in-app 更新) 上的使用者: 無法直接從 in-app 觸發升到第一個支援自動更新的版本. 需要手動下載一次新 installer 安裝. 從那個版本開始, 後續更新都走 in-app.
* 舊版 portable (BuildKind=portable) 使用者在 installer-only 切換之後: in-app updater 會 emit `[download] portable asset not in release vX.Y.Z` 並中止. UI 上會看到清楚的錯誤訊息提示「download the latest installer manually to migrate」. 使用者下載 installer 安裝後, BuildKind 變 installer, 之後就走 in-app 更新.
