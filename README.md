# RedShell

Wails v2 桌面工具,用於管理 AI coding agents (Claude Code, GitHub Copilot) 的 plugin marketplace.

## 發佈流程 (Cutting a release)

RedShell 的 portable 版本支援自動更新 (僅 Windows). 發新版時務必同時上傳 binary 與 `checksums.txt` 到 GitHub 與 GitLab 的 release page,讓兩邊的 active source 都能解析到一致的 asset.

### 1. 更新版本號

編輯 `wails.json` 的 `info.productVersion`:

```jsonc
{
  "info": {
    "productVersion": "0.5.0"
  }
}
```

### 2. Build

```sh
wails build --target windows/amd64 --nsis
```

產出兩個檔案在 `build/bin/`:
- `redshell.exe` (portable, 將被自動更新使用)
- `RedShell-amd64-installer.exe` (NSIS 安裝程式)

### 3. 重新命名 portable binary 並產生 checksums

auto-update 期望的 asset 名稱固定為 `redshell-<goos>-<goarch>(.exe)`. 例如 Windows amd64 是 `redshell-windows-amd64.exe`:

```sh
cd build/bin
mv redshell.exe redshell-windows-amd64.exe
sha256sum redshell-windows-amd64.exe RedShell-amd64-installer.exe > checksums.txt
```

`checksums.txt` 內容範例:

```
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  redshell-windows-amd64.exe
da39a3ee5e6b4b0d3255bfef95601890afd80709da39a3ee5e6b4b0d3255bfef  RedShell-amd64-installer.exe
```

### 4. 上傳到 GitHub 與 GitLab releases

以同一個 git tag (例如 `v0.5.0`) 在 **兩邊** 都建立 release,各自附上以下三個檔案:

- `redshell-windows-amd64.exe`
- `RedShell-amd64-installer.exe`
- `checksums.txt`

範例:

```sh
git tag v0.5.0 && git push origin v0.5.0
gh release create v0.5.0 build/bin/redshell-windows-amd64.exe build/bin/RedShell-amd64-installer.exe build/bin/checksums.txt --title "v0.5.0" --notes "..."
glab release create v0.5.0 build/bin/redshell-windows-amd64.exe build/bin/RedShell-amd64-installer.exe build/bin/checksums.txt --name "v0.5.0" --notes "..."
```

### 5. 驗證自動更新流程

從一個較舊版本 (例如 `0.4.0`) 啟動 portable 版,到 `Settings -> Updates`:

- 兩個 source 都應該顯示 `v0.5.0`.
- 點 `Check now`.
- 點 `Update to v0.5.0`.
- 流程: 下載 -> SHA256 驗證 -> rename swap -> spawn 新 process -> 舊 process 退出.
- 重啟後 `redshell.exe` 應為新版本; 舊 `redshell.exe.old` 應在啟動時被清掉.

### 注意事項

- **沒有 code signing**: 第一次更新後 SmartScreen 可能跳警告. 完整性靠 `checksums.txt` 保護, 不依賴簽章.
- **兩邊版本可能不一致**: 平行 source 各自獨立, UI 會在 Settings 並排顯示兩邊最新版讓使用者比較.
- **assertions 命名**: 如果未來支援 macOS / Linux, asset 名稱規則固定為 `redshell-<goos>-<goarch>` (例如 `redshell-darwin-arm64`, `redshell-linux-amd64`).
