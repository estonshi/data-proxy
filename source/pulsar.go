package source

import (
	common "agentsc/data-proxy/common"
	"log"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

func ConsumePulsar(conf common.PulsarConf) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               conf.ServiceUrl,
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
		Authentication:    pulsar.NewAuthenticationToken(conf.Auth["jwt-token"]),
	})
	if err != nil {
		log.Fatalf("连接pulsar失败: %v", err)
	}
	// consume
	channel := make(chan pulsar.ConsumerMessage, 100)
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		// fill `TopicsPattern` field will create a regex consumer
		TopicsPattern:    conf.DataTopics,
		SubscriptionName: conf.Subscription,
		Type:             pulsar.Shared,
		MessageChannel:   channel,
	})
	if err != nil {
		log.Fatalf("订阅topic失败: %v", err)
	}
	defer consumer.Close()
	// fetch data
	for cm := range channel {
		consumer := cm.Consumer
		msg := cm.Message
		key := cm.Key()
		keyMsg := common.TransferByteMsg(key, msg.Payload(), msg.ID(), common.Config)
		if keyMsg != nil {
			common.PreProcessChan <- keyMsg
		}
		consumer.Ack(msg)
	}
}
