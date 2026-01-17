package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	// Получаем конфиг для сервера
	cfg, err := config.GetServerConfig()
	if err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	// Создаем синглтон объект логирования
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	// Инициализируем сервер
	memStorage := storage.NewMemStorage(cfg)
	handlers := handler.NewHandlers(memStorage)

	loadFileError := memStorage.LoadFromFile()
	if loadFileError != nil {
		return loadFileError
	}

	r := handlers.GetRoutes()
	r = logger.WithLogging(r)

	// Инициализируем обновление метрик
	if cfg.StoreInterval > 0 {
		// Через интервал времени
		ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		defer ticker.Stop()

		go func() {
			for range ticker.C {
				if err := memStorage.SaveToFile(); err != nil {
					log.Printf("Failed to save metrics: %v", err)
				} else {
					log.Println("Metrics saved by StoreInterval")
				}
			}
		}()
	} else {
		// Синхронно через middleware
		r = memStorage.SyncMetricSaving(r)
	}

	// Запускаем сервер
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed start server: %v", err)
		}
	}()
	logger.Log.Info("Server started")
	log.Println("Server is running. Press Ctrl+C to stop.")

	// Отключаем сервер
	<-quit
	log.Println("Received shutdown signal...")

	log.Println("Saving metrics...")
	if err := memStorage.SaveToFile(); err != nil {
		log.Printf("Failed to save metrics: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to stop server: %v", err)
	}
	logger.Log.Info("Server stopped")

	return nil
}
