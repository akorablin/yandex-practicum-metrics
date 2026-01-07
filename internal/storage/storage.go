package storage

import (
	"encoding/json"
	"errors"
	"log"
	"maps"
	"net/http"
	"os"
	"strings"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
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
	SaveToFile() error
}

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	cfg      *config.ServerConfig
}

func NewMemStorage(cfg *config.ServerConfig) *MemStorage {
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

func (m *MemStorage) LoadFromFile() error {
	if m.cfg.FileStoragePath == "" || !m.cfg.Restore {
		return nil
	}

	if _, err := os.Stat(m.cfg.FileStoragePath); err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	path := m.cfg.FileStoragePath
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var loadedMetrics []models.Metrics
	if err := json.Unmarshal(data, &loadedMetrics); err != nil {
		return err
	}

	for _, metric := range loadedMetrics {
		mType, ID, value, delta := metric.MType, metric.ID, metric.Value, metric.Delta
		if mType == "gauge" {
			m.UpdateGauge(ID, *value)
		}
		if mType == "counter" {
			m.UpdateCounter(ID, *delta)
		}
	}

	return nil
}

func (m *MemStorage) SaveToFile() error {
	path := m.cfg.FileStoragePath
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	var gauges, counters = m.GetAllMetrics()
	all := make([]models.Metrics, 0, len(gauges)+len(counters))
	for k, v := range gauges {
		item := models.Metrics{ID: k, MType: "gauge", Value: &v}
		all = append(all, item)
	}
	for k, v := range counters {
		item := models.Metrics{ID: k, MType: "counter", Delta: &v}
		all = append(all, item)
	}

	bytes, err := json.Marshal(all)
	if err != nil {
		log.Printf("Marshal error")
		return err
	}

	WriteFileError := os.WriteFile(path, bytes, 0o644)
	if WriteFileError != nil {
		log.Printf("os.WriteFile error for path %s", path)
		return WriteFileError
	}

	return nil
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (m *MemStorage) SyncMetricSaving(h http.Handler) http.Handler {
	syncSavingFn := func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		h.ServeHTTP(rw, r)

		// Сохраняем после успешного POST запроса к /update
		if r.Method == http.MethodPost &&
			(strings.HasPrefix(r.URL.Path, "/update/") || r.URL.Path == "/update") &&
			rw.statusCode == http.StatusOK {
			if err := m.SaveToFile(); err != nil {
				log.Printf("Failed to save metrics: %v", err)
			} else {
				log.Println("Metrics saved synchronously")
			}
		}
	}
	return http.HandlerFunc(syncSavingFn)
}
