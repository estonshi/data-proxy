package source

import (
	"agentsc/data-proxy/common"
	"agentsc/data-proxy/common/prompb"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
)

func PromRemoteWrite(w http.ResponseWriter, r *http.Request) {
	defer func() {
		error := recover()
		if error != nil {
			log.Printf("Error while process remote write message: %v", error)
			http.Error(w, fmt.Sprintf("%v", error), http.StatusInternalServerError)
		}
	}()

	query := r.URL.Query()
	ds := query.Get("ds")
	
	isVMRemoteWrite := r.Header.Get("Content-Encoding") == "zstd"
	var req *prompb.WriteRequest
	var err error
	if isVMRemoteWrite {
		req, err = DecodeWriteRequestVM(r.Body)
	} else {
		req, err = DecodeWriteRequest(r.Body)
	}
	
	if err != nil {
		panic(err)
	}

	samples := protoToMetrics(req, ds)
	for _,msg := range samples {
		select {
		case common.PreProcessChan <- msg:
			continue
		default:
			panic("Whooops! Processors are blocked, please wait")
		}
	}
	
}

func DecodeWriteRequest(r io.Reader) (*prompb.WriteRequest, error) {
	compressed, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		return nil, err
	}

	var req prompb.WriteRequest
	if err := req.Unmarshal(reqBuf); err != nil {
		return nil, err
	}

	return &req, nil
}

func DecodeWriteRequestVM(r io.Reader) (*prompb.WriteRequest, error) {

	compressed, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	decoder, err := getZstdDecoder()
	if err != nil {
		return nil, err
	}
	defer putZstdDecoder(decoder)

	reqBuf, err := decoder.DecodeAll(compressed, nil)
	if err != nil {
		return nil, err
	}

	var req prompb.WriteRequest
	if err := req.Unmarshal(reqBuf); err != nil {
		return nil, err
	}

	return &req, nil
}

func protoToMetrics(req *prompb.WriteRequest, ds string) map[string]*common.KeyMetricsMsg {
	metricsMsgMap := make(map[string]*common.KeyMetricsMsg)
	count := int32(len(req.Timeseries))
	var metricsMsg common.MetricsMsg = common.MetricsMsg{}
	metricsMsg.Count = int32(len(req.Timeseries))
	metricsMsg.Ds = ds
	metricsMsg.Metrics = make([]common.MetricsMsgData, metricsMsg.Count)
	for _, ts := range req.Timeseries {
		labels := make(map[string]string)
		for _, l := range ts.Labels {
			labels[string(l.Name)] = string(l.Value)
		}
		for _, s := range ts.Samples {
			metricsName := labels["__name__"]
			delete(labels, "__name__")
			instance := labels["instance"]
			if instance == "" {
				instance = "_default_"
			}
			msg := metricsMsgMap[instance]
			if msg == nil {
				msg = &common.KeyMetricsMsg{
					Key: instance,
					Metrics: make([]common.MetricsMsgData, count),
				}
				metricsMsgMap[instance] = msg
			}
			var d common.MetricsMsgData = common.MetricsMsgData{
				Labels:    labels,
				Value:     float32(s.Value),
				Timestamp: s.Timestamp,
				Name:      metricsName,
			}
			msg.Metrics = append(msg.Metrics, d)
			common.ReceivedSeriesRemoteWrite.Inc()
		}
	}
	for _, msg := range metricsMsgMap {
		msg.Count = int32(len(msg.Metrics))
		msg.Ds = ds
	}
	return metricsMsgMap
}

func getZstdDecoder() (*zstd.Decoder, error) {
	v := zstdDecoderPool.Get()
	if v == nil {
		return zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	}
	return v.(*zstd.Decoder), nil
}

func putZstdDecoder(d *zstd.Decoder) {
	d.Reset(nil)
	zstdDecoderPool.Put(d)
}

var zstdDecoderPool sync.Pool