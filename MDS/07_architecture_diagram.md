# 架構圖與資料流

本文件是偏維運與交接視角的架構圖版本，重點是快速理解元件關係、資料流與 reload 路徑。

## 整體架構圖

```mermaid
flowchart LR
    CLI[CLI<br/>main.go / cmd/root.go] --> AUTO[AutoLoader<br/>pkg/autoload/loader.go]
    AUTO --> CFG[config.ConfigInit]
    AUTO --> SRV[server.Server]
    AUTO --> MQ[ListenRabbitMQ]
    AUTO --> BB[BlackboxProcess]

    CFG --> YAML1[yaml/config.yaml]
    BB --> YAML2[yaml/target.yaml]
    BB --> YAML3[blackbox_exporter/blackbox.yaml]

    BB --> ADPLOAD[blackboxadapter loader]
    ADPLOAD --> NTPCFG[adapter NTP config]
    ADPLOAD --> UPCFG[upstream sanitized config]

    BB --> TIME[TimeControl]
    TIME --> JOB[dataResolve]
    JOB --> EXP[exporter.CheckModuleAndDoProbe]
    EXP --> ADPREG[blackboxadapter registry]
    ADPREG --> UPPROBER[official blackbox_exporter probers]
    ADPREG --> NTPPROBER[custom NTP runner]

    UPPROBER --> METRIC[metric.ProcessMetrics]
    NTPPROBER --> METRIC
    METRIC --> PROM[Prometheus Remote Write]

    JOB --> OS[model.BulkInsert]
    OS --> OPEN[OpenSearch]

    MQ --> VALID[YAML Validation]
    VALID --> SAVE[覆蓋本地 YAML]
    SAVE --> RELOAD[reload chan]
    RELOAD --> BB
```

## 啟動流程圖

```mermaid
sequenceDiagram
    participant U as User
    participant C as CLI
    participant A as AutoLoader
    participant S as Server
    participant M as MQ Listener
    participant B as BlackboxProcess

    U->>C: 執行 ./blackbox_agent
    C->>A: AutoLoader(config, target, blackbox)
    A->>A: ConfigInit()
    A->>S: NewServer()
    A->>S: 設定 Const / Logger / Prometheus
    A->>S: 設定 OpenSearch / RabbitMQ
    A->>M: goroutine 啟動 ListenRabbitMQ
    A->>B: goroutine 啟動 BlackboxProcess
```

## Probe 執行資料流

```mermaid
flowchart TD
    T[target.yaml 的 scrape_configs] --> I[解析 job_name / interval / module / targets]
    I --> K[每個 job 建 ticker]
    K --> R[dataResolve]
    R --> G[每個 target 啟 goroutine]
    G --> S[依 module 選 adapter runner]
    S --> P1[執行 HTTP/TCP/ICMP/DNS/GRPC upstream probe]
    S --> P2[執行 custom NTP probe]
    P1 --> C[Gather metrics]
    P2 --> C
    C --> W[轉成 prompb.TimeSeries]
    W --> PRW[送往 Prometheus Remote Write]
    C --> D[整理 probe 文件]
    D --> O{OpenSearch enable?}
    O -- yes --> BI[BulkInsert]
    O -- no --> END[只保留 Prometheus 輸出]
```

## RabbitMQ Reload 流程圖

```mermaid
flowchart TD
    Q[RabbitMQ RPC 訊息] --> T{queue 類型}
    T -- modules --> BM[驗證 blackbox YAML]
    T -- targets --> TG[驗證 target YAML]
    BM --> SB[寫入 blackbox.yaml 或 blackbox_error.yaml]
    TG --> ST[寫入 target.yaml 或 target_error.yaml]
    SB --> RC[reload chan <- true]
    ST --> RC
    RC --> CX[取消舊 blackbox context]
    CX --> NB[重新啟動 BlackboxProcess]
```

## 元件說明

### CLI 層

- `main.go`
- `cmd/root.go`

用途：

- 接收參數
- 指定 config / target / blackbox 檔名
- 交給 AutoLoader 啟動

### 啟動與共享狀態層

- `pkg/autoload/loader.go`
- `server/server.go`
- `pkg/tool/log.go`
- `pkg/tool/gs.go`

用途：

- 初始化全域依賴
- 保存共享連線與常數
- 管理 graceful shutdown
- 管理 reload

### Probe 執行層

- `handler/blackbox.go`
- `exporter/config.go`
- `exporter/handler.go`
- `internal/blackboxadapter/*`

用途：

- 排程 job
- 平行執行 target probe
- 收集 probe 結果與 metrics
- 分流官方 blackbox config 與自定義 `ntp` config

### YAML 資產層

- `blackbox_exporter/blackbox.yaml`
- `blackbox_exporter/blackbox_example.yaml`
- `blackbox_exporter/blackbox_error.yaml`

用途：

- 保留本地 module 設定檔與樣板
- 提供 reload 與錯誤落盤使用
- 不再承載 blackbox runtime source code

### 設定更新層

- `handler/mq.go`
- `handler/yaml_check/module/*`
- `handler/yaml_check/target/*`

用途：

- 接收新 YAML
- 驗證格式與內容
- 寫回本地檔案
- 觸發重載

### 輸出層

- `model/metric/metric.go`
- `model/prometheusremotewrite/remotewriteclient.go`
- `model/opensearch.go`

用途：

- 轉換 metrics
- 發送 remote write
- 寫入 OpenSearch

## 目前架構上的注意點

- reload 後固定使用 `target.yaml` 與 `blackbox.yaml` 重啟，未保留原本 CLI 自訂檔名
- `http_server/` 目前未接入主啟動流程
- metrics 處理時會產生 `output.txt`
- `server.Server` 是全域共享狀態，修改時要注意併發與 reload 影響
- `blackbox.yaml` 雖然是單一檔案，但 `ntp` 區塊不會直接交給官方 blackbox_exporter config parser，而是先由 adapter 抽出並分流
