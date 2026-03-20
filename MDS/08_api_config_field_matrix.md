# API / 設定欄位對照表

本文件整理目前專案內三份核心設定檔的欄位定義：

- `yaml/config.yaml`
- `yaml/target.yaml`
- `blackbox_exporter/blackbox.yaml`

說明基準來自：

- 程式實際讀取邏輯
- YAML 驗證器
- 範例檔案

若三者不一致，會在備註欄直接標明。

補充：

- `blackbox_exporter/blackbox.yaml` 仍是單一設定檔入口
- 載入時由 `internal/blackboxadapter` 先抽出 `prober: ntp` 的 `ntp:` 區塊
- 清洗後的 YAML 才會交給官方 `github.com/prometheus/blackbox_exporter/config` 解析

## 1. config.yaml

檔案用途：定義外部系統連線資訊與執行常數。

實作來源：

- `config/config.go`
- `config/struct.go`
- `pkg/autoload/loader.go`

### 根層欄位

| 欄位 | 型別 | 必填 | 說明 | 程式用途 |
|---|---|---|---|---|
| `opensearch` | object | 否 | OpenSearch 相關設定 | 初始化 OpenSearch client 與 bulk index |
| `rabbitMQ` | object | 否 | RabbitMQ 相關設定 | 啟動 RPC listener |
| `prometheus` | object | 否 | Prometheus remote write 設定 | 建立 remote write client |
| `const` | map[string]interface{} | 否 | 執行時常數 | logger、timeout、併發數等 |

### `opensearch`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `opensearch.index` | string | 否 | OpenSearch index 名稱 | 例如 `lex-test12-%{YYYY-MM-DD}` |
| `opensearch.enable` | bool | 否 | 是否真的寫入 OpenSearch | `false` 時只做 probe 與 Prometheus 輸出 |
| `opensearch.<name>` | object | 否 | 具名 OpenSearch 連線設定 | 例如 `One` |
| `opensearch.<name>.host` | []string | 否 | OpenSearch URL 清單 | 會傳入 opensearch client `Addresses` |
| `opensearch.<name>.username` | string | 否 | 帳號 |  |
| `opensearch.<name>.password` | string | 否 | 密碼 |  |

### `rabbitMQ`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `rabbitMQ.<name>` | object | 否 | 具名 RabbitMQ 連線設定 | 例如 `One` |
| `rabbitMQ.<name>.host` | []string | 否 | RabbitMQ broker 清單 | 目前實作只使用第一筆 `Host[0]` |
| `rabbitMQ.<name>.username` | string | 否 | 帳號 |  |
| `rabbitMQ.<name>.password` | string | 否 | 密碼 |  |
| `rabbitMQ.<name>.RabbitMQExchange` | string | 否 | exchange 名稱 |  |
| `rabbitMQ.<name>.RabbitMQRoutingKey` | string | 否 | routing key |  |
| `rabbitMQ.<name>.RabbitMQQueue` | []string | 否 | 要監聽的 queue 清單 | 包含 `modules` 字樣時視為 module 更新 |

### `prometheus`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `prometheus.prometheusUrl` | string | 實務上是 | Remote write endpoint | 若空值，remote write 會失敗 |
| `prometheus.username` | string | 否 | Basic auth 帳號 |  |
| `prometheus.password` | string | 否 | Basic auth 密碼 |  |
| `prometheus.prometheusCert` | string | 否 | CA 憑證檔路徑 | YAML tag 為 `prometheus_cert`，對應 struct 欄位 `PrometheusCert` |
| `prometheus.insecuretls` | bool | 否 | 是否跳過 TLS 驗證 | YAML tag 為 `insecuretls` |

### `const`

| 欄位 | 型別 | 必填 | 說明 | 程式使用位置 |
|---|---|---|---|---|
| `const.httpRetrySecond` | int | 實務上是 | 單次 probe timeout 秒數 | `exporter/timeOutSetting()` |
| `const.debug` | bool | 實務上是 | 是否輸出 debug log | `pkg/tool/log.go` |
| `const.maxGoroutines` | int | 實務上是 | target probe 併發上限 | `handler/dataResolve()` |
| `const.http_server_port` | string | 否 | HTTP server port | 目前未見主流程實際使用 |

