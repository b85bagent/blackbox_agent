# Blackbox Agent 專案導覽

## 專案定位

本專案是一個以 Go 實作的黑箱探測代理程式，負責：

- 讀取 `yaml/config.yaml`、`yaml/target.yaml`、`blackbox_exporter/blackbox.yaml`
- 根據 target 與 module 定義執行 blackbox probe
- 將 probe 指標轉成 Prometheus remote write 格式並推送到 Prometheus
- 視設定將 probe 結果批次寫入 OpenSearch
- 監聽 RabbitMQ RPC 訊息，接收新的 target/module YAML 後覆蓋本地檔案並觸發 reload

## 執行入口

- `main.go`
  - 僅呼叫 `cmd.Run()`
- `cmd/root.go`
  - 使用 Cobra 建立 CLI
  - 支援三個參數：
    - `-t`, `--target`
    - `-c`, `--config`
    - `-b`, `--blackbox`
  - 進入 `autoload.AutoLoader(...)`

## 啟動後主流程

1. `config.ConfigInit()` 載入 `yaml/config*.yaml`
2. `server.NewServer()` 建立全域 `Server` 實例
3. `pkg/tool.NewLogger()` 依 `const.debug` 建立 logger
4. 初始化 OpenSearch 與 RabbitMQ 設定
5. 建立 graceful shutdown context
6. 啟動 `handler.ListenRabbitMQ(reload)`
7. 啟動 `handler.BlackboxProcess(...)`
8. 收到 reload 訊號後，取消舊的 blackbox context，重新啟動 probe 流程

## 主要模組

- `pkg/autoload`
  - 負責系統初始化、外部資源掛載、reload 管理
- `handler/blackbox.go`
  - 讀取 YAML、建立 job ticker、啟動 probe goroutine、整理結果
- `exporter`
  - 根據 module prober 類型執行 probe，並轉出指標資料
- `model/prometheusremotewrite`
  - 將 metrics 封裝成 remote write 請求送往 Prometheus
- `model/opensearch.go`
  - 將結果壓縮並批次寫入 OpenSearch
- `handler/mq.go`
  - 監聽 RabbitMQ RPC，接收 YAML 更新並觸發 reload
- `handler/yaml_check`
  - 驗證 target 與 blackbox module 的 YAML 格式
- `blackbox_exporter`
  - 內含 blackbox config 與 probe 邏輯，類似 vendor/內嵌整合層

## 設定檔角色

- `yaml/config.yaml`
  - 外部服務連線資訊與常數設定
- `yaml/target.yaml`
  - 定義 job、scrape interval、module、target、labels/tags
- `blackbox_exporter/blackbox.yaml`
  - 定義可被 target 引用的 probe modules

## 專案現況觀察

- Prometheus remote write 是實際有串接的主輸出之一
- OpenSearch 寫入受 `opensearch.enable` 控制
- RabbitMQ 用於遠端熱更新 YAML
- `http_server/` 目前只有 `/ping` 路由定義，未看到在主流程中啟動 Gin server
- `config.yaml` 存在 `http_server_port`，但目前未見對應 server 啟動流程
- `model/metric/metric.go` 會輸出 `output.txt`，屬於本地除錯副產物

## 文件索引

- `MDS/01_project_overview.md`
- `MDS/02_bootstrap_and_runtime.md`
- `MDS/03_probe_pipeline.md`
- `MDS/04_mq_reload_and_yaml_validation.md`
- `MDS/05_configuration_reference.md`
- `MDS/06_outputs_and_integrations.md`
- `MDS/07_architecture_diagram.md`
- `MDS/08_api_config_field_matrix.md`
