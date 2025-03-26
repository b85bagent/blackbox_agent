package exporter

import (
	"blackbox_agent/server"
	"log"
	"time"

	bep "blackbox_agent/blackbox_exporter/prober"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	Probers = map[string]bep.ProbeFn{
		"http": bep.ProbeHTTP,
		"tcp":  bep.ProbeTCP,
		"icmp": bep.ProbeICMP,
		"dns":  bep.ProbeDNS,
		"grpc": bep.ProbeGRPC,
	}
)

type HostMonitor struct{}

// 創建結構體對應指標
func NewHostMonitor() *HostMonitor {
	return &HostMonitor{}
}

// 實現Describe接口，傳遞指標到channel
func (h *HostMonitor) Describe(ch chan<- *prometheus.Desc) {}

// 實現collect接口，執行抓取函數返回數據
func (h *HostMonitor) Collect(ch chan<- prometheus.Metric) {}

func timeOutSetting() time.Duration {

	timeoutSetting, ok := server.GetServerInstance().GetConst()["httpRetrySecond"]
	if !ok {
		log.Println("timeoutSetting Get failed")
		return 0
	}

	timeout := time.Duration(timeoutSetting.(int)) * time.Second // 設定超時時間 5秒 修改請見config.yaml

	return timeout
}