### 環境變數替換規則

| 寫法 | 行為 |
|---|---|
| `${ENV_NAME}` | 以環境變數值替換 |
| `${ENV_NAME}  # default` | 若環境變數不存在，取 `#` 後內容作為預設值 |

注意：

- 這個替換只實作在 `config.yaml` 讀取流程
- `target.yaml` 與 `blackbox.yaml` 不走同一套環境變數替換

## 2. target.yaml

檔案用途：定義 job 排程、對應 module、targets、labels、tags。

實作來源：

- `handler/blackbox.go`
- `handler/yaml_check/target/check.go`

### 根層欄位

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `scrape_configs` | array | 是 | job 清單 |

### `scrape_configs[]`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `job_name` | string | 是 | job 名稱 | 寫入結果欄位 `jobName` |
| `scrape_interval` | string | 是 | 執行間隔 | 需可被 `time.ParseDuration` 解析，例如 `10s` |
| `metrics_path` | string | 是 | 指標路徑描述 | 實作中主要作為 metadata 保存 |
| `params` | object | 是 | probe 參數容器 | 目前主要使用 `module` |
| `static_configs` | array | 是 | target 群組 | 空值時驗證器可能重寫檔案並過濾 job |

### `scrape_configs[].params`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `module` | array | 是 | 對應 blackbox module 名稱清單 | 目前執行時只取第一個元素 `module[0]` |

### `scrape_configs[].static_configs[]`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `targets` | []string | 是 | target 清單 | 每個 target 會各自 probe |
| `labels` | map | 否 | 額外 labels | 會附加到 Prometheus labels 與 OpenSearch doc |
| `tags` | map | 否 | 額外 tags | 執行邏輯有支援 |
| `tag` | string | 範例曾出現 | 舊格式單值 tag | 驗證 struct 有 `Tag`，但執行邏輯主要讀 `tags` |

### `labels` / `tags`

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `labels.<key>` | string | 否 | 自訂 label |
| `tags.<key>` | string | 否 | 自訂 tag |

### target.yaml 最小可用範例

```yaml
scrape_configs:
  - job_name: icmp
    scrape_interval: 30s
    metrics_path: /probe/icmp
    params:
      module:
        - icmp
    static_configs:
      - targets:
          - 127.0.0.1
        labels:
          device_name: local_test
        tags:
          env: local
```

### target 驗證規則摘要

| 規則 | 說明 |
|---|---|
| `scrape_configs` 不可為空 | 否則驗證失敗 |
| `job_name` / `scrape_interval` / `metrics_path` 必須存在 | 缺一即失敗 |
| `params.module` 必須存在且每個值為 string | 否則失敗 |
| `params.module` 必須存在於 `blackbox.yaml` 的 `modules` 中 | 否則失敗 |
| `static_configs.targets` 不可為空 | 空的 job 可能被過濾並回寫檔案 |
| `labels` 若存在，驗證器只明確檢查 `check` 欄位舊格式 | 但執行邏輯可接受泛型 map[string]string |

## 3. blackbox.yaml

檔案用途：定義可被 `target.yaml` 引用的 probe modules。

實作來源：

- `handler/yaml_check/module/check.go`
- `handler/yaml_check/module/http.go`
- `handler/yaml_check/module/dns.go`
- `handler/yaml_check/module/icmp.go`
- `handler/yaml_check/module/tcp.go`
- `handler/yaml_check/module/grpc.go`

### 根層欄位

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `modules` | map | 是 | module 名稱到 module 設定的映射 |

### `modules.<moduleName>`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `prober` | string | 是 | 指定探測器類型 | 只允許 `http` `tcp` `grpc` `icmp` `dns` |
| `timeout` | string | 否 | probe timeout | 須為 duration 字串 |
| `http` | object | `prober: http` 時使用 | HTTP probe 設定 |  |
| `tcp` | object | `prober: tcp` 時使用 | TCP probe 設定 |  |
| `icmp` | object | `prober: icmp` 時使用 | ICMP probe 設定 |  |
| `dns` | object | `prober: dns` 時使用 | DNS probe 設定 |  |
| `grpc` | object | `prober: grpc` 時使用 | gRPC probe 設定 |  |

