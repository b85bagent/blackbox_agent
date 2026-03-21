# Blackbox Import 重構計畫

## 目標

將目前 repo 內嵌的 `blackbox_exporter/` fork 架構，重構為：

- 以 Go module 方式依賴 upstream 或自家 fork
- 專案業務碼不再直接耦合 upstream `config` / `prober` 細節
- 保留目前既有功能：
  - MDS / 本地 YAML 載入
  - `target.yaml` / `blackbox.yaml` 流程
  - Prometheus remote write
  - OpenSearch 輸出
  - RabbitMQ reload
  - YAML check
  - 自有 `ntp` prober
- 讓未來 blackbox_exporter 升版時，影響集中在 adapter 層

## 現況判斷

目前專案不是單純「引用 blackbox_exporter」，而是直接把它當內部框架使用。

主要直接耦合點：

- [handler/blackbox.go](/home/systexadmin/blackbox_agent/handler/blackbox.go)
  - 直接依賴 `blackbox_exporter/config.SafeConfig`
  - 直接呼叫 `ReloadConfig()`
- [exporter/config.go](/home/systexadmin/blackbox_agent/exporter/config.go)
  - 直接維護 `ProbeFn` map
- [exporter/handler.go](/home/systexadmin/blackbox_agent/exporter/handler.go)
  - 直接使用 upstream/fork `config.Module`
  - 直接使用 upstream/fork `prober.ProbeFn`
- [handler/mq.go](/home/systexadmin/blackbox_agent/handler/mq.go)
  - reload 驗證直接吃 `ReloadConfig()`
- [blackbox_exporter/prober/ntp.go](/home/systexadmin/blackbox_agent/blackbox_exporter/prober/ntp.go)
  - 是本地自有擴充，不在 upstream

這代表如果只做 import path 替換，未來升版痛點仍然存在。

## 重構原則

1. 不直接讓 `handler/`、`exporter/` 依賴 upstream package。
2. 先抽 adapter，再替換 backend。
3. `ntp` 必須從 upstream/fork 主體中拆出，改成自有註冊 prober。
4. 重構過程中每一階段都要可編譯、可跑 `docker compose -f docker-compose.dev.yml run --rm test`。
5. 避免在同一個 PR 同時做架構切換與大規模升版。

## 目標架構

建議新增一層 `internal/blackboxadapter`，由它向上提供穩定介面，向下連接 blackbox backend。

建議結構：

- `internal/blackboxadapter/`
  - `types.go`
  - `interfaces.go`
  - `loader.go`
  - `registry.go`
  - `upstream_backend.go`
  - `custom_ntp.go`

上層業務碼只依賴 adapter，不直接依賴：

- `prometheus/blackbox_exporter/config`
- `prometheus/blackbox_exporter/prober`

## 建議抽出的 interface

### 1. ConfigLoader

用途：包住 blackbox module config 載入與 reload。

建議介面：

```go
type ConfigLoader interface {
    Reload(path string) error
    Module(name string) (ModuleDef, bool)
    Modules() map[string]ModuleDef
}
```

說明：

- `handler/blackbox.go`
- `handler/mq.go`

應只依賴這個介面，不再碰 `SafeConfig`。

### 2. ModuleDef

用途：隔離 upstream `config.Module`。

建議：

```go
type ModuleDef struct {
    Name   string
    Prober string
    Raw    any
}
```

說明：

- 先用 `Raw any` 過渡，避免第一階段就做完整 schema 重建
- 後續若穩定，再逐步改成明確 adapter types

### 3. ProbeRunner

用途：統一 probe 執行入口。

建議介面：

```go
type ProbeRunner interface {
    Run(ctx context.Context, module ModuleDef, target string, registry *prometheus.Registry, logger log.Logger) bool
}
```

### 4. ProberRegistry

用途：集中管理 upstream probers + custom probers。

建議介面：

```go
type ProberRegistry interface {
    Register(name string, runner ProbeRunner)
    Get(name string) (ProbeRunner, bool)
}
```

說明：

- `http/tcp/icmp/dns/grpc` 由 upstream backend 註冊
- `ntp` 由本地註冊

### 5. BackendAdapter

