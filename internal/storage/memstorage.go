package storage

import (
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
)

type Repositories interface {
	Put(metrics.Name, metrics.Gauge)
	Count(metrics.Name, metrics.Counter)
}

type MemStorage struct {
	*metrics.Metrics
}

func New() *MemStorage {
	return &MemStorage{
		Metrics: metrics.New(),
	}
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
		return nil, fmt.Errorf("gauge metric with key '%s' not found", key)
	}

	return &gauge, nil
}

func (s *MemStorage) GetCounter(key metrics.Name) (*metrics.Counter, error) {
	counter, ok := s.Counters[key]
	if !ok {
		return nil, fmt.Errorf("counter metric with key '%s' not found", key)
	}

	return &counter, nil
}
