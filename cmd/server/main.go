package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
	"github.com/akorablin/yandex-practicum-metrics/internal/logger"
	"go.uber.org/zap"
)

type ServerConfig struct {
	Address  string
	LogLevel string
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	// Флаги командной строки
	config, err := parseFlags()
	if err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	// Флаги из переменных окружения
	config, err = applyEnv(config)
	if err != nil {
		return fmt.Errorf("error apply env: %w", err)
	}

	// Логирование
	if err := logger.Initialize(config.LogLevel); err != nil {
		return err
	}

	logger.Log.Info("Running server", zap.String("address", config.Address))

	handlers := handler.NewHandlers()
	return http.ListenAndServe(config.Address, logger.WithLogging(handlers.GetRoutes()))
}

func parseFlags() (*ServerConfig, error) {
	config := &ServerConfig{}

	// Работа с командной строкой
	flag.StringVar(&config.Address, "a", "localhost:8080", "HTTP server endpoint address")
	flag.StringVar(&config.LogLevel, "l", "info", "Log level")
	flag.Parse()

	// Валидация
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		fmt.Fprintf(os.Stderr, "Usage options:\n")
		flag.PrintDefaults()
		return nil, fmt.Errorf("unknown arguments provided")
	}

	return config, nil
}

func applyEnv(config *ServerConfig) (*ServerConfig, error) {
	// Переменная окружения ADDRESS
	if addr := os.Getenv("ADDRESS"); addr != "" {
		config.Address = addr
	}

	return config, nil
}
