package prometheus_remote

import (
	"blackbox_agent/server"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/snappy"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/prompb"
)

func Prometheus_remote(metricFamilies []*io_prometheus_client.MetricFamily, target, prober, jobName, endpoint string, label, tags map[string]interface{}) error {
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
			labels["probe"] = prober
			labels["jobName"] = jobName
			for k, v := range label {
				labels[k] = v.(string)
			}
			for k, v := range tags {
				labels[k] = v.(string)
			}

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

	// 匯出檔案到文件
	file, err := os.Create("output.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)

	}
	defer file.Close() // 确保在函数结束时关闭文件

	for _, metric := range metrics {
		fmt.Fprintf(file, "Name: %s\n", metric.Name)
		fmt.Fprintf(file, "Help: %s\n", metric.Help)
		fmt.Fprintf(file, "Type: %s\n", metric.Type)
		for _, m := range metric.Metrics {
			fmt.Fprintf(file, "Label:")
			for key, value := range m.Label {
				fmt.Fprintf(file, " %s=%s", key, value)
			}
			fmt.Fprintln(file)
			fmt.Fprintf(file, "Value: %v\n", m.Gauge.Value)
		}
		fmt.Fprintln(file, "-----------------------")
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

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(compressedData))
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode > 300 {
		log.Printf("Failed %s reason: %s", target, bodyString)
		return nil
	}

	l.Println("Metrics pushed to Prometheus successfully")
	return nil
}
