package storage

import (
	"context"
	"errors"

	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
)

var (
	ErrMetricNotFound = errors.New("metric not found")
	ErrInvalidType    = errors.New("invalid metric type")
)

type Storage interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	UpdateMetricsBatch([]models.Metrics) error
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAllMetrics() (map[string]float64, map[string]int64)
	Retry(ctx context.Context, operation func() error) error
}
