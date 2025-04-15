package exporter

import (
	"blackbox_agent/model/metric"
	rmclient "blackbox_agent/model/prometheusremotewrite"
	bec "blackbox_agent/pkg/blackbox_exporter/config"
	bep "blackbox_agent/pkg/blackbox_exporter/prober"
	"blackbox_agent/server"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	logger "github.com/go-kit/log"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/prompb"
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
		remoteWriteClient, err := rmclient.NewRemoteWriteClient(
			prometheus.PrometheusUrl,
			prometheus.Username,
			prometheus.Password,
			"",
			"",
			prometheus.PrometheusCert,
			prometheus.InsecureTLS)
		if err != nil {
			log.Printf("建立 remoteWriteClient 時出現錯誤: %v", err)
		} else {
			log.Println("建立 remoteWriteClient 成功")
			log.Println("推送 metrics 至 Prometheus 開始")
			err = remoteWriteClient.SendMetrics(timeSeries)
			if err != nil {
				log.Printf("推送 metrics 至 Prometheus 時出現錯誤: %v", err)
			}
			log.Println("推送 metrics 至 Prometheus 成功")
		}

		// err = remote.Prometheus_remote(metrics, target, module.Prober, data["jobName"].(string), host, label, tags)
		// if err != nil {
		// 	log.Printf("推送 metrics 至 Prometheus 时出现错误: %v", err)
		// }
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

func pushMetricsToPrometheus(metricFamilies []*io_prometheus_client.MetricFamily, target, endpoint string) error {
	l := server.GetServerInstance().GetLogger()
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Error: ", err)
	}

	type Metric struct {
		Name    string
		Help    string
		Type    string
		Metrics []struct {
			Label map[string]string
			Gauge struct {
				Value float64
			}
		}
	}

	var metrics []Metric

	for _, metricFamily := range metricFamilies {
		var metric Metric
		metric.Name = metricFamily.GetName()
		metric.Help = metricFamily.GetHelp()
		metric.Type = metricFamily.GetType().String()

		for _, m := range metricFamily.Metric {

			var metricData struct {
				Label map[string]string
				Gauge struct {
					Value float64
				}
			}
			labels := make(map[string]string)

			labels["hostname"] = hostname
			labels["target"] = target //為了避免push 時會有重複值

			if len(m.Label) > 0 {

				for _, label := range m.Label {
					labels[label.GetName()] = label.GetValue()
				}
			}

			metricData.Label = labels

			switch metricFamily.Type.String() {
			case "GAUGE":
				metricData.Gauge.Value = m.GetGauge().GetValue()
			case "COUNTER":
				metricData.Gauge.Value = m.GetCounter().GetValue()
			case "SUMMARY":
				metricData.Gauge.Value = float64(m.GetSummary().GetSampleCount())
			}

			metric.Metrics = append(metric.Metrics, metricData)
		}

		metrics = append(metrics, metric)
	}

	// // 打印結果
	// for _, metric := range metrics {
	// 	fmt.Println("Name:", metric.Name)
	// 	fmt.Println("Help:", metric.Help)
	// 	fmt.Println("Type:", metric.Type)
	// 	for _, m := range metric.Metrics {
	// 		fmt.Print("Label:")
	// 		for key, value := range m.Label {
	// 			fmt.Printf(" %s=%s", key, value)
	// 		}
	// 		fmt.Println()
	// 		fmt.Println("Value:", m.Gauge.Value)
	// 	}
	// 	fmt.Println("-----------------------")
	// }

	// 转换为 Prometheus 的 TimeSeries 格式
	timeSeries := make([]prompb.TimeSeries, 0, len(metrics))
	for _, m := range metrics {
		for _, metric := range m.Metrics {
			labels := make([]prompb.Label, 0, len(metric.Label)+2)
			labels = append(labels, prompb.Label{Name: "__name__", Value: m.Name})
			for k, v := range metric.Label {
				labels = append(labels, prompb.Label{Name: k, Value: v})
			}
			sample := prompb.Sample{
				Value:     metric.Gauge.Value,
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			}
			ts := prompb.TimeSeries{
				Labels:  labels,
				Samples: []prompb.Sample{sample},
			}
			timeSeries = append(timeSeries, ts)
		}
	}

	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}
	data, err := writeRequest.Marshal()
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}
	compressedData := snappy.Encode(nil, data)

	req, err := http.NewRequest("POST", "http://10.11.233.10:9090/api/v1/write", bytes.NewBuffer(compressedData))
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Could not push metrics to Prometheus: %v", err)
	}
	defer resp.Body.Close()

	// bodyBytes, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Fatalf("Failed to read response body: %v", err)
	// }
	// bodyString := string(bodyBytes)
	// log.Println("Response body: ", bodyString)

	if resp.StatusCode > 300 {
		log.Fatalf("Failed to push metrics to Prometheus: %d", resp.StatusCode)
	}

	l.Println("Metrics pushed to Prometheus successfully")
	return nil
}
