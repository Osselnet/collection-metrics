package storage

import (
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
