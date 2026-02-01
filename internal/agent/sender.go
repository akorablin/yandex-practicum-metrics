package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
)

type Sender struct {
	client      *http.Client
	baseURL     string
	retryConfig RetryConfig
}

func NewSender(baseURL string) *Sender {
	return &Sender{
		client:      &http.Client{},
		baseURL:     baseURL,
		retryConfig: DefaultRetryConfig(),
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

func (s *Sender) SendAllMetrics(gauge map[string]float64, counter map[string]int64) error {
	totalMetrics := len(gauge) + len(counter)
	sentMetrics := 0

	log.Printf("Sending %d gauge metrics and %d counter metrics", len(gauge), len(counter))

	// Отправляем все gauge метрики
	for name, value := range gauge {
		if err := s.SendGauge(name, value); err != nil {
			return fmt.Errorf("failed to send gauge %s: %w", name, err)
		}
		sentMetrics++
	}

	// Отправляем все counter метрики
	for name, value := range counter {
		if err := s.SendCounter(name, value); err != nil {
			return fmt.Errorf("failed to send counter %s: %w", name, err)
		}
		sentMetrics++
	}

	log.Printf("Successfully sent %d/%d metrics", sentMetrics, totalMetrics)
	return nil
}

func (s *Sender) SendGauge(name string, value float64) error {
	url := fmt.Sprintf("%s/update/gauge/%s/%s", s.baseURL, name, strconv.FormatFloat(value, 'f', -1, 64))
	return s.sendMetric(url, "gauge", name)
}

func (s *Sender) SendCounter(name string, value int64) error {
	url := fmt.Sprintf("%s/update/counter/%s/%d", s.baseURL, name, value)
	return s.sendMetric(url, "counter", name)
}

func (s *Sender) sendMetric(url, metricType, metricName string) error {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d for %s %s", resp.StatusCode, metricType, metricName)
	}

	return nil
}

func (s *Sender) SendAllMetricsJSON(ctx context.Context, gauge map[string]float64, counter map[string]int64) error {
	totalMetrics := len(gauge) + len(counter)

	log.Printf("Sending %d gauge metrics and %d counter metrics", len(gauge), len(counter))

	var metricItem models.Metrics
	data := make([]models.Metrics, 0, totalMetrics)

	// Формируем gauge метрики
	for name, value := range gauge {
		metricItem = models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &value,
		}
		data = append(data, metricItem)
	}

	// Отправляем все counter метрики
	for name, value := range counter {
		metricItem = models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &value,
		}
		data = append(data, metricItem)
	}

	return s.SendBatchJSON(ctx, data)
}

func (s *Sender) SendGaugeJSON(ctx context.Context, name string, value float64) error {
	url := fmt.Sprintf("%s/update", s.baseURL)

	data := models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	return s.sendMetricJSON(ctx, url, jsonData)
}

func (s *Sender) SendCounterJSON(ctx context.Context, name string, value int64) error {
	url := fmt.Sprintf("%s/update", s.baseURL)

	data := models.Metrics{
		ID:    name,
		MType: "counter",
		Delta: &value,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	return s.sendMetricJSON(ctx, url, jsonData)
}

func (s *Sender) SendBatchJSON(ctx context.Context, data []models.Metrics) error {
	url := fmt.Sprintf("%s/updates/", s.baseURL)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	return s.sendMetricJSON(ctx, url, jsonData)
}

func (s *Sender) sendMetricJSON(ctx context.Context, url string, data []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := s.retryRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s", string(body))
	}

	return nil
}

func (s *Sender) retryRequest(ctx context.Context, request *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < s.retryConfig.MaxAttempts; attempt++ {
		resp, err := s.client.Do(request)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		delay := s.retryConfig.InitialDelay + (time.Duration(attempt) * s.retryConfig.DelayStep)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("операция отменена: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return nil, fmt.Errorf("все %d попыток завершились ошибкой, последняя ошибка: %w", s.retryConfig.MaxAttempts, lastErr)
}
