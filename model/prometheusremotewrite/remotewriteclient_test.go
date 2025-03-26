package prometheusremotewrite

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

// 測試 NewRemoteWriteClient 正常初始化
func TestNewRemoteWriteClient_WithValidParams(t *testing.T) {
	client, err := NewRemoteWriteClient("http://localhost:9090", "user", "pass", "", "", "", true)
	if err != nil {
		t.Fatalf("Failed to initialize NewRemoteWriteClient: %v", err)
	}
	if client.endpoint != "http://localhost:9090" {
		t.Errorf("Unexpected endpoint. Expected: http://localhost:9090, got: %s", client.endpoint)
	}
}

// 測試 NewRemoteWriteClient 無憑證的情況
func TestNewRemoteWriteClient_WithoutCerts(t *testing.T) {
	client, err := NewRemoteWriteClient("https://example.com", "", "", "", "", "", false)
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}
	if client == nil {
		t.Fatalf("Expected client instance, but got nil")
	}
}

// 測試 NewRemoteWriteClient 當 CA 憑證不存在
func TestNewRemoteWriteClient_InvalidCACert(t *testing.T) {
	_, err := NewRemoteWriteClient("https://example.com", "", "", "", "", "/invalid/path/to/ca.pem", false)
	if err == nil {
		t.Errorf("Expected error due to invalid CA cert path, but got nil")
	}
}

// 測試 NewRemoteWriteClient 當 Client Cert/Key 不匹配
func TestNewRemoteWriteClient_InvalidClientCert(t *testing.T) {
	_, err := NewRemoteWriteClient("https://example.com", "", "", "/invalid/client.crt", "/invalid/client.key", "", false)
	if err == nil {
		t.Errorf("Expected error due to invalid Client Cert/Key, but got nil")
	}
}

// 測試 SendMetrics 成功發送數據
func TestSendMetrics_ValidTimeSeries(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Unexpected HTTP method. Expected POST, got: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK) // 伺服器回傳 200 OK
	}))
	defer mockServer.Close()

	client, _ := NewRemoteWriteClient(mockServer.URL, "", "", "", "", "", true)
	err := client.SendMetrics([]prompb.TimeSeries{})
	if err != nil {
		t.Errorf("SendMetrics failed: %v", err)
	}
}

// 測試 SendMetrics 伺服器回應 500 錯誤
func TestSendMetrics_ServerReturns500(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	client, _ := NewRemoteWriteClient(mockServer.URL, "", "", "", "", "", true)
	err := client.SendMetrics([]prompb.TimeSeries{})
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
}

// 測試 SendMetrics 伺服器回應 403 錯誤
func TestSendMetrics_ServerReturns403(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}))
	defer mockServer.Close()

	client, _ := NewRemoteWriteClient(mockServer.URL, "", "", "", "", "", true)
	err := client.SendMetrics([]prompb.TimeSeries{})
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
}

// 測試 SendMetrics 伺服器回應 404 錯誤
func TestSendMetrics_ServerReturns404(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer mockServer.Close()

	client, _ := NewRemoteWriteClient(mockServer.URL, "", "", "", "", "", true)
	err := client.SendMetrics([]prompb.TimeSeries{})
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
}

// 測試 SendMetrics 空的 timeSeries
func TestSendMetrics_EmptyTimeSeries(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	client, _ := NewRemoteWriteClient(mockServer.URL, "", "", "", "", "", true)
	err := client.SendMetrics([]prompb.TimeSeries{})
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
}

// 測試 SendMetrics 當伺服器關閉
func TestSendMetrics_ServerClosed(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	mockServer.Close() // 立即關閉伺服器

	client, _ := NewRemoteWriteClient(mockServer.URL, "", "", "", "", "", true)
	err := client.SendMetrics([]prompb.TimeSeries{})
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
}

// 測試 insecureTLS=false 的情況
func TestNewRemoteWriteClient_SecureTLS(t *testing.T) {
	client, err := NewRemoteWriteClient("https://example.com", "", "", "", "", "", false)
	if err != nil {
		t.Fatalf("Failed to initialize NewRemoteWriteClient: %v", err)
	}
	if client == nil {
		t.Fatalf("Expected client instance, but got nil")
	}
}

// 測試 HTTP URL 但開啟了 TLS
func TestNewRemoteWriteClient_InvalidHTTPWithTLS(t *testing.T) {
	_, err := NewRemoteWriteClient("http://localhost:9090", "", "", "client.crt", "client.key", "ca.crt", true)
	if err == nil {
		t.Errorf("Expected error due to HTTP URL with TLS enabled, but got nil")
	} else {
		t.Logf("Successfully detected invalid TLS config: %v", err)
	}
}

// 測試使用無效憑證
func TestNewRemoteWriteClient_InvalidCerts(t *testing.T) {
	_, err := NewRemoteWriteClient("https://localhost:9090", "", "", "invalid.crt", "invalid.key", "invalid-ca.crt", true)
	if err == nil {
		t.Errorf("Expected error due to invalid certificates, but got nil")
	} else {
		t.Logf("Successfully detected invalid certificates: %v", err)
	}
}

// 測試 Basic Auth 錯誤
func TestSendMetrics_InvalidAuth(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized) // 401 Unauthorized
	}))
	defer mockServer.Close()

	client, _ := NewRemoteWriteClient(mockServer.URL, "wrongUser", "wrongPass", "", "", "", true)
	err := client.SendMetrics([]prompb.TimeSeries{})
	if err == nil {
		t.Errorf("Expected error due to invalid authentication, but got nil")
	} else {
		t.Logf("Successfully detected authentication error: %v", err)
	}
}

// 測試連線超時
func TestSendMetrics_Timeout(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模擬超時，讓請求等超過 5 秒
		time.Sleep(6 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	client, _ := NewRemoteWriteClient(mockServer.URL, "", "", "", "", "", true)
	client.httpClient.Timeout = 2 * time.Second // 設定 2 秒超時

	err := client.SendMetrics([]prompb.TimeSeries{})
	if err == nil {
		t.Errorf("Expected timeout error, but got nil")
	} else {
		t.Logf("Successfully detected timeout: %v", err)
	}
}
