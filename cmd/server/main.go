package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	"github.com/akorablin/yandex-practicum-metrics/internal/config/db"
	logger "github.com/akorablin/yandex-practicum-metrics/internal/config/logger"
	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
	"github.com/akorablin/yandex-practicum-metrics/internal/middleware"
	dbStorage "github.com/akorablin/yandex-practicum-metrics/internal/repository/db"
	fileStorage "github.com/akorablin/yandex-practicum-metrics/internal/repository/file"
	memoryStorage "github.com/akorablin/yandex-practicum-metrics/internal/repository/memory"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
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

	// Инициализируем хранилище
	var repo storage.Storage
	var DB *sql.DB
	logger.Log.Info("Подключение к БД", zap.String("DataBaseDSN", cfg.DataBaseDSN))
	if DB, err = db.Init(cfg.DataBaseDSN); err != nil {
		log.Printf("БД недоступна: %v", err)
		repo = memoryStorage.New(cfg)
	} else {
		log.Printf("БД доступна!")
		repo = dbStorage.New(cfg, DB)
		defer DB.Close()
	}

	// Инициализируем обработчики запросов
	handlers := handler.NewHandlers(repo, DB)

	// Инициализируем структуру для работы с файлом
	file := fileStorage.New(cfg, repo)

	// Загруженам метрики из файла
	loadFileError := file.Load()
	if loadFileError != nil {
		return loadFileError
	}

	// Инициализируем обработчики запросов
	r := handlers.GetRoutes()

	// Обновление метрик
	if cfg.StoreInterval > 0 {
		// Через интервал времени
		ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		defer ticker.Stop()

		go func() {
			for range ticker.C {
				if err := file.Save(); err != nil {
					log.Printf("Failed to save metrics: %v", err)
				} else {
					log.Println("Metrics saved by StoreInterval")
				}
			}
		}()
	} else {
		// Синхронно через middleware
		r = middleware.SyncSaving(r, file)
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
	if err := file.Save(); err != nil {
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
