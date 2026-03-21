# 設定檔參考

## config.yaml

用途：定義外部系統連線與執行常數。

主要區塊：

- `opensearch`
  - `index`
  - `enable`
  - 具名節點，例如 `One.host / username / password`
- `rabbitMQ`
  - 具名節點與 exchange、routing key、queue
- `prometheus`
  - `prometheusUrl`
  - `username`
  - `password`
  - `prometheusCert`
  - `insecuretls`
- `const`
  - `httpRetrySecond`
  - `debug`
  - `maxGoroutines`
  - `http_server_port`

### 環境變數替換

`config/config.go` 支援 `${ENV_NAME}` 形式替換。

若該環境變數不存在，會嘗試取同一行 `#` 註解後的值當預設值。

## target.yaml

用途：定義要探測的 job 與 target。

典型結構：

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

關鍵欄位：

- `job_name`
- `metrics_path`
- `params.module`
- `scrape_interval`
- `static_configs.targets`
- `static_configs.labels`
- `static_configs.tags`

## blackbox.yaml

用途：定義可供 target 引用的 modules。

典型 module：

- `http_2xx`
- `tcp_connect`
- `icmp`
- `dns_test`
- `grpc`
- `ntp`

每個 module 至少包含：

- `prober`
- 對應 prober 的設定區塊，例如 `http:`、`tcp:`、`dns:`

### Blackbox Exporter 0.28 已開放欄位

本專案目前已對齊 `github.com/prometheus/blackbox_exporter v0.28.0`，並在 validator 與 sample YAML 開放以下欄位：

- `http.enable_http3`
- `grpc.metadata`
- `tcp.query_response.expect_bytes`

補充限制：

- `grpc.metadata` 需使用 `map[string][]string` 格式，而不是單一字串
- `http.enable_http3` 需為 `bool`
- `tcp.query_response.expect_bytes` 需為字串；是否與 `expect` 並用，請依 probe 語意自行控制

範例：

```yaml
modules:
  http_2xx:
    prober: http
    timeout: 30s
    http:
      preferred_ip_protocol: ip4
      enable_http3: false

  grpc_with_metadata:
    prober: grpc
    grpc:
      tls: true
      metadata:
        authorization:
          - Bearer example-token
        x-tenant-id:
          - demo

  tcp_expect_bytes:
    prober: tcp
    tcp:
      query_response:
        - expect_bytes: "\\x00\\x01"
```

### NTP module

可新增 `prober: ntp` 的 module，支援欄位：

- `preferred_ip_protocol`: `ip4` 或 `ip6`
- `ip_protocol_fallback`: 是否允許 IP 協定 fallback
- `source_ip_address`: 指定來源 IP
- `protocol_version`: NTP version，允許 `2`、`3`、`4`
- `measurement_duration`: 高 drift 時的追加量測時間，例如 `30s`
- `high_drift_threshold`: 觸發追加量測的門檻，例如 `10ms`

範例：

```yaml
modules:
  ntp:
    prober: ntp
    timeout: 5s
    ntp:
      preferred_ip_protocol: ip4
      protocol_version: 4
      measurement_duration: 30s
      high_drift_threshold: 10ms
```
