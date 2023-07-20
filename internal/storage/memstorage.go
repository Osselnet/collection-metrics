package storage

import (
	"encoding/json"
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"os"
)

type Repositories interface {
	Put(metrics.Name, metrics.Gauge)
	Count(metrics.Name, metrics.Counter)
}

type MemStorage struct {
	*metrics.Metrics
}

func (s *MemStorage) Put(key metrics.Name, val metrics.Gauge) {
	s.Gauges[key] = val
}

func (s *MemStorage) Count(key metrics.Name, val metrics.Counter) {
	_, ok := s.Counters[key]
	if !ok {
		s.Counters[key] = val
		return
	}

	s.Counters[key] += val
}

func (s *MemStorage) GetGauge(key metrics.Name) (*metrics.Gauge, error) {
	gauge, ok := s.Gauges[key]
	if !ok {
		return &gauge, fmt.Errorf("gauge metric with key '%s' not found", key)
	}

	return &gauge, nil
}

func (s *MemStorage) GetCounter(key metrics.Name) (*metrics.Counter, error) {
	counter, ok := s.Counters[key]
	if !ok {
		return &counter, fmt.Errorf("counter metric with key '%s' not found", key)
	}

	return &counter, nil
}

func (s *MemStorage) WriteDataToFile(filename string) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	err = f.Truncate(0)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}
