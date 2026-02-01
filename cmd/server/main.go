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
	"github.com/akorablin/yandex-practicum-metrics/internal/config/logger"
	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
	"github.com/akorablin/yandex-practicum-metrics/internal/middleware"
	dbRepo "github.com/akorablin/yandex-practicum-metrics/internal/repository/db"
	memoryRepo "github.com/akorablin/yandex-practicum-metrics/internal/repository/memory"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
	fileStorage "github.com/akorablin/yandex-practicum-metrics/internal/storage/file"
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

	// Инициализируем логирование
	var Log *zap.Logger
	if Log, err = logger.Initialize(cfg.LogLevel); err != nil {
		return fmt.Errorf("ошибка инициализации логирования: %w", err)
	}

	// Инициализируем хранилище (БД или оперативная память)
	var DB *sql.DB
	var repo storage.Storage
	log.Printf("Подключение к БД: %s", cfg.DataBaseDSN)
	if DB, err = db.Init(cfg.DataBaseDSN); err != nil {
		log.Printf("БД недоступна: %v", err)
		repo = memoryRepo.New(cfg)
	} else {
		log.Printf("БД доступна!")
		repo = dbRepo.New(cfg, DB)
		defer DB.Close()
	}

	// Инициализируем обработчики запросов
	handlers := handler.NewHandlers(repo, DB, Log)

	// Загруженам метрики из файла
	file := fileStorage.New(cfg, repo)
	loadFileError := file.Load()
	if loadFileError != nil {
		return loadFileError
	}

	// Получаем роутинг
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
	log.Println("Server started")
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
	log.Println("Server stopped")

	return nil
}
