package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"os"
)

type Repositories interface {
	Put(context.Context, string, interface{}) error
	Get(context.Context, string) (interface{}, error)
	GetMetrics(context.Context) (metrics.Metrics, error)
}

type MemStorage struct {
	*metrics.Metrics
}

func New() *MemStorage {
	return &MemStorage{
		Metrics: metrics.New(),
	}
}

func (s *MemStorage) Put(_ context.Context, key string, val interface{}) error {
	switch m := val.(type) {
	case metrics.Gauge:
		s.Gauges[metrics.Name(key)] = m
	case metrics.Counter:
		_, ok := s.Counters[metrics.Name(key)]
		if !ok {
			s.Counters[metrics.Name(key)] = m
		} else {
			s.Counters[metrics.Name(key)] += m
		}
	default:
		return fmt.Errorf("metric not implemented")
	}

	return nil
}

func (s *MemStorage) Get(_ context.Context, key string) (interface{}, error) {

	delta, ok := s.Counters[metrics.Name(key)]
	if ok {
		return delta, nil
	}

	value, ok := s.Gauges[metrics.Name(key)]
	if ok {
		return value, nil
	}

	return nil, fmt.Errorf("metric not implemented")
}

func (s *MemStorage) GetMetrics(_ context.Context) (metrics.Metrics, error) {
	gauges := s.Metrics.Gauges

	counters := s.Metrics.Counters

	return metrics.Metrics{
		Gauges:   gauges,
		Counters: counters,
	}, nil
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