用途：包住特定 blackbox backend 實作。

建議介面：

```go
type BackendAdapter interface {
    NewLoader() ConfigLoader
    DefaultRegistry() ProberRegistry
}
```

## 哪些檔案先改

### 第一批：先建立隔離層

新增：

- `internal/blackboxadapter/interfaces.go`
- `internal/blackboxadapter/types.go`
- `internal/blackboxadapter/registry.go`
- `internal/blackboxadapter/upstream_backend.go`

先不刪任何現有 fork code。

### 第二批：把業務碼改依賴 adapter

優先修改：

- [handler/blackbox.go](/home/systexadmin/blackbox_agent/handler/blackbox.go)
- [handler/mq.go](/home/systexadmin/blackbox_agent/handler/mq.go)
- [exporter/config.go](/home/systexadmin/blackbox_agent/exporter/config.go)
- [exporter/handler.go](/home/systexadmin/blackbox_agent/exporter/handler.go)

目標：

- 移除對 `blackbox_exporter/config` 的直接 import
- 移除對 `blackbox_exporter/prober` 的直接 import

### 第三批：拆 NTP

新增：

- `internal/blackboxadapter/custom_ntp.go`

來源：

- [blackbox_exporter/prober/ntp.go](/home/systexadmin/blackbox_agent/blackbox_exporter/prober/ntp.go)

目標：

- 不再把 `ntp` 放在 fork 目錄裡
- 由 adapter registry 額外註冊 `ntp`

### 第四批：替換 backend 來源

這階段才把 backend 從 repo 內嵌 fork，換成：

- upstream module
- 或你自己的 blackbox fork module

補充：

- 由於官方 `github.com/prometheus/blackbox_exporter` 不支援本專案自定義的 `ntp:` schema
- PR4 需要保留單一 `blackbox.yaml`，但在 adapter loader 中做雙路解析：
  - 一路抽出 `ntp` 設定，供 custom runner 使用
  - 一路移除 `ntp:` 欄位後交給官方 upstream config 載入

### 第五批：清理 fork 目錄

最後再移除 repo 內不需要的 `blackbox_exporter/` 原始碼。

## 哪些 fork code 可以保留

建議保留：

- `blackbox_exporter/blackbox.yaml`
  - 這是你的本地 module config 樣板，不屬於 upstream runtime library 問題
- `blackbox_exporter/blackbox_example.yaml`
  - 作為樣板可保留
- `config testdata`
  - 若仍有用可暫留，直到 adapter 測試重建完成

## 哪些 fork code 要拆

必拆：

- [blackbox_exporter/prober/ntp.go](/home/systexadmin/blackbox_agent/blackbox_exporter/prober/ntp.go)
  - 這是自有功能，不應混在 upstream 本體

應逐步替換掉：

- [blackbox_exporter/config/config.go](/home/systexadmin/blackbox_agent/blackbox_exporter/config/config.go)
- [blackbox_exporter/prober/prober.go](/home/systexadmin/blackbox_agent/blackbox_exporter/prober/prober.go)
- [blackbox_exporter/prober/handler.go](/home/systexadmin/blackbox_agent/blackbox_exporter/prober/handler.go)
- 其他 `http.go` / `tcp.go` / `dns.go` / `icmp.go` / `grpc.go`

說明：

- 不是一開始就刪
- 而是先讓 adapter 轉接，最後再抽掉

## 哪些 fork code 可以暫時不動

可延後處理：

- `blackbox_exporter/*_test.go`
  - 若第一階段仍暫用 fork backend，可先保留
- `blackbox_exporter/config/testdata/*`
  - 可等到 adapter 測試補齊後再清理

## 建議分幾個 PR

### PR1：建立 adapter 骨架，不改行為

內容：

- 新增 `internal/blackboxadapter`
- 定義 interface / types / registry
- 寫一個包裝目前 fork backend 的 adapter

不做：

- 不換 module import
- 不改 `handler/` / `exporter/` 行為

驗證：

- `docker compose -f docker-compose.dev.yml run --rm test`

風險：

- 低

### PR2：業務碼切到 adapter

內容：

