# mtr-runner

跨平台 CLI，從你電腦執行 traceroute 並把結果送到 [mtr-web](https://github.com/adalf0722/mtr-web) 結果頁顯示。資料透過 URL（gzip + base64url）傳遞，不經任何後端伺服器。

- 不需 sudo、不需安裝 mtr 或其他依賴
- 單一 binary，下載即用
- macOS / Linux 用內建 `traceroute`，Windows 用 `tracert`

## 使用方式

### 1. 下載對應平台的 binary

到 [Releases](https://github.com/adalf0722/mtr-runner/releases/latest) 下載：

| 平台 | 檔名 |
| --- | --- |
| macOS (Apple Silicon) | `mtr-runner-darwin-arm64` |
| macOS (Intel) | `mtr-runner-darwin-amd64` |
| Linux (64-bit) | `mtr-runner-linux-amd64` |
| Windows (64-bit) | `mtr-runner-windows-amd64.exe` |

### 2. 執行

**macOS：**

```bash
chmod +x ./mtr-runner-darwin-arm64
xattr -dr com.apple.quarantine ./mtr-runner-darwin-arm64
./mtr-runner-darwin-arm64 --target 8.8.8.8 --site https://your-mtr-web.example.com
```

`xattr` 那行是繞過 Gatekeeper 對網路下載 binary 的隔離標記。

**Linux：**

```bash
chmod +x ./mtr-runner-linux-amd64
./mtr-runner-linux-amd64 --target 8.8.8.8 --site https://your-mtr-web.example.com
```

**Windows（PowerShell 或 cmd）：**

```cmd
mtr-runner-windows-amd64.exe --target 8.8.8.8 --site https://your-mtr-web.example.com
```

執行約 15 秒（10 輪 traceroute、輪間隔 1.5 秒以避開 ICMP rate limit），完成後會自動開啟瀏覽器到結果頁。

### Flag

| Flag | 說明 |
| --- | --- |
| `--target` | 目標 IP 或域名（必填） |
| `--site` | 結果頁網址，如 `https://mtr.example.com`（必填） |

## 為什麼用 traceroute 而不是 raw socket？

純 Go 的 mtr library（go-mtr、tonobo/mtr）需要 raw socket，意味著要 sudo 才能跑——使用者下載個小工具還要 `sudo` 體驗很差。

相對地系統內建的 `traceroute` / `tracert` 不需要任何特殊權限，到處都有。代價是受 ICMP rate limit 影響較大（特別是 Google 8.8.8.8 這類 anycast 目標的最後一跳），透過多輪採樣 + 前端「假性掉包」識別邏輯處理。

## 開發

```bash
go test ./...        # 跑單元測試
go build .           # 建構當前平台 binary
go run . --target 1.1.1.1 --site http://localhost:5173
```

### 發布新版本

推一個 `v*` tag 即觸發 CI 跨平台建構並上傳到 GitHub Releases：

```bash
git tag v0.1.4
git push origin v0.1.4
```

`.github/workflows/release.yml` 矩陣建構 darwin-arm64 / darwin-amd64 / linux-amd64 / windows-amd64。