## 3.1 HTTP module 欄位

### `modules.<name>.http`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `valid_http_versions` | []string | 否 | 允許的 HTTP 版本 | 驗證器接受字串陣列 |
| `valid_status_codes` | []int | 否 | 允許的狀態碼 |  |
| `method` | string | 否 | HTTP 方法 | 驗證器允許 `GET POST PUT DELETE PATCH HEAD OPTIONS` |
| `headers` | map[string]string | 否 | 自訂 header |  |
| `follow_redirects` | bool | 否 | 是否跟隨轉址 | 若欄位存在但不是 bool 會失敗 |
| `fail_if_ssl` | bool | 否 | 若目標是 SSL 則失敗 |  |
| `fail_if_not_ssl` | bool | 否 | 若目標不是 SSL 則失敗 |  |
| `fail_if_body_matches_regexp` | []string | 否 | body 命中 regex 視為失敗 |  |
| `fail_if_body_not_matches_regexp` | []string | 否 | body 未命中 regex 視為失敗 |  |
| `fail_if_header_matches` | []map | 否 | header 命中條件視為失敗 | 驗證器只檢查每項是 map |
| `fail_if_header_not_matches` | []map | 否 | header 未命中條件視為失敗 | 驗證器只檢查每項是 map |
| `tls_config` | object | 否 | TLS 設定 | 見 TLS 小節 |
| `basic_auth` | object | 否 | Basic auth 設定 | 需含 `username`、`password` |
| `authorization` | object | 否 | Authorization 設定 | 驗證器要求 `type`、`credentials`、`credentials_file` 都是 string |
| `proxy_url` | string | 否 | Proxy URL | 不可為空字串 |
| `no_proxy` | string | 否 | 不走 proxy 的目標 | 不可為空字串 |
| `proxy_from_environment` | bool | 否 | 是否採用環境變數 proxy |  |
| `proxy_connect_headers` | map[string]string | 否 | Proxy connect headers |  |
| `skip_resolve_phase_with_proxy` | bool | 否 | 使用 proxy 時是否跳過 resolve 階段 |  |
| `oauth2` | object | 否 | OAuth2 設定 | 見 OAuth2 小節 |
| `enable_http2` | bool | 否 | 是否啟用 HTTP/2 |  |
| `preferred_ip_protocol` | string | 否 | 偏好 IP 版本 | 只允許 `ip4` `ip6` |
| `ip_protocol_fallback` | bool | 否 | IP 協議 fallback |  |
| `body` | string | 否 | request body | 不可為空字串 |
| `body_file` | string | 否 | request body 檔案路徑 | 不可為空字串 |

### `modules.<name>.http.tls_config`

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `insecure_skip_verify` | bool | 否 | 是否跳過憑證驗證 |
| `ca_file` | string | 否 | CA 憑證路徑 |
| `cert_file` | string | 否 | Client cert 路徑 |
| `key_file` | string | 否 | Client key 路徑 |
| `server_name` | string | 否 | TLS server name |
| `min_version` | string | 否 | 最低 TLS 版本，允許 `TLS10` `TLS11` `TLS12` `TLS13` |
| `max_version` | string | 否 | 最高 TLS 版本，允許 `TLS10` `TLS11` `TLS12` `TLS13` |

### `modules.<name>.http.oauth2`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `client_id` | string | 是 | OAuth2 client id |  |
| `client_secret` | string | 否 | OAuth2 secret | 與 `client_secret_file` 互斥 |
| `client_secret_file` | string | 否 | secret 檔案路徑 | 與 `client_secret` 互斥 |
| `scopes` | []string | 否 | scopes |  |
| `token_url` | string | 是 | token endpoint |  |
| `endpoint_params` | map[string]string | 否 | 額外參數 |  |

## 3.2 TCP module 欄位

### `modules.<name>.tcp`

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `preferred_ip_protocol` | string | 否 | 偏好 IP 協議 |
| `ip_protocol_fallback` | bool | 否 | 是否允許 fallback |
| `source_ip_address` | string | 否 | 指定來源 IP |
| `query_response` | array | 否 | TCP 互動腳本 |
| `tls` | bool | 否 | 是否使用 TLS |
| `tls_config` | object | 否 | TLS 設定，結構同 HTTP TLS |

