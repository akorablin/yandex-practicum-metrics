package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
		return fmt.Errorf("error parsing flags: %w", err)
	}

	config, err = applyEnv(config)
	if err != nil {
		return fmt.Errorf("error apply env: %w", err)
	}

	log.Printf("Starting metrics agent with config:")
	log.Printf("  Server address: %s", config.Address)
	log.Printf("  Poll interval: %v", config.PollInterval)
	log.Printf("  Report interval: %v", config.ReportInterval)

	// Создаем "сборщик" метрик
	collector := agent.NewCollector()

	// Создаем "отправщик" метрик
	serverURL := config.Address
	if !strings.Contains(serverURL, "http://") && !strings.Contains(serverURL, "https://") {
		serverURL = "http://" + serverURL
	}
	sender := agent.NewSender(serverURL)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Started metrics collection with interval: %v", config.PollInterval)
		ticker := time.NewTicker(config.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping metrics collection...")
				return
			case <-ticker.C:
				collector.UpdateMetrics()
				gaugeCount, counterCount := collector.GetMetricsCount()
				log.Printf("Collected metrics: %d gauges, %d counters", gaugeCount, counterCount)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Started metrics reporting with interval: %v", config.ReportInterval)
		ticker := time.NewTicker(config.ReportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("Stopping metrics reporting...")
				return
			case <-ticker.C:
				gauges := collector.GetGauges()
				counters := collector.GetCounters()
				if err := sender.SendAllMetrics(gauges, counters); err != nil {
					log.Printf("Failed to send metrics: %v", err)
				} else {
					log.Printf("Successfully sent all metrics")
				}
			}
		}
	}()

	log.Printf("Agent is running. Press Ctrl+C to stop.")

	<-ctx.Done()
	log.Printf("Received shutdown signal...")

	stop()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("Agent stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Printf("Shutdown timeout, forcing exit")
	}

	return nil
}

func parseFlags() (*AgentConfig, error) {
	config := &AgentConfig{}

	// Работа с командной строкой
	var pollInterval, reportInterval int
	flag.StringVar(&config.Address, "a", defaultAddress, "HTTP server endpoint address")
	flag.IntVar(&pollInterval, "p", int(defaultPollInterval.Seconds()), "Poll interval in seconds")
	flag.IntVar(&reportInterval, "r", int(defaultReportInterval.Seconds()), "Report interval in seconds")
	flag.Parse()

	// Валидация
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		fmt.Fprintf(os.Stderr, "Usage options:\n")
		flag.PrintDefaults()
		return nil, fmt.Errorf("unknown arguments provided")
	}
	if pollInterval <= 0 {
		fmt.Fprintf(os.Stderr, "Error: poll interval must be positive, got %d\n", pollInterval)
		os.Exit(1)
	}
	if reportInterval <= 0 {
		fmt.Fprintf(os.Stderr, "Error: report interval must be positive, got %d\n", reportInterval)
		os.Exit(1)
	}

	// Конвертируем в time.Duration
	config.ReportInterval = time.Duration(reportInterval) * time.Second
	config.PollInterval = time.Duration(pollInterval) * time.Second

	return config, nil
}

func applyEnv(config *AgentConfig) (*AgentConfig, error) {
	// Переменная окружения ADDRESS
	if addr := os.Getenv("ADDRESS"); addr != "" {
		config.Address = addr
	}

	// Переменная окружения POLL_INTERVAL
	if pollStr := os.Getenv("POLL_INTERVAL"); pollStr != "" {
		sec, err := strconv.Atoi(pollStr)
		if err != nil {
			return nil, fmt.Errorf("invalid POLL_INTERVAL: %w", err)
		}
		config.PollInterval = time.Duration(sec) * time.Second
	}

	// Переменная окружения REPORT_INTERVAL
	if reportStr := os.Getenv("REPORT_INTERVAL"); reportStr != "" {
		sec, err := strconv.Atoi(reportStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REPORT_INTERVAL: %w", err)
		}
		config.ReportInterval = time.Duration(sec) * time.Second
	}

	return config, nil
}
