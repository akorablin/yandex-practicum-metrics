package main

import (
	"log"
	"strings"
	"time"

	"github.com/akorablin/yandex-practicum-metrics/internal/agent"
)

// Конфигурация агента
type AgentConfig struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

// Константы по умолчанию согласно требованиям
const (
	defaultServerAddress  = "localhost:8080"
	defaultPollInterval   = 2 * time.Second  // Обновление метрик каждые 2 секунды
	defaultReportInterval = 10 * time.Second // Отправка метрик каждые 10 секунд
)

func main() {
	config := &AgentConfig{}

	// Инициализируем конфиг
	config.ServerAddress = defaultServerAddress
	config.ReportInterval = defaultPollInterval
	config.PollInterval = defaultReportInterval

	// Создаем компоненты
	collector := agent.NewCollector()

	// Сбор метрик
	collector.UpdateMetrics()
	gaugeCount, counterCount := collector.GetMetricsCount()
	log.Printf("Collected metrics: %d gauges, %d counters", gaugeCount, counterCount)

	// Сон на 10 секунды
	time.Sleep(config.PollInterval)

	// Отправка метрик
	serverURL := config.ServerAddress
	if !strings.Contains(serverURL, "http://") && !strings.Contains(serverURL, "https://") {
		serverURL = "http://" + serverURL
	}
	sender := agent.NewSender(serverURL)
	gauges := collector.GetGauges()
	counters := collector.GetCounters()
	if err := sender.SendAllMetrics(gauges, counters); err != nil {
		log.Printf("Failed to send metrics: %v", err)
	} else {
		log.Printf("Successfully sent all metrics")
	}
}
