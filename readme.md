# Agent Application

Agent Application 是一個基於Go語言寫的CLI程式，用於設定YAML配置檔案並執行一些特定的任務。

## 主要功能

- 讀取並解析配置文件（YAML格式）。
- 在指定的interval間隔下進行probe操作並將結果寫入OpenSearch。

## 如何使用

### 命令參數

這個程式提供了以下的命令行參數：

- `-t` 或 `--target`：指定要載入的Target YAML檔案（預設為 `target.yaml`）。
- `-c` 或 `--config`：指定要載入的設定 Config YAML 檔案（預設為 `config.yaml`）。
- `-b` 或 `--blackbox`：指定要載入的設定 Blackbox YAML 檔案（預設為 `blackbox.yaml`）。

### 執行

編譯並執行主程式：

```bash
go build -o agent
./agent

./agent -t target777.yaml 
./agent -t target777.yaml -c config777.yaml 
./agent -t target777.yaml -c config777.yaml -b blackbox777.yaml
```

### 配置文件

你需要為每個 `-t`、`-c` 和 `-b` 參數提供一個對應的YAML文件。這些文件將被程式讀取並用於設定伺服器實例和probe任務。

#### 配置文件位置

- `config` : /model/yaml/config
- `target` : /model/yaml/target
- `blackbox` : /blackbox_exporter

## 程式結構與流程

這個程式主要包含以下幾個部分：

- `main.go`：程式的入口點，會調用 `cmd.Run()` 函數開始程式的運行。

- `cmd.Run()`：建立了cobra命令，並設定了三個標誌（targetFile、configFile、blackboxFile）。使用者可以指定不同的YAML配置文件。

- `autoload.AutoLoader()`：負責初始化配置、創建新的伺服器實例、設定日誌記錄器等。此函數最後會啟動一個HTTP伺服器。此外，該函數也會啟動兩個goroutine，一個用於第一次啟動 `BlackboxProcess`，另一個則是監聽reload channel，如果收到訊號，就會`重新`啟動新的 `BlackboxProcess`。

- `handler.BlackboxProcess()`：是一個從target YAML和blackbox YAML配置文件讀取設定並進行probe的過程。此函數會根據配置文件設定的間隔定期啟用相對應數量的goroutine並調用 `dataResolve()` 函數。

- `dataResolve()`: 是實際執行probe和將結果寫入OpenSearch的函數。每個target都會在自己的goroutine中執行此函數Probe以及收集結果並且寫入Opensearch。

## 開發者資訊

- 作者：Lex
- 聯絡方式：<tsunglintsai@systex.com>
