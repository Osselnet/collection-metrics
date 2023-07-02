package agent

import (
	"encoding/json"
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-resty/resty/v2"
	"log"
	"math/rand"
	"net/http"
	"runtime"
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
	client *resty.Client
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
		client:  resty.New(),
	}
	a.client.SetTimeout(cfg.Timeout)

	return a, nil
}

func (a *Agent) Run() {
	go a.RunPool()
	a.RunReport()
}

func (a *Agent) RunPool() {
	for {
		time.Sleep(config.PollInterval)
		a.Update()
	}
}

func (a *Agent) RunReport() {
	for {
		time.Sleep(config.ReportInterval)
		a.sendReport()
	}
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

	response, err := a.client.R().
		SetBody(data).
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

func (a *Agent) Update() {
	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)

	a.Gauges[metrics.Alloc] = metrics.Gauge(ms.Alloc)
	a.Gauges[metrics.BuckHashSys] = metrics.Gauge(ms.BuckHashSys)
	a.Gauges[metrics.Frees] = metrics.Gauge(ms.Frees)
	a.Gauges[metrics.GCCPUFraction] = metrics.Gauge(ms.GCCPUFraction)
	a.Gauges[metrics.GCSys] = metrics.Gauge(ms.GCSys)
	a.Gauges[metrics.HeapAlloc] = metrics.Gauge(ms.HeapAlloc)
	a.Gauges[metrics.HeapIdle] = metrics.Gauge(ms.HeapIdle)
	a.Gauges[metrics.HeapInuse] = metrics.Gauge(ms.HeapInuse)
	a.Gauges[metrics.HeapObjects] = metrics.Gauge(ms.HeapObjects)
	a.Gauges[metrics.HeapReleased] = metrics.Gauge(ms.HeapReleased)
	a.Gauges[metrics.HeapSys] = metrics.Gauge(ms.HeapSys)
	a.Gauges[metrics.LastGC] = metrics.Gauge(ms.LastGC)
	a.Gauges[metrics.Lookups] = metrics.Gauge(ms.Lookups)
	a.Gauges[metrics.MCacheInuse] = metrics.Gauge(ms.MCacheInuse)
	a.Gauges[metrics.MCacheSys] = metrics.Gauge(ms.MCacheSys)
	a.Gauges[metrics.MSpanInuse] = metrics.Gauge(ms.MSpanInuse)
	a.Gauges[metrics.MSpanSys] = metrics.Gauge(ms.MSpanSys)
	a.Gauges[metrics.Mallocs] = metrics.Gauge(ms.Mallocs)
	a.Gauges[metrics.NextGC] = metrics.Gauge(ms.NextGC)
	a.Gauges[metrics.NumForcedGC] = metrics.Gauge(ms.NumForcedGC)
	a.Gauges[metrics.NumGC] = metrics.Gauge(ms.NumGC)
	a.Gauges[metrics.OtherSys] = metrics.Gauge(ms.OtherSys)
	a.Gauges[metrics.PauseTotalNs] = metrics.Gauge(ms.PauseTotalNs)
	a.Gauges[metrics.StackInuse] = metrics.Gauge(ms.StackInuse)
	a.Gauges[metrics.StackSys] = metrics.Gauge(ms.StackSys)
	a.Gauges[metrics.Sys] = metrics.Gauge(ms.Sys)
	a.Gauges[metrics.TotalAlloc] = metrics.Gauge(ms.TotalAlloc)
	a.Gauges[metrics.RandomValue] = metrics.Gauge(rand.Float64())

	a.Counters[metrics.PollCount] += 1

	log.Println("Metrics updated")
}
