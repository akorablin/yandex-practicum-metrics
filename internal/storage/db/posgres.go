package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	"github.com/akorablin/yandex-practicum-metrics/internal/config/db"
	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
)

type PostgresStorage struct {
	db  *sql.DB
	cfg *config.ServerConfig
}

func New(cfg *config.ServerConfig) *PostgresStorage {
	return &PostgresStorage{
		db:  db.GetDB(),
		cfg: cfg,
	}
}

func (p *PostgresStorage) UpdateGauge(name string, value float64) {
	_, err := p.db.Exec(`
		INSERT INTO metrics (id, mtype, value, delta) 
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id) 
		DO UPDATE SET 
			value = $3,
			delta = NULL,
			updated_at = CURRENT_TIMESTAMP
	`, name, "gauge", value)

	if err != nil {
		log.Printf("Ошибка сохранения gauge метрики: %v", err)
	}
}

func (p *PostgresStorage) UpdateCounter(name string, value int64) {
	_, err := p.db.Exec(`
		INSERT INTO metrics (id, mtype, delta, value) 
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id) 
		DO UPDATE SET 
			delta = COALESCE(metrics.delta, 0) + $3,
			value = NULL,
			updated_at = CURRENT_TIMESTAMP
	`, name, "counter", value)

	if err != nil {
		log.Printf("Ошибка сохранения counter метрики: %v", err)
	}
}

func (p *PostgresStorage) GetGauge(name string) (float64, error) {
	var value float64
	err := p.db.QueryRow(
		"SELECT value FROM metrics WHERE mtype = $1 AND id = $2 AND value IS NOT NULL",
		"gauge", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		log.Printf("Ошибка получения gauge метрики: %v", err)
		return 0, err
	}

	return value, nil
}

func (p *PostgresStorage) GetCounter(name string) (int64, error) {
	var value int64
	err := p.db.QueryRow(
		"SELECT delta FROM metrics WHERE mtype = $1 AND id = $2 AND delta IS NOT NULL",
		"counter", name).Scan(&value)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		log.Printf("Ошибка получения counter метрики: %v", err)
		return 0, err
	}

	return value, nil
}

func (p *PostgresStorage) GetAllMetrics() (map[string]float64, map[string]int64) {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	// Получаем все gauge метрики
	rows, err := p.db.Query(
		"SELECT id, value FROM metrics WHERE mtype = 'gauge' AND value IS NOT NULL")
	if err != nil {
		log.Printf("Ошибка получения gauge метрик: %v", err)
		return gauges, counters
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var value float64
		if err := rows.Scan(&id, &value); err != nil {
			log.Printf("Ошибка сканирования gauge метрики: %v", err)
			continue
		}
		gauges[id] = value
	}
	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при итерации gauge метрик: %v", err)
	}

	// Получаем все counter метрики
	rows, err = p.db.Query(
		"SELECT id, delta FROM metrics WHERE mtype = 'counter' AND delta IS NOT NULL")
	if err != nil {
		log.Printf("Ошибка получения counter метрик: %v", err)
		return gauges, counters
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var value int64
		if err := rows.Scan(&id, &value); err != nil {
			log.Printf("Ошибка сканирования counter метрики: %v", err)
			continue
		}
		counters[id] = value
	}
	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при итерации counter метрик: %v", err)
	}

	return gauges, counters
}

func (p *PostgresStorage) LoadFromFile() error {
	if p.cfg.FileStoragePath == "" || !p.cfg.Restore {
		return nil
	}

	if _, err := os.Stat(p.cfg.FileStoragePath); err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	path := p.cfg.FileStoragePath
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
			p.UpdateGauge(ID, *value)
		}
		if mType == "counter" {
			p.UpdateCounter(ID, *delta)
		}
	}

	return nil
}

func (p *PostgresStorage) SaveToFile() error {
	path := p.cfg.FileStoragePath
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	var gauges, counters = p.GetAllMetrics()
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