### `modules.<name>.tcp.query_response[]`

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `expect` | string | 否 | 預期回應 regex |
| `send` | string | 否 | 要送出的字串 |
| `starttls` | bool | 否 | 是否在此步切換 starttls |

## 3.3 ICMP module 欄位

### `modules.<name>.icmp`

| 欄位 | 型別 | 必填 | 說明 | 備註 |
|---|---|---|---|---|
| `preferred_ip_protocol` | string | 否 | 偏好 IP 協議 |  |
| `ip_protocol_fallback` | bool | 否 | 是否允許 fallback |  |
| `source_ip_address` | string | 否 | 指定來源 IP |  |
| `dont_fragment` | bool | 否 | 是否設定 DF |  |
| `payload_size` | int | 否 | payload 大小 |  |
| `ttl` | int | 否 | TTL | 必須在 `0..255` |

## 3.4 DNS module 欄位

### `modules.<name>.dns`

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `query_name` | string | 是 | 查詢名稱 |
| `preferred_ip_protocol` | string | 否 | 偏好 IP 協議 |
| `ip_protocol_fallback` | bool | 否 | 是否允許 fallback |
| `source_ip_address` | string | 否 | 指定來源 IP |
| `transport_protocol` | string | 否 | 傳輸協議 |
| `dns_over_tls` | bool | 否 | 是否啟用 DoT |
| `tls_config` | object | 否 | TLS 設定，結構同 HTTP TLS |
| `query_type` | string | 否 | DNS 類型，例如 `A` |
| `query_class` | string | 否 | DNS class，例如 `IN` |
| `recursion_desired` | bool | 否 | 是否要求遞迴 |
| `valid_rcodes` | []string | 否 | 合法回應碼 |
| `validate_answer_rrs` | object | 否 | 驗證 answer RRs 規則 |
| `validate_authority_rrs` | object | 否 | 驗證 authority RRs 規則 |
| `validate_additional_rrs` | object | 否 | 驗證 additional RRs 規則 |

### `validate_*_rrs`

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `fail_if_matches_regexp` | []string | 否 | 命中任一 regex 即失敗 |
| `fail_if_all_match_regexp` | []string | 否 | 全部命中才失敗 |
| `fail_if_not_matches_regexp` | []string | 否 | 未命中則失敗 |
| `fail_if_none_matches_regexp` | []string | 否 | 全部未命中則失敗 |

## 3.5 gRPC module 欄位

### `modules.<name>.grpc`

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `service` | string | 否 | 指定 gRPC service |
| `preferred_ip_protocol` | string | 否 | 偏好 IP 協議 |
| `ip_protocol_fallback` | bool | 否 | 是否允許 fallback |
| `tls` | bool | 否 | 是否使用 TLS |
| `tls_config` | object | 否 | TLS 設定，結構同 HTTP TLS |

## 4. 常見注意事項

| 主題 | 說明 |
|---|---|
| module 陣列 | `target.yaml` 的 `params.module` 雖然是陣列，但執行時目前只使用第一個元素 |
| `tags` vs `tag` | 範例檔同時出現兩種格式；執行邏輯以 `tags` map 為主 |
| reload 行為 | MQ reload 後固定重新讀 `target.yaml` 與 `blackbox.yaml` |
| `http_server_port` | 設定檔有欄位，但目前未見主流程實際啟動 HTTP server |
| `output.txt` | metrics 處理流程目前會產生本地 `output.txt` |
| 欄位完整度 | `blackbox.yaml` 真正可用欄位受官方 blackbox_exporter schema 與 adapter 自定義 `ntp` 分流邏輯共同影響，本表以專案內驗證器與樣板為主 |

## 5. 建議使用方式

- 新增 probe 類型時，先在 `blackbox.yaml` 建 module
- 再在 `target.yaml` 的 `params.module` 引用 module 名稱
- 若要附帶自訂標籤，優先使用 `labels` 與 `tags`
- 若要查欄位是否真正有被程式使用，優先對照：
  - `handler/blackbox.go`
  - `exporter/handler.go`
  - `handler/yaml_check/module/*`
  - `handler/yaml_check/target/*`
