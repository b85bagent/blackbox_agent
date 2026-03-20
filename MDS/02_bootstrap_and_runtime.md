# 啟動與執行流程

## CLI 入口

`main.go` 只做一件事：呼叫 `cmd.Run()`。

`cmd/root.go` 使用 Cobra 接受：

- `--target`, `-t`
- `--config`, `-c`
- `--blackbox`, `-b`

預設值分別是：

- `target.yaml`
- `config.yaml`
- `blackbox.yaml`

## AutoLoader 職責

`pkg/autoload/loader.go` 是整個系統真正的啟動核心，負責：

- 載入 config
- 建立全域 `server.Server`
- 掛入 `const`、Prometheus、OpenSearch、RabbitMQ 設定
- 建立 debug logger
- 建立 graceful shutdown context
- 啟動 RabbitMQ 監聽
- 啟動 BlackboxProcess
- 接收 reload 訊號並重建 blackbox context

## Server 物件用途

`server/server.go` 中的 `Server` 是全域共享狀態容器，存放：

- 常數設定 `Constant`
- RabbitMQ 連線設定
- OpenSearch client 與 index
- Prometheus remote write 設定
- logger
- graceful shutdown context

這個結構讓 probe、MQ、輸出模組都能共用同一份執行狀態。

## 關閉流程

`pkg/tool/gs.go` 會監聽：

- `SIGHUP`
- `SIGINT`
- `SIGQUIT`
- `SIGTERM`
- `os.Interrupt`

收到訊號後會取消 context，讓 probe ticker 與上層流程退出。

