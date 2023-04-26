package source

import (
	common "agentsc/data-proxy/common"
	"context"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

func ConsumeKafka(conf common.KafkaConf) {
	urls := strings.Split(conf.Brokers, ",")
	var r *kafka.Reader
	if conf.Auth["algo"] == "scram.SHA512" {
		mechanism, err := scram.Mechanism(scram.SHA512, conf.Auth["username"], conf.Auth["password"])
		if err != nil {
			panic(err)
		} else {
			r = kafka.NewReader(kafka.ReaderConfig{
				Brokers: urls,
				Topic:   conf.DataTopic,
				GroupID: conf.ConsumerGroup,
				Dialer: &kafka.Dialer{
					SASLMechanism: mechanism,
				},
				MaxBytes: 10e6,
			})
		}
	} else {
		r = kafka.NewReader(kafka.ReaderConfig{
			Brokers:  urls,
			Topic:    conf.DataTopic,
			GroupID:  conf.ConsumerGroup,
			MaxBytes: 10e6,
		})
	}

	defer r.Close()

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Kafka Read Error : %v", err)
			time.Sleep(60 * time.Second)
			continue
		}
		msg := m.Value
		key := string(m.Key)
		keyMsg := common.TransferByteMsg(key, msg, nil, common.Config)
		if keyMsg != nil {
			common.PreProcessChan <- keyMsg
		}
	}

}
