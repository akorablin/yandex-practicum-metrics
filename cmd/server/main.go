package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
)

type ServerConfig struct {
	Address string
}

func main() {
	if err := run(); err != nil {
		panic(err)
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

	fmt.Println("Running server on", config.Address)
	handlers := handler.NewHandlers()
	return http.ListenAndServe(config.Address, handlers.GetRoutes())
}

func parseFlags() (*ServerConfig, error) {
	config := &ServerConfig{}

	// Работа с командной строкой
	flag.StringVar(&config.Address, "a", "localhost:8080", "HTTP server endpoint address")
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
