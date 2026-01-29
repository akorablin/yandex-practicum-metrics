package memory

import (
	"context"
	"fmt"
	"log"
	"maps"
	"time"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
)

type MemStorage struct {
	gauges      map[string]float64
	counters    map[string]int64
	cfg         *config.ServerConfig
	retryConfig RetryConfig
}

func New(cfg *config.ServerConfig) *MemStorage {
	return &MemStorage{
		gauges:      make(map[string]float64),
		counters:    make(map[string]int64),
		cfg:         cfg,
		retryConfig: DefaultRetryConfig(),
	}
}

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
	}
}

func (m *MemStorage) UpdateGauge(name string, value float64) error {
	m.gauges[name] = value
	return nil
}

func (m *MemStorage) UpdateCounter(name string, value int64) error {
	m.counters[name] += value
	return nil
}

func (m *MemStorage) UpdateMetricsBatch(metrics []models.Metrics) error {
	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			m.gauges[metric.ID] = *metric.Value
		case "counter":
			m.counters[metric.ID] = *metric.Delta
		}
	}

	return nil
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

func (m *MemStorage) Retry(ctx context.Context, operation func() error) error {
	delays := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	var lastErr error

	for attempt := 0; attempt < m.retryConfig.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		lastErr = err

		log.Printf("Попытка %d failed, retrying in %v: %v", attempt+1, delays[attempt], err)

		if attempt < len(delays) {
			select {
			case <-ctx.Done():
				return fmt.Errorf("операция отменена: %w", ctx.Err())
			case <-time.After(delays[attempt]):
			}
		}
	}

	return fmt.Errorf("все %d попыток завершились с ошибкой, последняя ошибка: %w", m.retryConfig.MaxAttempts, lastErr)
}
