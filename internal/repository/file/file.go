package file

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
)

type Files struct {
	cfg     *config.ServerConfig
	storage storage.Storage
}

func New(cfg *config.ServerConfig, repo storage.Storage) *Files {
	return &Files{
		cfg:     cfg,
		storage: repo,
	}
}

func (f *Files) Load() error {
	if f.cfg.FileStoragePath == "" || !f.cfg.Restore {
		return nil
	}

	if _, err := os.Stat(f.cfg.FileStoragePath); err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	path := f.cfg.FileStoragePath
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
			f.storage.UpdateGauge(ID, *value)
		}
		if mType == "counter" {
			f.storage.UpdateCounter(ID, *delta)
		}
	}

	return nil
}

func (f *Files) Save() error {
	path := f.cfg.FileStoragePath
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	var gauges, counters = f.storage.GetAllMetrics()
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
