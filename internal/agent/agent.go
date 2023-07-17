package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-resty/resty/v2"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type Config struct {
	Timeout        time.Duration
	PollInterval   time.Duration
	ReportInterval time.Duration
	Address        string
}

type Agent struct {
	*metrics.Metrics
	storage storage.Repositories
	client  *resty.Client
}

type Metrics struct {
	ID    string          `json:"id"`    // имя метрики
	MType string          `json:"type"`  // параметр, принимающий значение gauge или counter
	Delta metrics.Counter `json:"delta"` // значение метрики в случае передачи counter
	Value metrics.Gauge   `json:"value"` // значение метрики в случае передачи gauge
}

var config Config

func New(cfg Config) (*Agent, error) {
	if cfg.Timeout == 0 {
		return nil, fmt.Errorf("you need to ask TimeoutTimeout")
	}
	if cfg.PollInterval == 0 {
		return nil, fmt.Errorf("you need to ask PollInterval")
	}
	if cfg.ReportInterval == 0 {
		return nil, fmt.Errorf("you need to ask ReportInterval")
	}
	if cfg.Address == "" {
		return nil, fmt.Errorf("you need to ask server address")
	}

	config = cfg

	a := &Agent{
		Metrics: metrics.New(),
		storage: storage.New(),
		client:  resty.New(),
	}
	a.client.SetTimeout(cfg.Timeout)

	return a, nil
}

func (a *Agent) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.RunPool(ctx)
	go a.RunReport(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	sig := <-c
	log.Println("Shutdown signal received:", sig)
	log.Println("Agent work completed")
}

func (a *Agent) RunPool(ctx context.Context) {
	ticker := time.NewTicker(config.PollInterval)
	for {
		select {
		case <-ticker.C:
			a.Update(ctx)
		case <-ctx.Done():
			log.Println("Regular completion of the metrics update")
			ticker.Stop()
			return
		}
	}
}

func (a *Agent) RunReport(ctx context.Context) {
	ticker := time.NewTicker(config.ReportInterval)
	for {
		select {
		case <-ticker.C:
			//	a.sendReport()
			a.sendReportUpdates(ctx)
		case <-ctx.Done():
			log.Println("Regular shutdown of sending metrics")
			ticker.Stop()
			return
		}
	}
}

func (a *Agent) sendReportUpdates(ctx context.Context) {
	hm := make([]Metrics, 0, metrics.GaugeLen+metrics.CounterLen)

	prm, err := a.storage.GetMetrics(ctx)
	if err != nil {
		a.handleError(err)
		return
	}

	for k, v := range prm.Gauges {
		value := float64(v)

		hm = append(hm, Metrics{
			ID:    string(k),
			MType: metrics.TypeGauge,
			Value: metrics.Gauge(value),
		})
	}

	for k, v := range prm.Counters {
		delta := int64(v)

		hm = append(hm, Metrics{
			ID:    string(k),
			MType: metrics.TypeCounter,
			Delta: metrics.Counter(delta),
		})
	}

	if len(hm) == 0 {
		log.Println("Empty array of metrics, nothing to send")
		return
	}

	_, err = a.sendUpdates(ctx, hm)
	if err != nil {
		a.handleError(err)
		return
	}

	log.Println("Report sent")
}

func (a *Agent) sendUpdates(ctx context.Context, hm []Metrics) (*resty.Response, error) {
	var endpoint = fmt.Sprintf("http://%s/updates/", config.Address)

	resp, err := a.client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("Content-Type", "application/json").
		SetContext(ctx).
		SetBody(hm).
		Post(endpoint)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return resp, fmt.Errorf("invalid status code %v", resp.StatusCode())
	}

	return resp, nil
}

func (a *Agent) sendReport() {
	for key, val := range a.Gauges {
		a.sendRequest(key, val)
	}
	for key, val := range a.Counters {
		a.sendRequest(key, val)
	}

	log.Println("Report sent")
}

func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %v", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %v", err)
	}
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}
	return b.Bytes(), nil
}

