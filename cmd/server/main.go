package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

	ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
	defer ticker.Stop()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	wd, _ := os.Getwd()
	dir, _ := filepath.Split(cfg.FileStoragePath)
	if err := os.MkdirAll(filepath.Join(wd, dir), 0o777); err != nil {
		fmt.Println(err)
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				err := memStorage.SaveToFile()
				if err != nil {
					log.Printf("Failed to save metrics: %v", err)
				}
			case <-done:
				err := memStorage.SaveToFile()
				if err != nil {
					log.Printf("Failed to save metrics: %v", err)
				}
				close(done)
				return
			}
		}
	}()

	<-done

	return nil
}
