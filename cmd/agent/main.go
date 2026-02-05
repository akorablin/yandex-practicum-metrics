package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/akorablin/yandex-practicum-metrics/internal/agent"
	"github.com/akorablin/yandex-practicum-metrics/internal/config"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	// Получаем настройки конфигурации для агента
	cfg, err := config.GetAgentConfig()
	if err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	// Создаем "сборщик" метрик
	collector := agent.NewCollector()

	// Создаем "отправщик" метрик
	serverURL := cfg.Address
	if !strings.Contains(serverURL, "http://") && !strings.Contains(serverURL, "https://") {
		cfg.Address = "http://" + serverURL
	}
	sender := agent.NewSender(cfg)

	// Запускаем агент
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var wg sync.WaitGroup

	// Рутина "сборщика" метрик
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Started metrics collection with interval: %v", cfg.PollInterval)
		ticker := time.NewTicker(cfg.PollInterval)
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

	// Рутина "отправщика" метрик
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Started metrics reporting with interval: %v", cfg.ReportInterval)
		ticker := time.NewTicker(cfg.ReportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("Stopping metrics reporting...")
				return
			case <-ticker.C:
				gauges := collector.GetGauges()
				counters := collector.GetCounters()
				err := sender.SendAllMetricsJSON(ctx, gauges, counters)
				if err != nil {
					log.Printf("Failed to send metrics after retries: %v", err)
				} else {
					log.Printf("Successfully sent all metrics")
				}
			}
		}
	}()

	log.Printf("Agent is running. Press Ctrl+C to stop.")

	// Останавливаем агент
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
