# 專案總覽

## 核心用途

`blackbox_agent` 會依照 YAML 設定，定期對指定 target 執行黑箱探測，例如：

- HTTP
- TCP
- ICMP
- DNS
- gRPC
- NTP

每個 target 的檢查結果會被整理成：

- Prometheus metrics，透過 remote write 推送
- OpenSearch 文件，供查詢與後續分析

## 核心設計

- 以 CLI 程式啟動
- 以 YAML 驅動 probe 行為
- 對同一份 `blackbox.yaml` 同時承接 upstream blackbox module 與本專案自定義 `ntp` 設定
- 以 goroutine 平行執行 target 檢查
- 以 RabbitMQ RPC 接收設定熱更新
- 以 context 管理關閉與重載

## 主要資料流

1. CLI 讀取指定檔案名稱
2. AutoLoader 解析 config 並初始化依賴
3. BlackboxProcess 載入 target/module 設定
4. 每個 job 依 `scrape_interval` 啟動 ticker
5. 每個 target 以對應 prober 執行 probe
6. probe metrics 送往 Prometheus
7. probe 結果視設定寫入 OpenSearch

## 目錄重點

- `cmd/`: CLI 入口
- `config/`: config 解析與結構定義
- `pkg/autoload/`: 啟動與重載控制
- `handler/`: 執行流程、MQ、YAML 檢查
- `exporter/`: probe 執行與 metrics 收集
- `internal/blackboxadapter/`: blackbox backend adapter、config 分流、custom NTP probe
- `model/`: OpenSearch 與 Prometheus remote write
- `blackbox_exporter/`: 本地保留的 blackbox YAML 樣板與尚待清理的 fork 資產
- `yaml/`: 本地設定檔樣板與實際目標檔
