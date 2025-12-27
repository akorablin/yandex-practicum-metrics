package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
)

type Sender struct {
	client  *http.Client
	baseURL string
}

func NewSender(baseURL string) *Sender {
	return &Sender{
		client:  &http.Client{},
		baseURL: baseURL,
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

	return s.sendMetric(url, jsonData)
}

func (s *Sender) SendCounter(name string, value int64) error {
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

	return s.sendMetric(url, jsonData)
}

func (s *Sender) sendMetric(url string, data []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
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
