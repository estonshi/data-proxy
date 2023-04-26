package main

import (
	common "agentsc/data-proxy/common"
	"agentsc/data-proxy/process"
	sink "agentsc/data-proxy/sink"
	source "agentsc/data-proxy/source"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	// read flag
	var confile string
	var debugMode bool
	flag.StringVar(&confile, "c", "", "配置文件")
	flag.BoolVar(&debugMode, "debug", false, "开启debug日志")
	flag.Parse()
	if confile == "" {
		log.Fatal("请提供配置文件")
	}

	// read conf
	config := common.GetConfig(confile)
	common.Config = config
	common.DebugMod = debugMode
	common.SetDefaultConfigs(&common.Config)

	// open channel
	var chanBuff int = 100
	if config.ChanBuff > chanBuff {
		chanBuff = config.ChanBuff
	}
	common.PostProcessChan = make(chan *common.KeyMetricsMsg, chanBuff)
	defer close(common.PostProcessChan)
	common.PreProcessChan = make(chan *common.KeyMetricsMsg, chanBuff)
	defer close(common.PreProcessChan)

	// start sink goroutine
	switch config.Sink.Type {
	case "vm":
		vmConf := config.VMConf[config.Sink.Id]
		log.Printf("Sink: victoriametrics, %v", vmConf.ServiceUrl)
		go sink.SinkToVM(vmConf)
	case "pulsar":
		pulsarConf := config.PulsarConf[config.Sink.Id]
		log.Printf("Sink: pulsar, %v", pulsarConf.ServiceUrl)
		go sink.SinkToPulsar(pulsarConf)
	case "kafka":
		kafkaConf := config.KafkaConf[config.Sink.Id]
		log.Printf("Sink: kafka, %v", kafkaConf.Brokers)
		go sink.SinkToKafka(kafkaConf)
	default:
		log.Fatalf("Error: invalid sink '%s'", config.Sink.Id)
	}

	// start process goroutine
	var processor common.Processor
	switch config.Process.Id {
	case "", "empty":
		log.Printf("Process: empty")
		processor = process.EmptyProcess{}
	case "label":
		log.Printf("Process: label")
		processor = process.LabelProcess{}
	}
	go processor.StartProcess()
	
	// start source goroutine
	switch config.Source.Type {
	case "pulsar":
		pulsarConf := config.PulsarConf[config.Source.Id]
		log.Printf("Source: pulsar, %v", pulsarConf.ServiceUrl)
		go source.ConsumePulsar(pulsarConf)
	case "kafka":
		kafkaConf := config.KafkaConf[config.Source.Id]
		log.Printf("Source: kafka, %v", kafkaConf.Brokers)
		go source.ConsumeKafka(kafkaConf)
	case "":
		log.Print("Warn: No source provided.")
	default:
		log.Fatalf("Error: invalid source '%s'", config.Source.Id)
	}

	// start remote api
	for key,api := range config.RemoteApi {
		switch key {
		case "prometheus":
			if api.Enabled {
				http.HandleFunc("/write", source.PromRemoteWrite)
				http.HandleFunc("/write/prometheus", source.PromRemoteWrite)
				log.Printf("RemoteApi: prometheus remote-wirte api opened")
			}
		}
	}

	// start expose prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}
