# Probe 執行流程

## 入口

`handler.BlackboxProcess()` 會先做兩件事：

1. 載入 `blackbox.yaml`
2. 載入 `target.yaml`

成功後進入 `TimeControl()`。

`blackbox.yaml` 雖然仍是一份檔案，但在載入時會被 `internal/blackboxadapter` 分成兩條路：

- upstream blackbox_exporter 可理解的 module 設定
- 本專案自定義的 `ntp` 設定

## TimeControl

`TimeControl()` 會讀取 `scrape_configs`，每個 job 解析：

- `job_name`
- `scrape_interval`
- `metrics_path`
- `params.module`
- `static_configs`

每個 job 會建立一個 ticker：

- 啟動後立即先跑一次 `dataResolve()`
- 之後依 `scrape_interval` 重複執行
- context cancel 時停止

## dataResolve

`dataResolve()` 是單一 job 的核心執行器，負責：

- 讀取 job metadata
- 取出所有 target
- 根據 `const.maxGoroutines` 以 semaphore 限制同時執行數
- 為每個 target 啟動 goroutine 做 probe
- 收集 probe 結果並合併

## Probe 分派

`exporter.CheckModuleAndDoProbe()` 會依 module 的 `prober` 欄位決定使用：

- `ProbeHTTP`
- `ProbeTCP`
- `ProbeICMP`
- `ProbeDNS`
- `ProbeGRPC`
- adapter custom `NTP` runner

對應關係：

- `http/tcp/icmp/dns/grpc` 由官方 `github.com/prometheus/blackbox_exporter` backend 執行
- `ntp` 由 `internal/blackboxadapter/custom_ntp.go` 執行

## blackbox.yaml 載入分流

`internal/blackboxadapter.UpstreamConfigLoader` 在 `Reload()` 時會做三件事：

1. 先從原始 YAML 抽出 `prober: ntp` module 的 `ntp:` 設定
2. 產生一份移除 `ntp:` 欄位的暫存 YAML
3. 將暫存 YAML 交給官方 `blackbox_exporter/config` 載入

這樣可以保留單一 `blackbox.yaml`，同時避免官方 upstream config parser 因不認得 `ntp:` 欄位而失敗。

## 指標收集

`exporter/doProbe()` 會：

- 建立 Prometheus registry
- 註冊 `probe_success` 與 `probe_duration_seconds`
- 以 timeout context 執行對應 prober
- Gather 所有 metrics
- 呼叫 `model/metric.ProcessMetrics()` 轉成 `prompb.TimeSeries`
- 送到 remote write endpoint

其中：

- upstream module 仍透過 adapter 內包住的官方 `config.Module` 執行
- `ntp` probe 則改用 adapter 自有 `NTPProbeConfig`

## 結果整理

成功 probe 後，系統會把以下欄位組進文件：

- `jobName`
- `target`
- `labels`
- `tags`
- `params`
- `scrape_interval`
- `metrics_path`
- 各 probe 指標值
- `result`

若 target probe 失敗，該 target 不會被併入 OpenSearch bulk payload。
