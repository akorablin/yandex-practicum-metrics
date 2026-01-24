package memory

import (
	"maps"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	cfg      *config.ServerConfig
}

func New(cfg *config.ServerConfig) *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		cfg:      cfg,
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
		return 0, storage.ErrMetricNotFound
	}
	return value, nil
}

func (m *MemStorage) GetCounter(name string) (int64, error) {
	value, exists := m.counters[name]
	if !exists {
		return 0, storage.ErrMetricNotFound
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
