package sink

import (
	common "agentsc/data-proxy/common"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

func SinkToKafka(conf common.KafkaConf) {
	urls := strings.Split(conf.Brokers, ",")
	var w *kafka.Writer
	if conf.Auth["algo"] == "scram.SHA512" {
		mechanism, err := scram.Mechanism(scram.SHA512, conf.Auth["username"], conf.Auth["password"])
		if err != nil {
			panic(err)
		} else {
			w = kafka.NewWriter(kafka.WriterConfig{
				Brokers: urls,
				Topic:   conf.DataTopic,
				Dialer: &kafka.Dialer{
					SASLMechanism: mechanism,
				},
			})
		}
	} else {
		w = kafka.NewWriter(kafka.WriterConfig{
			Brokers: urls,
			Topic:   conf.DataTopic,
		})
	}

	defer w.Close()

	message := []kafka.Message{}
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
			message = append(message, kafka.Message{
				Key:   []byte(data.Key),
				Value: d,
			})
		} else {
			message = append(message, kafka.Message{
				Value: d,
			})
		}
		for i := 0; i < conf.Retries; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := w.WriteMessages(ctx, message...)
			if errors.Is(err, kafka.BrokerNotAvailable) || errors.Is(err, context.DeadlineExceeded) {
				time.Sleep(time.Millisecond * 250)
				cancel()
				continue
			} else if err == nil {
				common.SinkedSeries.Add(float64(len(data.Metrics)))
				cancel()
				break
			} else {
				cancel()
				break
			}
		}
	}

}
