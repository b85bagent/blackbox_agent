# Blackbox Agent

`blackbox_agent` 是一個以 Go 撰寫的黑箱探測代理程式。它會讀取本地 YAML 設定，定期對指定 target 執行 probe，並把結果送到 Prometheus Remote Write 與 OpenSearch；同時也支援透過 RabbitMQ RPC 熱更新 target 與 module 設定。

如果你是第一次接手這個專案，先看這份 README，再看 [AGENTS.md](/home/systexadmin/blackbox_agent/AGENTS.md) 與 [07_architecture_diagram.md](/home/systexadmin/blackbox_agent/MDS/07_architecture_diagram.md)。

## 專案能做什麼

- 依 `target.yaml` 定義定期對 target 做 blackbox probe
- 支援 `http`、`tcp`、`icmp`、`dns`、`grpc` 五種 prober
- 將 probe 指標轉成 Prometheus remote write 格式推送
- 視設定將 probe 結果寫入 OpenSearch
- 透過 RabbitMQ RPC 接收新的 YAML 設定並觸發 reload

## 專案入口

- [main.go](/home/systexadmin/blackbox_agent/main.go)
  - 呼叫 `cmd.Run()`
- [cmd/root.go](/home/systexadmin/blackbox_agent/cmd/root.go)
  - 處理 CLI 參數
- [pkg/autoload/loader.go](/home/systexadmin/blackbox_agent/pkg/autoload/loader.go)
  - 真正的初始化入口

## 快速開始

### 1. 準備設定檔

預設會讀以下檔案：

- [yaml/config.yaml](/home/systexadmin/blackbox_agent/yaml/config.yaml)
- [yaml/target.yaml](/home/systexadmin/blackbox_agent/yaml/target.yaml)
- [blackbox_exporter/blackbox.yaml](/home/systexadmin/blackbox_agent/blackbox_exporter/blackbox.yaml)

`config.yaml` 中部分欄位支援 `${ENV_NAME}` 環境變數替換。

### 2. 建置與執行

```bash
go build -o blackbox_agent
./blackbox_agent
```

或指定自訂檔名：

```bash
./blackbox_agent -t target_sample.yaml
./blackbox_agent -t target_sample.yaml -c confignew.yaml
./blackbox_agent -t target_sample.yaml -c confignew.yaml -b blackbox_example.yaml
```

## CLI 參數

- `-t`, `--target`
  - 指定 target YAML，預設 `target.yaml`
- `-c`, `--config`
  - 指定 config YAML，預設 `config.yaml`
- `-b`, `--blackbox`
  - 指定 blackbox module YAML，預設 `blackbox.yaml`

## 執行流程

1. 載入 config 並建立全域 server 狀態
2. 初始化 OpenSearch、RabbitMQ、Prometheus 設定
3. 建立 shutdown context 與 debug logger
4. 啟動 RabbitMQ listener
5. 啟動 blackbox probe 排程
6. 每個 job 依 `scrape_interval` 定期執行 probe
7. 每個 target 的結果送往 Prometheus，成功資料可再寫入 OpenSearch
8. 若 RabbitMQ 收到新的 YAML，覆蓋本地檔案後重啟 blackbox 流程

## 重要設定檔

### config.yaml

用途：定義外部依賴與執行參數。

主要欄位：

- `opensearch`
- `rabbitMQ`
- `prometheus`
- `const`

其中 `const` 內較重要的是：

- `httpRetrySecond`
- `debug`
- `maxGoroutines`
- `http_server_port`

### target.yaml

用途：定義要被排程探測的 job 與 target。

最小範例：

```yaml
scrape_configs:
  - job_name: icmp
    metrics_path: /probe/icmp
    params:
      module:
        - icmp
    scrape_interval: 30s
    static_configs:
      - labels:
          device_name: local_test
        targets:
          - 127.0.0.1
```

### blackbox.yaml

用途：定義 module 與其 prober 細節，例如 timeout、http/tcp/dns 子設定。

## 核心目錄

- [cmd](/home/systexadmin/blackbox_agent/cmd)
  - CLI 入口
- [config](/home/systexadmin/blackbox_agent/config)
  - 設定讀取與結構定義
- [pkg/autoload](/home/systexadmin/blackbox_agent/pkg/autoload)
  - 啟動、reload、資源初始化
- [handler](/home/systexadmin/blackbox_agent/handler)
  - probe 主流程、MQ、YAML 驗證
- [exporter](/home/systexadmin/blackbox_agent/exporter)
  - module 分派、probe 執行、metrics 收集
- [model](/home/systexadmin/blackbox_agent/model)
  - OpenSearch 與 Prometheus remote write
- [blackbox_exporter](/home/systexadmin/blackbox_agent/blackbox_exporter)
  - 保留中的 blackbox YAML、example 與 runtime 相關資產
- [internal/blackboxadapter](/home/systexadmin/blackbox_agent/internal/blackboxadapter)
  - blackbox backend adapter、官方 upstream 整合與 custom NTP probe
- [yaml](/home/systexadmin/blackbox_agent/yaml)
  - config/target 樣板與本地設定
- [MDS](/home/systexadmin/blackbox_agent/MDS)
  - 專案說明文件

## 新人建議閱讀順序

1. [readme.md](/home/systexadmin/blackbox_agent/readme.md)
2. [AGENTS.md](/home/systexadmin/blackbox_agent/AGENTS.md)
3. [02_bootstrap_and_runtime.md](/home/systexadmin/blackbox_agent/MDS/02_bootstrap_and_runtime.md)
4. [03_probe_pipeline.md](/home/systexadmin/blackbox_agent/MDS/03_probe_pipeline.md)
5. [04_mq_reload_and_yaml_validation.md](/home/systexadmin/blackbox_agent/MDS/04_mq_reload_and_yaml_validation.md)
6. [07_architecture_diagram.md](/home/systexadmin/blackbox_agent/MDS/07_architecture_diagram.md)

## 開發與維運注意事項

- RabbitMQ 熱更新是覆蓋本地固定檔案後 reload，不是保留 CLI 自訂檔名重新載入
- `http_server/` 目前只有 `/ping` 路由定義，未見主流程實際啟動
- `model/metric/metric.go` 會輸出 `output.txt`，屬於本地除錯副產物
- OpenSearch 寫入受 `opensearch.enable` 控制
- target 或 module YAML 驗證失敗時，會寫入對應 `_error.yaml`

## 文件索引

- [AGENTS.md](/home/systexadmin/blackbox_agent/AGENTS.md)
- [01_project_overview.md](/home/systexadmin/blackbox_agent/MDS/01_project_overview.md)
- [02_bootstrap_and_runtime.md](/home/systexadmin/blackbox_agent/MDS/02_bootstrap_and_runtime.md)
- [03_probe_pipeline.md](/home/systexadmin/blackbox_agent/MDS/03_probe_pipeline.md)
- [04_mq_reload_and_yaml_validation.md](/home/systexadmin/blackbox_agent/MDS/04_mq_reload_and_yaml_validation.md)
- [05_configuration_reference.md](/home/systexadmin/blackbox_agent/MDS/05_configuration_reference.md)
- [06_outputs_and_integrations.md](/home/systexadmin/blackbox_agent/MDS/06_outputs_and_integrations.md)
- [07_architecture_diagram.md](/home/systexadmin/blackbox_agent/MDS/07_architecture_diagram.md)
- [08_api_config_field_matrix.md](/home/systexadmin/blackbox_agent/MDS/08_api_config_field_matrix.md)
