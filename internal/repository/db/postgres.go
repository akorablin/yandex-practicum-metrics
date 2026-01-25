package db

import (
	"database/sql"
	"log"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	"github.com/akorablin/yandex-practicum-metrics/internal/config/db"
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
