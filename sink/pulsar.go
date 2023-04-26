package sink

import (
	common "agentsc/data-proxy/common"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

func SinkToPulsar(conf common.PulsarConf) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               conf.ServiceUrl,
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
		Authentication:    pulsar.NewAuthenticationToken(conf.Auth["jwt-token"]),
	})
	if err != nil {
		log.Fatalf("Error: 连接pulsar失败: %v", err)
	}
	defer client.Close()
	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: conf.DataTopics,
	})
	if err != nil {
		log.Fatalf("Error: 创建pulsar生产者失败: %v", err)
	}
	// produce
	for data := range common.PostProcessChan {
		msg := common.MetricsMsg{
			Ds:      data.Ds,
			Count:   int32(len(data.Metrics)),
			Metrics: data.Metrics,
		}
		d, err := json.Marshal(msg)
		if err != nil {
			if common.DebugMod {
				log.Printf("Error: %v", err)
			}
			continue
		}
		if data.Key != "" {
			_, err2 := producer.Send(context.Background(), &pulsar.ProducerMessage{
				Payload: d,
			})
			if err2 != nil {
				continue
			}
		} else {
			_, err2 := producer.Send(context.Background(), &pulsar.ProducerMessage{
				Key:     data.Key,
				Payload: d,
			})
			if err2 != nil {
				continue
			}
		}
		common.SinkedSeries.Add(float64(len(data.Metrics)))
	}

}
