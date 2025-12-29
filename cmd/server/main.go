package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
	"github.com/akorablin/yandex-practicum-metrics/internal/logger"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg, err := config.GetServerConfig()
	if err != nil {
		return fmt.Errorf("eror parsing flags: %w", err)
	}

	// Логирование
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	logger.Log.Info("Running server")

	memStorage := storage.NewMemStorage(cfg)
	handlers := handler.NewHandlers(memStorage)

	loadFileError := memStorage.LoadFromFile()
	if loadFileError != nil {
		return loadFileError
	}

	go func() {
		err = http.ListenAndServe(cfg.Address, logger.WithLogging(handlers.GetRoutes()))
		if err != nil {
			logger.Log.Error("Failed start server")
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := memStorage.SaveToFile()
				if err != nil {
					log.Printf("Failed to save metrics: %v", err)
				}
			case <-ctx.Done():
				err := memStorage.SaveToFile()
				if err != nil {
					log.Printf("Failed to save metrics: %v", err)
				}
				return
			}
		}
	}()

	log.Printf("Server is running. Press Ctrl+C to stop.")

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
		log.Printf("Server stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Printf("Shutdown timeout, forcing exit")
	}

	return nil
}
