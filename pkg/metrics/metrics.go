package metrics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

const (
	TypeGauge   = "gauge"
	TypeCounter = "counter"

	GaugeLen   = 31
	CounterLen = 1

	Alloc         = Name("Alloc")
	TotalAlloc    = Name("TotalAlloc")
	Sys           = Name("Sys")
	Lookups       = Name("Lookups")
	Mallocs       = Name("Mallocs")
	Frees         = Name("Frees")
	HeapAlloc     = Name("HeapAlloc")
	HeapSys       = Name("HeapSys")
	HeapIdle      = Name("HeapIdle")
	HeapInuse     = Name("HeapInuse")
	HeapReleased  = Name("HeapReleased")
	HeapObjects   = Name("HeapObjects")
	StackInuse    = Name("StackInuse")
	StackSys      = Name("StackSys")
	MSpanInuse    = Name("MSpanInuse")
	MSpanSys      = Name("MSpanSys")
	MCacheInuse   = Name("MCacheInuse")
	MCacheSys     = Name("MCacheSys")
	BuckHashSys   = Name("BuckHashSys")
	GCSys         = Name("GCSys")
	OtherSys      = Name("OtherSys")
	NextGC        = Name("NextGC")
	LastGC        = Name("LastGC")
	PauseTotalNs  = Name("PauseTotalNs")
	GCCPUFraction = Name("GCCPUFraction")
	NumForcedGC   = Name("NumForcedGC")
	NumGC         = Name("NumGC")
	RandomValue   = Name("RandomValue")
	TotalMemory   = Name("TotalMemory")
	FreeMemory    = Name("FreeMemory")

	PollCount = Name("PollCount")
)

type Name string

type Gauge float64

type Counter int64

type Metrics struct {
	Gauges   map[Name]Gauge
	Counters map[Name]Counter
}

func New() *Metrics {
	return &Metrics{
		Gauges:   make(map[Name]Gauge, GaugeLen),
		Counters: make(map[Name]Counter, CounterLen),
	}
}

func (g *Gauge) FromString(str string) error {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}

	gauge := Gauge(val)
	*g = gauge

	return nil
}

func (c *Counter) FromString(str string) error {
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return err
	}

	counter := Counter(val)
	*c = counter

	return nil
}

func GaugeHash(key, id string, value float64) string {
	msg := fmt.Sprintf("%s:gauge:%f", id, value)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}

func CounterHash(key, id string, delta int64) string {
	msg := fmt.Sprintf("%s:counter:%d", id, delta)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}
