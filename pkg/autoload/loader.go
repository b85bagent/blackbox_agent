package autoload

import (
	"blackbox_agent/config"
	"blackbox_agent/handler"
	"time"

	"blackbox_agent/pkg/tool"
	"blackbox_agent/server"
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"sync"

	"github.com/opensearch-project/opensearch-go"
)

var (
	reload      chan bool
	reloadMutex sync.Mutex
)

const (
	default_targetFile   = "target.yaml"
	default_blackboxFile = "blackbox.yaml"
)

func AutoLoader(configFile, targetFile, blackboxFile string) {

	config, err := config.ConfigInit(configFile)

	if err != nil {
		log.Println(err)
		panic("config init fail")
	}

	serverInstance, err := server.NewServer()
	if err != nil {
		log.Println(err)
		panic("autoload fail")
	}

	serverInstance.Constant = config.Const

	serverInstance.Prometheus = &config.Prometheus

	handlerServer := &handler.Server{
		ServerStruct: serverInstance,
	}

	debugMode := getDebugSetting()

	logger := tool.NewLogger(debugMode)

	handlerServer.ServerStruct.SetLogger(logger)

	if len(config.Opensearch.Opensearch) > 0 {
		logger.Println("Auto loading opensearch")

		handlerServer.ServerStruct.OpensearchIndex = config.Opensearch.Index
		handlerServer.ServerStruct.OpensearchEnable = config.Opensearch.Enable

		opensearch, err := initOpensearch(config.Opensearch.Opensearch)
		if err != nil {
			log.Println(err)
			panic("initOpensearch fail")
		}

		handlerServer.ServerStruct.SetOpensearch(opensearch)
	}

	if len(config.RabbitMQ.RabbitMQ) > 0 {

		logger.Println("Auto loading RabbitMQ")

		for _, v := range config.RabbitMQ.RabbitMQ {
			handlerServer.ServerStruct.RabbitMQConfig.Host = v.Host
			handlerServer.ServerStruct.RabbitMQConfig.Username = v.Username
			handlerServer.ServerStruct.RabbitMQConfig.Password = v.Password
			handlerServer.ServerStruct.RabbitMQConfig.RabbitMQExchange = v.RabbitMQExchange
			handlerServer.ServerStruct.RabbitMQConfig.RabbitMQRoutingKey = v.RabbitMQRoutingKey
			handlerServer.ServerStruct.RabbitMQConfig.RabbitMQQueue = v.RabbitMQQueue
		}

	}

	// if len(config.Prometheus.Prometheus) > 0 {
	// 	for _, v := range config.Prometheus.Prometheus {
	// 		if v.Enable {

	// 			handlerServer.ServerStruct.PrometheusHost = v.Host
	// 			handlerServer.ServerStruct.PrometheusEnable = v.Enable
	// 			logger.Println("Auto loading Prometheus")
	// 		}
	// 	}
	// }

	logger.Println("AutoLoader Success")
	reload = make(chan bool, 1)

	// Main context
	ctx := tool.WaitShutdown(func() {})
	handlerServer.ServerStruct.SetGracefulCtx(&ctx)

	blackboxCtx, blackboxCancel := context.WithCancel(ctx)

	go handler.ListenRabbitMQ(reload)

	go handler.BlackboxProcess(blackboxCtx, targetFile, blackboxFile, true)

	go func() {
		for {
			select {
			case <-reload:
				reloadMutex.Lock()
				if blackboxCancel != nil {
					blackboxCancel() // 取消之前的协程
				}

				log.Println("啟動新的handler.BlackboxProcess")

				blackboxCtx, blackboxCancel = context.WithCancel(ctx)

				handler.BlackboxProcess(blackboxCtx, default_targetFile, default_blackboxFile, false)

				reloadMutex.Unlock()

			}

		}
	}()

	select {
	case s := <-ctx.Done():
		logger.Printf("shutdownObserver:", s)
	}

	var countdownTime = 5
	for t := countdownTime; t > 0; t-- {
		log.Printf("%d秒後退出", t)
		time.Sleep(time.Second * 1)
	}

}

func initOpensearch(setting map[string]config.OpensearchConfig) (map[string]*opensearch.Client, error) {

	opensearchClient := make(map[string]*opensearch.Client)

	for key, v := range setting {

		client, err := opensearch.NewClient(opensearch.Config{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Addresses: v.Host,
			Username:  v.Username,
			Password:  v.Password,
		})

		if err != nil {
			log.Println("無法建立 OpenSearch 客戶端:", err)
			return nil, err
		}
		// log.Println(client.Info())
		opensearchClient[key] = client
	}

	// Print OpenSearch version information on console.

	return opensearchClient, nil
}

func getDebugSetting() bool {

	debugSetting, ok := server.GetServerInstance().GetConst()["debug"]
	if !ok {
		log.Println("DebugSetting Get failed")
		return false
	}

	return debugSetting.(bool)
}
