package handler

import (
	"blackbox_agent/server"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	bCheck "blackbox_agent/handler/yaml_check/module"
	tCheck "blackbox_agent/handler/yaml_check/target"

	"github.com/b85bagent/rabbitmq"
	"github.com/streadway/amqp"
)

const (
	blackbox_file_location       = "./blackbox_exporter/blackbox.yaml"
	target_file_location         = "./yaml/target.yaml"
	blackbox_error_file_location = "./blackbox_exporter/blackbox_error.yaml"
	target_error_file_location   = "./yaml/target_error.yaml"
)

var wg sync.WaitGroup

// 開啟rabbitMQ監聽
func ListenRabbitMQ(reload chan bool) error {
	rabbitMQ := server.GetServerInstance().GetRabbitMQArg()
	if len(rabbitMQ.Host) == 0 || strings.TrimSpace(rabbitMQ.Host[0]) == "" {
		log.Println("RabbitMQ host is empty, skip ListenRabbitMQ")
		return nil
	}
	if len(rabbitMQ.RabbitMQQueue) == 0 {
		log.Println("RabbitMQ queue is empty, skip ListenRabbitMQ")
		return nil
	}

	for _, v := range rabbitMQ.RabbitMQQueue {
		if strings.TrimSpace(v) == "" {
			continue
		}
		rabbitMQArg := getRabbitMQArg(rabbitMQ, v)
		fileLocation := determineFileLocation(v)
		localResponse := initRPCResponse(rabbitMQArg)

		wg.Add(1)
		go handleRabbitMQMessage(rabbitMQArg, fileLocation, localResponse, reload)
	}

	wg.Wait()
	return nil
}

func getRabbitMQArg(rabbitMQ server.RabbitMQArg, queueName string) rabbitmq.RabbitMQArg {
	host := ""
	if len(rabbitMQ.Host) > 0 {
		host = rabbitMQ.Host[0]
	}
	return rabbitmq.RabbitMQArg{
		Host:               host,
		Username:           rabbitMQ.Username,
		Password:           rabbitMQ.Password,
		RabbitMQExchange:   rabbitMQ.RabbitMQExchange,
		RabbitMQRoutingKey: rabbitMQ.RabbitMQRoutingKey,
		RabbitMQQueue:      queueName,
	}
}

func determineFileLocation(queueName string) string {
	if strings.Contains(queueName, "modules") {
		return blackbox_file_location
	}
	return target_file_location
}

func initRPCResponse(arg rabbitmq.RabbitMQArg) rabbitmq.RPCResponse {
	t := make(map[string]interface{})
	t["message"] = "Agent get MQ message Successfully"
	return rabbitmq.RPCResponse{
		Status:     rabbitmq.Response_Success,
		StatusCode: rabbitmq.Response_Success_Code,
		Response:   t,
		Queue:      arg.RabbitMQQueue,
	}
}