func (a *Agent) sendRequest(key metrics.Name, value any) int {
	var endpoint = fmt.Sprintf("http://%s/update/", config.Address)
	var met Metrics

	switch v := value.(type) {
	case metrics.Gauge:
		met = Metrics{ID: string(key), MType: "gauge", Value: v}
	case metrics.Counter:
		met = Metrics{ID: string(key), MType: "counter", Delta: v}
	default:
		a.handleError(fmt.Errorf("unknown metric type"))
		return http.StatusBadRequest
	}

	data, err := json.Marshal(met)
	if err != nil {
		a.handleError(err)
	}

	data, err = Compress(data)
	if err != nil {
		a.handleError(err)
	}

	response, err := a.client.R().
		SetBody(data).
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		Post(endpoint)

	if err != nil {
		a.handleError(err)
	}

	if response.StatusCode() != http.StatusOK {
		a.handleError(fmt.Errorf("%v", response.StatusCode()))
	}

	return response.StatusCode()
}

func (a *Agent) handleError(err error) {
	log.Println("Error -", err)
}

func (a *Agent) Update(ctx context.Context) {
	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)

	prm := metrics.New()
	gauges := make(map[metrics.Name]metrics.Gauge, metrics.GaugeLen)

	gauges[metrics.Alloc] = metrics.Gauge(ms.Alloc)
	gauges[metrics.BuckHashSys] = metrics.Gauge(ms.BuckHashSys)
	gauges[metrics.Frees] = metrics.Gauge(ms.Frees)
	gauges[metrics.GCCPUFraction] = metrics.Gauge(ms.GCCPUFraction)
	gauges[metrics.GCSys] = metrics.Gauge(ms.GCSys)
	gauges[metrics.HeapAlloc] = metrics.Gauge(ms.HeapAlloc)
	gauges[metrics.HeapIdle] = metrics.Gauge(ms.HeapIdle)
	gauges[metrics.HeapInuse] = metrics.Gauge(ms.HeapInuse)
	gauges[metrics.HeapObjects] = metrics.Gauge(ms.HeapObjects)
	gauges[metrics.HeapReleased] = metrics.Gauge(ms.HeapReleased)
	gauges[metrics.HeapSys] = metrics.Gauge(ms.HeapSys)
	gauges[metrics.LastGC] = metrics.Gauge(ms.LastGC)
	gauges[metrics.Lookups] = metrics.Gauge(ms.Lookups)
	gauges[metrics.MCacheInuse] = metrics.Gauge(ms.MCacheInuse)
	gauges[metrics.MCacheSys] = metrics.Gauge(ms.MCacheSys)
	gauges[metrics.MSpanInuse] = metrics.Gauge(ms.MSpanInuse)
	gauges[metrics.MSpanSys] = metrics.Gauge(ms.MSpanSys)
	gauges[metrics.Mallocs] = metrics.Gauge(ms.Mallocs)
	gauges[metrics.NextGC] = metrics.Gauge(ms.NextGC)
	gauges[metrics.NumForcedGC] = metrics.Gauge(ms.NumForcedGC)
	gauges[metrics.NumGC] = metrics.Gauge(ms.NumGC)
	gauges[metrics.OtherSys] = metrics.Gauge(ms.OtherSys)
	gauges[metrics.PauseTotalNs] = metrics.Gauge(ms.PauseTotalNs)
	gauges[metrics.StackInuse] = metrics.Gauge(ms.StackInuse)
	gauges[metrics.StackSys] = metrics.Gauge(ms.StackSys)
	gauges[metrics.Sys] = metrics.Gauge(ms.Sys)
	gauges[metrics.TotalAlloc] = metrics.Gauge(ms.TotalAlloc)
	gauges[metrics.RandomValue] = metrics.Gauge(rand.Float64())

	prm.Gauges = gauges

	prm.Counters[metrics.PollCount] += 1

	err := a.storage.PutMetrics(ctx, *prm)
	if err != nil {
		a.handleError(err)
	}

	log.Println("Metrics updated")
}
