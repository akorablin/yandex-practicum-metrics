package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
	"github.com/akorablin/yandex-practicum-metrics/internal/repository/db/errors"
)

type PostgresStorage struct {
	db              *sql.DB
	cfg             *config.ServerConfig
	retryConfig     RetryConfig
	errorClassifier *errors.PostgresErrorClassifier
}

func New(cfg *config.ServerConfig, db *sql.DB) *PostgresStorage {
	return &PostgresStorage{
		db:              db,
		cfg:             cfg,
		retryConfig:     DefaultRetryConfig(),
		errorClassifier: errors.NewPostgresErrorClassifier(),
	}
}

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	DelayStep    time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		DelayStep:    2 * time.Second,
	}
}

func (p *PostgresStorage) UpdateGauge(name string, value float64) error {
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

	return err
}

func (p *PostgresStorage) UpdateCounter(name string, value int64) error {
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

	return err
}

func (p *PostgresStorage) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := ""
	countAttr := 0
	rowsSQL := make([]string, 0, len(metrics))
	argsSQL := make([]any, 0, len(metrics)*4)
	for _, metric := range metrics {
		row = "("
		for i := range 4 {
			countAttr++
			row = row + "$" + strconv.Itoa(countAttr)
			if i < 3 {
				row = row + ", "
			}
		}
		row = row + ")"
		rowsSQL = append(rowsSQL, row)
		argsSQL = append(argsSQL, metric.ID, metric.MType, metric.Value, metric.Delta)
	}
	sql := fmt.Sprintf(`INSERT INTO metrics (id, mtype, value, delta) 
		VALUES %s 
		ON CONFLICT (id) 
		DO UPDATE SET 
			value = EXCLUDED.value,
			delta = EXCLUDED.delta,
			updated_at = CURRENT_TIMESTAMP`,
		strings.Join(rowsSQL, ", "))
	err = p.retryExec(ctx, tx, sql, argsSQL...)
	if err != nil {
		return fmt.Errorf("ошибка при batch обновлении таблицы: %w", err)
	}

	return tx.Commit()
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

func (p *PostgresStorage) retryExec(ctx context.Context, tx *sql.Tx, sql string, argsSQL ...any) error {
	var lastErr error
	for attempt := 0; attempt < p.retryConfig.MaxAttempts; attempt++ {
		_, err := tx.Exec(sql, argsSQL...)
		if err == nil {
			return nil
		}
		lastErr = err

		if p.errorClassifier.Classify(err) != errors.Retriable {
			return fmt.Errorf("неповторяемая ошибка: %w", err)
		}

		log.Printf("Попытка %d завершилась ошибкой: %v", attempt+1, err)

		delay := p.retryConfig.InitialDelay + (time.Duration(attempt) * p.retryConfig.DelayStep)
		select {
		case <-ctx.Done():
			return fmt.Errorf("операция отменена: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("все %d попыток завершились с ошибкой, последняя ошибка: %w", p.retryConfig.MaxAttempts, lastErr)
}
