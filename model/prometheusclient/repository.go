package prometheusclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	tools "github.com/b85bagent/tools/prometheus"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

// 透過 HTTP 客戶端針對 Prometheus Remote Write API 發送 TimeSeries 數據
func SendMetrics(c *tools.Client, timeSeries []prompb.TimeSeries) error {
	// 創建 Prometheus WriteRequest
	writeRequest := &prompb.WriteRequest{Timeseries: timeSeries}

	// 轉換為 Protobuf
	data, err := writeRequest.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal WriteRequest: %w", err)
	}

	// 使用 Snappy 壓縮
	compressedData := snappy.Encode(nil, data)

	// 創建 HTTP Request
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 設定 Header
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")

	// 如果有 Basic Auth
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	// 發送請求
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 讀取回應
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to push metrics, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
