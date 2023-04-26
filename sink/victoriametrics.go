package sink

import (
	common "agentsc/data-proxy/common"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func SinkToVM(conf common.VMConf) {
	defer func() {
		error := recover()
		if error != nil {
			log.Printf("Error while sink to vm: %v", error)
		}
	}()
	for data := range common.PostProcessChan {
		if data.Count == 0 || len(data.Metrics) == 0 {
			continue
		}
		succ, err := putVMData(process(data.Metrics), conf)
		if !succ {
			panic(err)
		}
	}
}

func process(data []common.MetricsMsgData) string {
	var series []string = []string{}
	for _, metrics := range data {
		seriesLine := metrics.Name + "{"
		for k, v := range metrics.Labels {
			seriesLine += fmt.Sprintf("%s=\"%s\",", k, v)
		}
		seriesLine += fmt.Sprintf("} %v %d", metrics.Value, metrics.Timestamp/1000)
		series = append(series, seriesLine)
	}
	if len(series) > 0 {
		return strings.Join(series, "\n")
	}
	return ""
}

func putVMData(data string, conf common.VMConf) (succ bool, err error) {
	url := conf.ServiceUrl + "/api/v1/import/prometheus"
	req, err := http.NewRequest("PUT", url, strings.NewReader(data))
	if err != nil {
		return false, err
	}
	req.Header.Add("Content", "text/plain")
	req.Header.Add("Authorization", conf.Auth["basic-token"])
	var cli *http.Client
	if conf.Timeout > 0 {
		cli = &http.Client{Timeout: time.Second * time.Duration(conf.Timeout)}
	} else {
		cli = &http.Client{}
	}
	resp, err := cli.Do(req)
	if err != nil {
		return false, err
	}
	count := strings.Count(data, "\n") + 1
	log.Printf("- sink: %d series", count)
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("error while put vm data: return code %d", resp.StatusCode)
	}
	common.SinkedSeries.Add(float64(count))
	return true, nil
}