func handleRabbitMQMessage(arg rabbitmq.RabbitMQArg, fileLocation string, response rabbitmq.RPCResponse, reload chan bool) {
	defer wg.Done()

	localResponse := response // 使用本地副本，避免數據競爭

	err := rabbitmq.ListenRabbitMQUsingRPC(arg, localResponse, func(msg amqp.Delivery, ch *amqp.Channel, localResponse rabbitmq.RPCResponse) error {

		check := true
		shouldSave := true

		localResponse.Timestamp = time.Now()
		localResponse.CorrelationId = msg.CorrelationId
		localResponse.Queue = arg.RabbitMQQueue

		t := make(map[string]interface{})

		if strings.Contains(fileLocation, "blackbox") {

			blackboxConfigCheck, errBlackboxCheck := bCheck.CheckModuleFormat(msg.Body)
			if !blackboxConfigCheck {
				log.Println("The Modules YAML format is not valid!")
				t["message"] = "Invalid Module Format, Please check the _example configuration"
				t["reason"] = errBlackboxCheck.Error()
				localResponse.Status = rabbitmq.Response_Failed
				localResponse.StatusCode = rabbitmq.Response_Failed_Code
				localResponse.Response = t
				check = false
				// 將收到的message 寫入error檔案
				err := SaveYAMLToFile(msg.Body, blackbox_error_file_location)
				if err != nil {
					log.Printf("Error writing to "+blackbox_error_file_location+": %v", err)
				}

			} else {
				if errBlackboxCheck != nil {
					localResponse.Response["warn"] = errBlackboxCheck.Error()
					shouldSave = false
				}

			}

		} else {

			targetConfigCheck, errTargetCheck := tCheck.CheckTargetFormatAndReplace(msg.Body)
			if !targetConfigCheck {
				log.Println("The Target YAML format is not valid!")
				t["message"] = "Invalid Target Format, Please check the _example configuration"
				t["reason"] = errTargetCheck.Error()
				localResponse.Status = rabbitmq.Response_Failed
				localResponse.StatusCode = rabbitmq.Response_Failed_Code
				localResponse.Response = t

				check = false
				// 將收到的message 寫入error檔案
				err := SaveYAMLToFile(msg.Body, target_error_file_location)
				if err != nil {
					log.Printf("Error writing to "+target_error_file_location+": %v", err)
				}

			} else {

				if errTargetCheck != nil {
					localResponse.Response["warn"] = errTargetCheck.Error()
					shouldSave = false

				}
			}

		}

		//double check for blackbox config
		if strings.Contains(fileLocation, "blackbox") && check && shouldSave {

			if errFinalCheck := handleReloadConfig(msg.Body); errFinalCheck != nil {
				log.Println("The Modules YAML Final Check is valid")
				t["message"] = "The Modules YAML Final Check is valid, Please check the _example configuration and reason"
				t["reason"] = errFinalCheck.Error()
				localResponse.Status = rabbitmq.Response_Failed
				localResponse.StatusCode = rabbitmq.Response_Failed_Code
				localResponse.Response = t
				check = false

				// 將收到的message 寫入error檔案
				err := SaveYAMLToFile(msg.Body, blackbox_error_file_location)
				if err != nil {
					log.Printf("Error writing to "+blackbox_error_file_location+": %v", err)

				}
			}

		}

		if check && shouldSave {
			// 將收到的message 取代本地檔案
			err := SaveYAMLToFile(msg.Body, fileLocation)
			if err != nil {
				log.Printf("Error writing to %s: %v", fileLocation, err)
				t["message"] = "Error writing to " + fileLocation + ": " + err.Error()
				localResponse.Status = rabbitmq.Response_Failed
				localResponse.StatusCode = rabbitmq.Response_Failed_Code
				localResponse.Response = t

				check = false
			}

		}

		if check {
			// 觸發 reload 流程
			reload <- true
		}
		err := replyToPublisher(localResponse, ch, msg)
		if err != nil {
			log.Printf("Error replyToPublisher : %s", err)
			return err
		}

		return nil
	})

	if err != nil {
		log.Println("ListenRabbitMQUsingRPC ERROR: ", err)
	}
}

// 將回應發送回去
func replyToPublisher(localResponse rabbitmq.RPCResponse, ch *amqp.Channel, msg amqp.Delivery) error {

	response, err := json.Marshal(localResponse)
	if err != nil {
		log.Printf("Error marshaling to JSON: %v\n", err)
		return err
	}

	// 發送回應到 reply_to 隊列
	errReplay := ch.Publish(
		"",          // exchange
		msg.ReplyTo, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: msg.CorrelationId,
			Body:          response,
		})
	if errReplay != nil {
		log.Printf("Failed to publish a message Replay: %s", errReplay)
		return errReplay
	}

	// 發送 ack 確認消息已經被處理
	err = msg.Ack(false)
	if err != nil {
		log.Printf("Error acknowledging message : %s", err)
		return err
	}

	return nil
}

func SaveYAMLToFile(content []byte, filepath string) error {
	l := server.GetServerInstance().GetLogger()

	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		log.Println("無法寫入檔案：", err)
		return err
	}

	l.Println("已成功儲存 YAML 檔案：", filepath)
	return nil
}

type ScrapeConfig struct {
	JobName        string         `yaml:"job_name"`
	ScrapeInterval string         `yaml:"scrape_interval"`
	MetricsPath    string         `yaml:"metrics_path"`
	Params         Params         `yaml:"params"`
	StaticConfigs  []StaticConfig `yaml:"static_configs"`
}

type Params struct {
	Module []string `yaml:"module"`
}

type StaticConfig struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels,omitempty"`
	Tag     string            `yaml:"tag,omitempty"`
}

type Configuration struct {
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs"`
}

func handleReloadConfig(content []byte) error {
	// 將內容寫入一個臨時檔案
	tmpfile, err := os.CreateTemp("", "blackbox_doubleCheck*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name()) // 在返回前刪除臨時檔案

	if _, err := tmpfile.Write(content); err != nil {
		tmpfile.Close()
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}

	validationLoader := blackboxBackend.NewLoader()
	if err := validationLoader.Reload(tmpfile.Name()); err != nil {
		return err
	}

	return nil
}
