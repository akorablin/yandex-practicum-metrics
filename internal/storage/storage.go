package storage

import (
	"errors"
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
	LoadFromFile() error
	SaveToFile() error
}
