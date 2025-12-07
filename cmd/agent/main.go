package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/akorablin/yandex-practicum-metrics/internal/agent"
)

// Конфигурация агента
type AgentConfig struct {
	Address        string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

// Константы по умолчанию согласно требованиям
const (
	defaultAddress        = "localhost:8080"
	defaultPollInterval   = 2 * time.Second
	defaultReportInterval = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Printf("Application error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := parseFlags()
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	log.Printf("Starting metrics agent with config:")
	log.Printf("  Server address: %s", config.Address)
	log.Printf("  Poll interval: %v", config.PollInterval)
	log.Printf("  Report interval: %v", config.ReportInterval)

	// Создаем компоненты
	collector := agent.NewCollector()

	// Сбор метрик
	collector.UpdateMetrics()
	gaugeCount, counterCount := collector.GetMetricsCount()
	log.Printf("Collected metrics: %d gauges, %d counters", gaugeCount, counterCount)

	// Сон
	time.Sleep(config.ReportInterval)

	// Отправка метрик
	serverURL := config.Address
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

	return nil
}

func parseFlags() (*AgentConfig, error) {
	config := &AgentConfig{}

	// Работа с командной строкой
	var reportInterval, pollInterval int
	flag.StringVar(&config.Address, "a", defaultAddress, "HTTP server endpoint address")
	flag.IntVar(&reportInterval, "r", int(defaultReportInterval.Seconds()), "Report interval in seconds")
	flag.IntVar(&pollInterval, "p", int(defaultPollInterval.Seconds()), "Poll interval in seconds")
	flag.Parse()

	// Валидация
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		fmt.Fprintf(os.Stderr, "Usage options:\n")
		flag.PrintDefaults()
		return nil, fmt.Errorf("unknown arguments provided")
	}
	if reportInterval <= 0 {
		fmt.Fprintf(os.Stderr, "Error: report interval must be positive, got %d\n", reportInterval)
		os.Exit(1)
	}
	if pollInterval <= 0 {
		fmt.Fprintf(os.Stderr, "Error: poll interval must be positive, got %d\n", pollInterval)
		os.Exit(1)
	}

	// Конвертируем в time.Duration
	config.ReportInterval = time.Duration(reportInterval) * time.Second
	config.PollInterval = time.Duration(pollInterval) * time.Second

	return config, nil
}
