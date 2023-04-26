package process

import (
	"agentsc/data-proxy/common"
	"log"
	"math"
	"regexp"
)

type LabelProcess struct {}

func (p LabelProcess) StartProcess() {
	for msg := range common.PreProcessChan {
		pmsg := processStructKeyMsg(msg, common.Config)
		if (pmsg != nil) {
			common.PostProcessChan <- pmsg
		}
	}
}

func processStructKeyMsg(metricsMsg *common.KeyMetricsMsg, conf common.Conf) *common.KeyMetricsMsg {
	common.ReceivedMsg.Inc()
	var series common.KeyMetricsMsg = common.KeyMetricsMsg{
		Key:     metricsMsg.Key,
		Ds:      metricsMsg.Ds,
		Metrics: []common.MetricsMsgData{},
	}
	for _, metrics := range metricsMsg.Metrics {
		common.RecievedSeries.Inc()
		if metrics.Name == "" || metrics.Timestamp < 1665285120000 || metrics.Labels == nil ||
			len(metrics.Labels) == 0 || math.IsNaN(float64(metrics.Value)) {
			common.DroppedSeries.Inc()
			continue
		}
		if metricsMsg.Ds != "" {
			metrics.Labels["_ds"] = metricsMsg.Ds
		}
		if metrics.Labels["_device"] == "" && metrics.Labels["_host"] == "" &&
			metrics.Labels["_application"] == "" && metrics.Labels["_service"] == "" &&
			metrics.Labels["_process"] == "" {
			common.DroppedSeries.Inc()
			continue
		}
		reg2, _ := regexp.Compile(`^[\w.-]+(:\d+)*$`)
		if metrics.Labels["_agent"] != "" && !reg2.MatchString(metrics.Labels["_agent"]) {
			common.DroppedSeries.Inc()
			continue
		}
		series.Metrics = append(series.Metrics, metrics)
		common.ProccessedSeries.Inc()
	}
	if len(series.Metrics) > 0 {
		if common.DebugMod {
			log.Printf("- processed: %d series", len(series.Metrics))
		}
		series.Count = int32(len(series.Metrics))
		return &series
	} else {
		return nil
	}
}



