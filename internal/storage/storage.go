package storage

import (
	"errors"
	"maps"
)

var (
	ErrMetricNotFound = errors.New("metric not found")
	ErrInvalidType    = errors.New("invalid metric type")
)

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAllMetrics() (map[string]float64, map[string]int64)
}

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.counters[name] += value
}

func (m *MemStorage) GetGauge(name string) (float64, error) {
	value, exists := m.gauges[name]
	if !exists {
		return 0, ErrMetricNotFound
	}
	return value, nil
}

func (m *MemStorage) GetCounter(name string) (int64, error) {
	value, exists := m.counters[name]
	if !exists {
		return 0, ErrMetricNotFound
	}
	return value, nil
}

func (m *MemStorage) GetAllMetrics() (map[string]float64, map[string]int64) {
	gaugesCopy := make(map[string]float64)
	countersCopy := make(map[string]int64)

	maps.Copy(gaugesCopy, m.gauges)
	maps.Copy(countersCopy, m.counters)

	return gaugesCopy, countersCopy
}
