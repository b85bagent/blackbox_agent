package prometheusremotewrite

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

// RemoteWriteClient 負責發送 Remote Write 請求
type RemoteWriteClient struct {
	endpoint    string
	username    string
	password    string
	clientCert  string
	clientKey   string
	caCert      string
	insecureTLS bool
	httpClient  *http.Client
}

// NewRemoteWriteClient 創建 Remote Write 客戶端
func NewRemoteWriteClient(endpoint, username, password, clientCert, clientKey, caCert string, insecureTLS bool) (*RemoteWriteClient, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: insecureTLS}

	// 如果提供 CA 憑證，則加載 CA
	if caCert != "" {
		caCertData, err := os.ReadFile(caCert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCertData)
		tlsConfig.RootCAs = caCertPool
	}

	// 如果提供 Client Cert，則加載
	if clientCert != "" && clientKey != "" {
		cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 建立 HTTP 客戶端
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}

	return &RemoteWriteClient{
		endpoint:    endpoint,
		username:    username,
		password:    password,
		clientCert:  clientCert,
		clientKey:   clientKey,
		caCert:      caCert,
		insecureTLS: insecureTLS,
		httpClient:  client,
	}, nil
}

// SendMetrics 發送 TimeSeries 數據到 Remote Write
func (r *RemoteWriteClient) SendMetrics(timeSeries []prompb.TimeSeries) error {
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
	req, err := http.NewRequest("POST", r.endpoint, bytes.NewBuffer(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 設定 Header
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")

	// 如果有 Basic Auth
	if r.username != "" && r.password != "" {
		req.SetBasicAuth(r.username, r.password)
	}

	// 發送請求
	resp, err := r.httpClient.Do(req)
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
