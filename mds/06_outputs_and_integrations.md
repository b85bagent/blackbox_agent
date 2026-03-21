# 輸出與外部整合

## Prometheus Remote Write

`exporter/doProbe()` 在每次 probe 後都會：

- 收集 metrics
- 轉為 `prompb.TimeSeries`
- 建立 `RemoteWriteClient`
- POST 到 `prometheus.prometheusUrl`

附加 labels 來源包括：

- hostname
- target
- probe
- jobName
- target labels
- target tags
- blackbox 產生的 metric labels

## OpenSearch

`handler/dataResolve()` 會把成功 probe 的結果壓成 bulk 字串。

若 `opensearch.enable = true`，則呼叫 `model.BulkInsert()` 寫入 OpenSearch。

相關資訊來自：

- `config.opensearch.index`
- `config.opensearch.One.*`

## RabbitMQ RPC

RabbitMQ 用途不是傳送 probe 結果，而是：

- 接收新的 target/module YAML
- 驗證格式
- 覆蓋本地設定檔
- 透過 RPC 回覆成功或失敗

## 本地輸出

目前程式還會產生一些本地副產物：

- `output.txt`
  - `model/metric/metric.go` 每次處理 metrics 會覆蓋輸出
- `yaml/target_error.yaml`
  - target YAML 驗證失敗時寫入
- `blackbox_exporter/blackbox_error.yaml`
  - module YAML 驗證失敗時寫入

## 目前未完整串接的部分

- `http_server/` 只定義了 `/ping` 路由，未看到在主流程中啟動
- `const.http_server_port` 已存在，但未見實際使用

這表示目前專案主功能重心仍是：

- CLI 啟動
- Probe 排程
- Prometheus/OpenSearch 輸出
- RabbitMQ 熱更新

