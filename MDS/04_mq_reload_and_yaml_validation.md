# RabbitMQ 熱更新與 YAML 驗證

## RabbitMQ 監聽

`handler.ListenRabbitMQ()` 會讀取 `RabbitMQQueue` 設定，逐一建立 RPC listener。

目前 queue 名稱若包含：

- `modules`: 視為 blackbox module 更新
- 其他: 視為 target 更新

## 收到訊息後的流程

`handleRabbitMQMessage()` 會：

1. 判斷訊息應覆蓋哪個本地 YAML
2. 驗證 YAML 格式
3. 若驗證失敗，寫入對應 error 檔
4. 若驗證成功，覆蓋正式設定檔
5. 發送 reload 訊號
6. 用 RPC 回覆處理結果

## Target YAML 驗證

`handler/yaml_check/target/check.go` 會檢查：

- 是否存在 `scrape_configs`
- job 是否具備必要欄位
- `params.module` 是否存在且為字串
- module 是否在 `blackbox.yaml` 內存在
- `static_configs.targets` 是否有效

如果某些 job 沒有 target，程式會過濾無效 job，並把結果重寫回 `yaml/target.yaml`。

這代表驗證流程不只是檢查，也可能直接修改本地 target 檔案。

## Blackbox YAML 驗證

`handler/yaml_check/module/check.go` 會檢查：

- `prober` 是否是允許值
- 模組參數是否與 prober 類型相符
- 子區塊是否為空

允許的 prober：

- `http`
- `tcp`
- `grpc`
- `icmp`
- `dns`

若 module 缺少必要子設定，驗證流程也可能重寫 `blackbox_exporter/blackbox.yaml`。

## Reload 行為

AutoLoader 內部維護 `reload chan bool`。

收到訊號後：

- 先取消舊的 blackbox context
- 再用預設檔名 `target.yaml` 與 `blackbox.yaml` 啟動新的 `BlackboxProcess`

因此 MQ 熱更新實際上是覆蓋固定本地檔後再重新載入，而不是重新使用 CLI 指定的自訂檔名。

