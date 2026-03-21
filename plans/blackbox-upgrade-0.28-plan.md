# Blackbox Exporter 0.28 升級計畫

## 目標

將專案目前依賴的 `github.com/prometheus/blackbox_exporter v0.27.0` 升級到 `v0.28.0`，並評估是否同步開放 upstream 新功能與新協定，控制升級風險，避免把依賴升版、schema 變更、功能擴張綁在同一批改動中。

本計畫同時回答兩個問題：

- 是否應升級到 `0.28.0`
- 是否應新增 `0.28.0` 帶來的監控能力與新協定

## 結論摘要

建議升級到 `0.28.0`，但建議分階段。

原因：

- 目前 repo 實際版本仍是 `v0.27.0`
- `0.28.0` 有明確可用的新能力：
  - HTTP `enable_http3`
  - gRPC `metadata`
  - TCP `query_response.expect_bytes`
  - 新增 `unix` prober
- 本專案不是純 upstream binary 包裝，而是有自家 adapter、registry、YAML validator、sample YAML、reload 流程
- 因此升級 `go.mod` 不等於使用者可立即使用新功能

## 現況

### 依賴版本

- [go.mod](/home/systexadmin/blackbox_agent/go.mod#L16)
  - 目前依賴 `github.com/prometheus/blackbox_exporter v0.27.0`

### 現有支援的 prober 註冊

- [internal/blackboxadapter/upstream_backend.go](/home/systexadmin/blackbox_agent/internal/blackboxadapter/upstream_backend.go#L28)
  - 目前註冊：
    - `http`
    - `tcp`
    - `icmp`
    - `dns`
    - `grpc`
    - `ntp`
  - 尚未註冊 `unix`

### 現有 YAML validator 限制

- [handler/yaml_check/module/check.go](/home/systexadmin/blackbox_agent/handler/yaml_check/module/check.go#L37)
  - `allowedProbers` 目前只有：
    - `http`
    - `tcp`
    - `grpc`
    - `icmp`
    - `dns`
    - `ntp`
  - 尚未接受 `unix`

### 新功能目前會被 validator 擋住的點

- [handler/yaml_check/module/grpc.go](/home/systexadmin/blackbox_agent/handler/yaml_check/module/grpc.go#L16)
  - 尚未驗證 `metadata`
- [handler/yaml_check/module/tcp.go](/home/systexadmin/blackbox_agent/handler/yaml_check/module/tcp.go#L37)
  - 尚未驗證 `expect_bytes`
- [handler/yaml_check/module/http.go](/home/systexadmin/blackbox_agent/handler/yaml_check/module/http.go#L23)
  - 尚未涵蓋 `enable_http3`
  - 目前 HTTP validator 本身也偏脆弱，升級時應一起補齊

## Upstream 0.28 變更對本專案的影響

### 建議納入評估的變更

- `http.enable_http3`
  - 對外部網站、API gateway、CDN、邊界服務探測有價值
- `grpc.metadata`
  - 對需要攜帶 header / token / tenant context 的 gRPC 健康檢查有價值
- `tcp.query_response.expect_bytes`
  - 對二進位協定、非純文字 banner 驗證有價值
- `unix` prober
  - 可用於本機 socket、sidecar、node-local 服務探測

### 可先不追的變更

- upstream binary 的 auto-reload
  - 本專案已有自己的 reload 流程，不是直接跑 upstream binary
- upstream logging 行為調整
  - 本專案目前走 adapter 封裝，影響相對有限，但仍要做 smoke 驗證

## 是否要分階段

建議分 3 個階段，原因如下：

- 降低單一 PR 風險
- 清楚分離「依賴升版」與「功能擴張」
- 方便在每階段做 docker compose 驗證
- 若新功能需求不明確，可先完成低風險升版

## 分階段計畫

### Phase 1: 依賴升版與相容性驗證

目標：

- 將 `blackbox_exporter` 從 `v0.27.0` 升到 `v0.28.0`
- 不新增新 prober
- 不對外宣告新 YAML 功能
- 先確認現有 `http/tcp/icmp/dns/grpc/ntp` 行為不回歸

範圍：

- 更新 `go.mod` / `go.sum`
- 編譯修正
- 跑既有測試
- 跑 container build / verify
- 檢查 sample YAML 是否仍可載入

完成標準：

- `docker compose -f docker-compose.dev.yml run --rm test` 通過
- `docker compose -f docker-compose.dev.yml run --rm build` 通過
- 若本次有動到 probe / YAML / startup，補跑 `docker compose -f docker-compose.dev.yml run --rm verify`

風險：

- upstream `0.28` logger 行為調整可能影響 probe log 噪音
- upstream config/schema 細節變更可能影響 sample YAML

建議：

- 這一階段可以獨立成一個 PR

### Phase 2: 開放既有 prober 的新能力

目標：

- 不新增新協定種類
- 僅讓既有 `http/tcp/grpc` 可以使用 `0.28` 的新欄位

建議納入：

- `http.enable_http3`
- `grpc.metadata`
- `tcp.query_response.expect_bytes`

需要修改的位置：

- `handler/yaml_check/module/http.go`
- `handler/yaml_check/module/grpc.go`
- `handler/yaml_check/module/tcp.go`
- `blackbox_exporter/blackbox.yaml`
- `blackbox_exporter/blackbox_example.yaml`
- 說明文件與 MDS

完成標準：

- validator 接受新欄位
- loader / adapter 不需額外特判即可正常交給 upstream module
- sample YAML 提供最小可用範例
- 新欄位至少補單元測試或 YAML 驗證測試

風險：

- 現有 validator 寫法偏手工白名單，補欄位時容易遺漏互斥條件
- HTTP validator 結構與 upstream schema 的對齊程度需要重新檢查

建議：

- 這一階段可與 Phase 1 分開
- 若時間有限，優先順序建議：
  1. `grpc.metadata`
  2. `tcp.expect_bytes`
  3. `http.enable_http3`

### Phase 3: 評估並導入新協定 `unix`

目標：

- 評估是否真的需要 `unix` prober
- 若需要，再導入 registry、validator、sample 與文件

需要修改的位置：

- `internal/blackboxadapter/upstream_backend.go`
  - 註冊 `unix`
- `handler/yaml_check/module/check.go`
  - 允許 `unix`
- 新增 `handler/yaml_check/module/unix.go`
  - 驗證 `unix` 模組欄位
- `blackbox_exporter/blackbox.yaml`
- `blackbox_exporter/blackbox_example.yaml`
- 文件與使用說明

風險：

- `unix` 比較偏本機檢查，不一定符合目前大多數 target 使用情境
- target 寫法、部署位置、container 權限與 volume mount 可能都要一起定義

採納建議：

- 若目前需求仍以外部 endpoint 為主，`unix` 不建議跟 Phase 1 綁在一起
- 若有明確需求，例如：
  - 同機 container health check
  - node-local agent / daemon socket
  - sidecar / service mesh unix socket
  才建議納入

## 功能採納建議

### 建議本輪直接做

- 升級到 `0.28.0`
- 補齊：
  - `grpc.metadata`
  - `tcp.query_response.expect_bytes`
  - `http.enable_http3`

理由：

- 都是既有 prober 的擴充
- 不需擴大 target model
- 使用價值高
- 風險低於新增 `unix`

### 建議延後

- `unix` prober

理由：

- 屬於新協定類型，不只是欄位增加
- 需要額外定義 sample、validator、部署情境
- 沒有明確需求時，導入優先度低

## 建議 PR 切法

### PR1: Upgrade Only

內容：

- `go.mod` / `go.sum` 升到 `0.28.0`
- 修正編譯與測試
- 不引入新 schema

優點：

- 風險最小
- 容易回歸比對

### PR2: Existing Prober Features

內容：

- `http.enable_http3`
- `grpc.metadata`
- `tcp.expect_bytes`
- 補 sample / docs / validator tests

優點：

- 對外功能增量明確
- 不會跟依賴升版混在一起

### PR3: Unix Prober

內容：

- registry 註冊 `unix`
- YAML validator
- sample / docs
- 依需求補 smoke case

優點：

- 僅在確定需要時才做

## 驗證策略

每一階段都應至少執行：

- `docker compose -f docker-compose.dev.yml run --rm test`
- `docker compose -f docker-compose.dev.yml run --rm build`

以下情況補跑：

- 若有修改 probe 邏輯、啟動流程、YAML schema、sample config、Dockerfile
- 執行 `docker compose -f docker-compose.dev.yml run --rm verify`

針對新功能，建議補的測試：

- `grpc.metadata`
  - validator 可接受 map[string][]string 類型結構
- `tcp.expect_bytes`
  - validator 可接受位元組字串欄位
  - 與 `expect` 的互斥規則要明確
- `http.enable_http3`
  - validator 接受 bool
  - 與 `enable_http2` / `valid_http_versions` 的限制至少在文件說清楚
- `unix`
  - validator
  - registry 註冊
  - 最小 smoke sample

## 開發注意事項

- 本專案的 YAML validator 目前偏向手工欄位檢查，升版時不能假設 upstream schema 會自動透傳
- `ntp` 是自家擴充，升版過程中不能讓 upstream config sanitize 流程影響既有行為
- 若 Phase 1 發現 upstream `0.28` 對既有 probe 行為有不相容，應先停在只完成相容性修正，不要同時開新功能

## 最終建議

建議採分階段執行：

1. 先做 `0.28.0` 依賴升版與回歸驗證
2. 再開放 `grpc.metadata`、`tcp.expect_bytes`、`http.enable_http3`
3. `unix` prober 依實際需求獨立評估，不建議直接綁進升版 PR

這樣可以把風險控制在可回退範圍內，也比較符合目前專案 adapter 化之後的維護方式。
