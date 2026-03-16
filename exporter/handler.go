package exporter

import (
	"blackbox_agent/model/metric"
	promeRepo "blackbox_agent/model/prometheusclient"
	bec "blackbox_agent/pkg/blackbox_exporter/config"
	bep "blackbox_agent/pkg/blackbox_exporter/prober"
	"blackbox_agent/server"
	"context"
	"errors"
	"log"
	"time"

	tools "github.com/b85bagent/tools/prometheus"
	logger "github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

// 確認module類型，給予不同的Probe
func CheckModuleAndDoProbe(module string, data map[string]interface{}, target string, sc *bec.SafeConfig, label, tags map[string]interface{}) (resultData map[string]interface{}, err error) {

	result, err := comparisonConfigAndDoProbe(data, module, target, sc, label, tags)
	if err != nil {
		log.Println("comparisonConfig error: ", err)
		return nil, err
	}

	return result, nil
}

// 比對yaml檔內容，並且Probe
func comparisonConfigAndDoProbe(data map[string]interface{}, m, target string, sc *bec.SafeConfig, label, tags map[string]interface{}) (resultData map[string]interface{}, err error) {

	//comparisonConfig
	// sc.Lock()
	module, ok := sc.C.Modules[m]
	// sc.Unlock()

	if !ok {

		return nil, errors.New("Module " + m + " not found")
	}

	prober, ok := Probers[module.Prober]

	if !ok {

		return nil, errors.New("Prober: " + module.Prober + "not found")
	}

	//doProbe
	result, errProbe := doProbe(data, module, prober, target, label, tags)
	if errProbe != nil {

		log.Println("Probe failed: ", errProbe)
		return nil, err
	}

	return result, nil
}

// Probe
func doProbe(data map[string]interface{}, module bec.Module, prober bep.ProbeFn, target string, label, tags map[string]interface{}) (resultData map[string]interface{}, err error) {

	logger := logger.NewNopLogger()

	probeSuccessGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Displays whether or not the probe was a success",
	})

	probeDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})

	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(probeSuccessGauge)
	registry.MustRegister(probeDurationGauge)

	timeout := timeOutSetting()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// PrometheusEnable := server.GetServerInstance().GetPrometheusEnable()

	start := time.Now()
	success := prober(ctx, target, module, registry, logger)
	if success {
		probeSuccessGauge.Set(1)
	}

	duration := time.Since(start).Seconds()
	probeDurationGauge.Set(duration)

	registry.MustRegister(NewHostMonitor())

	metrics, err := registry.Gather()
	if err != nil {
		log.Printf("Could not gather metrics: %v", err)
		return nil, err
	}

	if true {
		prometheus := server.GetServerInstance().GetPrometheus()
		log.Println("處理 metrics 開始")
		log.Println(target, module.Prober, data["jobName"].(string))
		timeSeries, err := metric.ProcessMetrics(metrics, target, module.Prober, data["jobName"].(string), label, tags)
		if err != nil {
			log.Printf("處理 metrics 時出現錯誤: %v", err)
		}
		log.Println("處理 metrics 成功")
		log.Println("建立 remoteWriteClient 開始")

		promeClient, err := tools.NewPrometheusClient(
			prometheus.PrometheusUrl,
			prometheus.Username,
			prometheus.Password,
			"",
			"",
			prometheus.PrometheusCert,
			prometheus.EnableTLS)
		if err != nil || promeClient == nil {
			log.Printf("建立 remoteWriteClient 時出現錯誤: %v", err)
			log.Println("推送 metrics 至 Prometheus 失敗")
		} else {
			log.Println("建立 remoteWriteClient 成功")
			log.Println("推送 metrics 至 Prometheus 開始")
			err = promeRepo.SendMetrics(promeClient, timeSeries)
			if err != nil {
				log.Printf("推送 metrics 至 Prometheus 時出現錯誤: %v", err)
			}
			log.Println("推送 metrics 至 Prometheus 成功")
		}
	}
	// 在您的 doProbe 函数中，在收集 metrics 之后

	r := make(map[string]interface{})
	nested := make(map[string]interface{})

	for _, mf := range metrics {
		for i, m := range mf.Metric {
			if len(mf.Metric[i].Label) != 0 {
				name := *mf.Name
				if name == "probe_ssl_last_chain_info" {
					data[*mf.Name] = m.Gauge.Value
					continue
				}

				for _, v := range mf.Metric[i].Label {
					labelValue := *v.Value
					nested[labelValue] = m.Gauge.Value
				}

				r[*mf.Metric[i].Label[0].Name] = nested

				data[name] = r
			} else {
				data[*mf.Name] = m.Gauge.Value
			}
		}
	}

	if success {
		data["result"] = "Success"
		return data, nil
	}

	data["result"] = "Failed"

	return data, nil
}