- 修改 [handler/blackbox.go](/home/systexadmin/blackbox_agent/handler/blackbox.go)
- 修改 [handler/mq.go](/home/systexadmin/blackbox_agent/handler/mq.go)
- 修改 [exporter/config.go](/home/systexadmin/blackbox_agent/exporter/config.go)
- 修改 [exporter/handler.go](/home/systexadmin/blackbox_agent/exporter/handler.go)

目標：

- 業務碼不再 import `blackbox_exporter/config`、`blackbox_exporter/prober`

驗證：

- `docker compose -f docker-compose.dev.yml run --rm test`
- `docker compose -f docker-compose.dev.yml run --rm verify`

風險：

- 中

### PR3：拆出 NTP custom prober

內容：

- 將 [blackbox_exporter/prober/ntp.go](/home/systexadmin/blackbox_agent/blackbox_exporter/prober/ntp.go) 移到 adapter/custom
- 由 registry 額外註冊 `ntp`

目標：

- upstream backend 不含自有 patch

驗證：

- `docker compose -f docker-compose.dev.yml run --rm test`
- `docker compose -f docker-compose.dev.yml run --rm verify`
- 加一個最小 NTP adapter test

風險：

- 中

### PR4：backend 改成 module import

內容：

- `go.mod` 引入 upstream 或自家 fork module
- adapter backend 切到 module import
- adapter loader 對單一 `blackbox.yaml` 做 upstream/custom 分流

建議策略：

- 優先先引自家 fork module
- 等 adapter 穩定後再追 upstream tag

若直接引官方 upstream，前提是：

- `ntp` schema 已完全留在 adapter
- adapter 會先產生 upstream 可接受的 sanitized config

驗證：

- `docker compose -f docker-compose.dev.yml run --rm test`
- `docker compose -f docker-compose.dev.yml run --rm build`
- `docker compose -f docker-compose.dev.yml run --rm verify`
- `docker build -f Dockerfile .`
- `docker build -f Dockerfile_blackbox .`

風險：

- 高

### PR5：清理內嵌 fork 原始碼

內容：

- 移除不再使用的 `blackbox_exporter/config/*.go`
- 移除不再使用的 `blackbox_exporter/prober/*.go`
- 保留樣板 YAML 與必要測試資產

狀態：

- 已完成移除 embedded `config/` 與 `prober/` 原始碼
- `blackbox_exporter/` 目前只保留 YAML 樣板與 runtime 需要的檔案

驗證：

- 全量驗證同 PR4

風險：

- 中

## 最穩的 backend 選擇

### 選項 A：直接依賴 upstream module

優點：

- 最接近社群版本
- 後續安全更新最直接

缺點：

- upstream 並非穩定 library API
- 任何 logger/config/prober 變更都可能打到 adapter
- upstream 不接受本專案自定義 `ntp:` schema，必須由 adapter 先分流

適用：

- adapter 已經抽乾淨

### 選項 B：先依賴自家 fork module

優點：

- 可以保留少量必要 patch
- 升版節奏自己控制

缺點：

- 仍需維護 fork

適用：

- 目前最務實

## 建議決策

建議先走：

1. 抽 adapter
2. 拆 NTP
3. 改依賴自家 fork module
4. 等整體穩定後再追 upstream 最新版

不建議：

- 直接刪 fork 目錄後全面 import upstream

原因：

- 風險太大
- NTP / logger / config schema / test 調整會一次爆在同一個 PR

## 驗證基線

每一階段至少維持以下命令可用：

```bash
docker compose -f docker-compose.dev.yml run --rm test
docker compose -f docker-compose.dev.yml run --rm build
docker compose -f docker-compose.dev.yml run --rm verify
docker build -f Dockerfile -t blackbox-agent:test .
docker build -f Dockerfile_blackbox -t blackbox-agent:blackbox .
```

## 完成條件

當以下條件成立，代表重構完成：

- `handler/` 與 `exporter/` 不再 import `blackbox_exporter/config` / `prober`
- `ntp` 已完全脫離 fork 本體
- `blackbox_exporter` backend 來自 module import，而不是 repo 內嵌原始碼
- 全部 Docker / Compose 驗證可過
- 更換 blackbox backend 版本時，只需修改 adapter 層與少量測試
