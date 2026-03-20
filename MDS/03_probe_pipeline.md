# Probe 執行流程

## 入口

`handler.BlackboxProcess()` 會先做兩件事：

1. 載入 `blackbox.yaml`
2. 載入 `target.yaml`

成功後進入 `TimeControl()`。

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
- `ProbeNTP`

對應實作位於 `blackbox_exporter/prober/`。

## 指標收集

`exporter/doProbe()` 會：

- 建立 Prometheus registry
- 註冊 `probe_success` 與 `probe_duration_seconds`
- 以 timeout context 執行對應 prober
- Gather 所有 metrics
- 呼叫 `model/metric.ProcessMetrics()` 轉成 `prompb.TimeSeries`
- 送到 remote write endpoint

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
