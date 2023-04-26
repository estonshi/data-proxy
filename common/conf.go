package common

import (
	"encoding/json"
	"log"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	yaml "gopkg.in/yaml.v3"
)

type Source struct {
	Id   string `yaml:"id"`
	Type string `yaml:"type"`
}

type Process struct {
	Id   string `yaml:"id"`
}

type Sink struct {
	Id   string `yaml:"id"`
	Type string `yaml:"type"`
}

type PulsarConf struct {
	ServiceUrl   string            `yaml:"serviceUrl"`
	Auth         map[string]string `yaml:"auth"`
	DataTopics   string            `yaml:"data_topics"`
	CmdTopics    string            `yaml:"cmd_topics"`
	Subscription string            `yaml:"subscription"`
}

type KafkaConf struct {
	Brokers       string            `yaml:"brokers"`
	Auth          map[string]string `yaml:"auth"`
	DataTopic     string            `yaml:"data_topic"`
	Retries       int               `yaml:"retries"`
	ConsumerGroup string            `taml:"consumer_group"`
}

type VMConf struct {
	ServiceUrl string            `yaml:"serviceUrl"`
	Auth       map[string]string `yaml:"auth"`
	Labels     map[string]string `yaml:"labels"`
	Timeout    int               `yaml:"timeout"`
}

type ItmConf struct {
	ServiceUrl string            `yaml:"serviceUrl"`
	Auth       map[string]string `yaml:"auth"`
}

type RemoteApi struct {
	Enabled bool                `yaml:"enabled"`
}

type Conf struct {
	Port       int                   `yaml:"port"`
	ChanBuff   int                   `yaml:"chanbuffer"`
	Source     Source                `yaml:"source"`
	Process    Process               `yaml:"process"`
	Sink       Sink                  `yaml:"sink"`
	PulsarConf map[string]PulsarConf `yaml:"pulsar"`
	KafkaConf  map[string]KafkaConf  `yaml:"kafka"`
	VMConf     map[string]VMConf     `yaml:"vm"`
	RemoteApi  map[string]RemoteApi  `yaml:"remoteApi"`
}

func GetConfig(file string) Conf {
	info, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("读取配置文件出错: %v", err)
	}
	var data Conf
	err2 := yaml.Unmarshal(info, &data)
	if err2 != nil {
		log.Fatalf("读取配置文件出错: %f", err2)
	}
	return data
}

type MetricsMsgData struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Value     float32           `json:"value"`
	Timestamp int64             `json:"timestamp"`
}

type MetricsMsg struct {
	Ds      string           `json:"_ds"`
	Count   int32            `json:"count"`
	Metrics []MetricsMsgData `json:"metrics"`
}

type KeyMetricsMsg struct {
	Ds      string
	Count   int32
	Key     string
	Metrics []MetricsMsgData
}

var PreProcessChan chan *KeyMetricsMsg
var PostProcessChan chan *KeyMetricsMsg
var Config Conf
var DebugMod bool

var (
	RecievedSeries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "itm_data_proxy_received_series_counter",
		Help: "Total count of receivied time series",
	})
	ProccessedSeries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "itm_data_proxy_processed_series_counter",
		Help: "Total count of processed time series",
	})
	DroppedSeries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "itm_data_proxy_dropped_series_counter",
		Help: "Total count of dropped time series",
	})
	InvalidMsg = promauto.NewCounter(prometheus.CounterOpts{
		Name: "itm_data_proxy_invalid_msg_counter",
		Help: "Total count of MQ messages that are failed to receive",
	})
	ReceivedMsg = promauto.NewCounter(prometheus.CounterOpts{
		Name: "itm_data_proxy_received_msg_counter",
		Help: "Total count of MQ messages that are successfully received",
	})
	ReceivedSeriesRemoteWrite = promauto.NewCounter(prometheus.CounterOpts{
		Name: "itm_data_proxy_received_series_remotewrite_counter",
		Help: "Total number of received time series from remote-write api.",
	})
	SinkedSeries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "itm_data_proxy_sinked_series_counter",
		Help: "Total number of sinked series",
	})
)

func TransferByteMsg(key string, msg []byte, msgId interface{}, conf Conf) *KeyMetricsMsg {
	var metricsMsg MetricsMsg
	err := json.Unmarshal(msg, &metricsMsg)
	if err != nil {
		InvalidMsg.Inc()
		log.Printf("Error while process message: %v, msg: %v", err, msg)
		return nil
	}
	var keyMsg KeyMetricsMsg = KeyMetricsMsg{
		Key:     key,
		Ds:      metricsMsg.Ds,
		Count:   metricsMsg.Count,
		Metrics: metricsMsg.Metrics,
	}
	return &keyMsg
}

func SetDefaultConfigs(conf *Conf) {
	if conf.ChanBuff == 0 {
		Config.ChanBuff = 1000
	}
	if conf.Port == 0 {
		Config.Port = 9423
	}
	if conf.KafkaConf != nil {
		for k,c := range conf.KafkaConf {
			if c.ConsumerGroup == "" {
				c.ConsumerGroup = "data-proxy"
			}
			if c.Retries == 0 {
				c.Retries = 1
			}
			conf.KafkaConf[k] = c
		}
	}
	if conf.PulsarConf != nil {
		for k,c := range conf.PulsarConf {
			if c.Subscription == "" {
				c.Subscription = "data-proxy"
			}
			conf.PulsarConf[k] = c
		}
	}
	if conf.VMConf != nil {
		for k,c := range conf.VMConf {
			if c.Timeout == 0 {
				c.Timeout = 3
			}
			conf.VMConf[k] = c
		}
	}
	if conf.RemoteApi == nil {
		conf.RemoteApi = make(map[string]RemoteApi)
	}
}