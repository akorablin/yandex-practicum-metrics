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
	db "github.com/akorablin/yandex-practicum-metrics/internal/config/db"
	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
	"github.com/akorablin/yandex-practicum-metrics/internal/middleware"
	logger "github.com/akorablin/yandex-practicum-metrics/internal/middleware"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
	dbStorage "github.com/akorablin/yandex-practicum-metrics/internal/storage/db"
	memoryStorage "github.com/akorablin/yandex-practicum-metrics/internal/storage/memory"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	// Получаем настройки конфигурации для сервера
	cfg, err := config.GetServerConfig()
	if err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	// Создаем "singleton" логирования
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	var repo storage.Storage
	var useDB bool

	// Инициализируем хранилище
	logger.Log.Info("Подключение к БД", zap.String("DataBaseDSN", cfg.DataBaseDSN))
	if err := db.Init(cfg.DataBaseDSN); err != nil {
		useDB = false
		log.Printf("БД недоступна: %v", err)
		repo = memoryStorage.New(cfg)
	} else {
		useDB = true
		log.Printf("БД доступна!")
		repo = dbStorage.New(cfg)
	}

	// Инициализируем обработчики запросов
	handlers := handler.NewHandlers(repo)

	// Загруженам метрики из файла
	loadFileError := repo.LoadFromFile()
	if loadFileError != nil {
		return loadFileError
	}

	// Инициализируем обработчики запросов
	r := handlers.GetRoutes()

	// Подключаем middleware c логированием
	r = logger.WithLogging(r)

	// Обновление метрик
	if cfg.StoreInterval > 0 {
		// Через интервал времени
		ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		defer ticker.Stop()

		go func() {
			for range ticker.C {
				if err := repo.SaveToFile(); err != nil {
					log.Printf("Failed to save metrics: %v", err)
				} else {
					log.Println("Metrics saved by StoreInterval")
				}
			}
		}()
	} else {
		// Синхронно через middleware
		r = middleware.SyncSaving(r, repo)
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
	if err := repo.SaveToFile(); err != nil {
		log.Printf("Failed to save metrics: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to stop server: %v", err)
	}
	if useDB {
		db.DB.Close()
	}
	logger.Log.Info("Server stopped")

	return nil
}
