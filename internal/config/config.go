package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type ServerConfig struct {
	Address         string
	LogLevel        string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

func getEnvOrDefaultString(envVar string, defaultValue string) string {
	if value, ok := os.LookupEnv(envVar); ok {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(envVar string, defaultValue int) int {
	if value, ok := os.LookupEnv(envVar); ok {
		if parsedValue, err := strconv.Atoi(value); err == nil {
			return parsedValue
		}
	}
	return defaultValue
}

func getEnvOrDefaultBool(envVar string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(envVar); ok {
		if parsedValue, err := strconv.ParseBool(value); err == nil {
			return parsedValue
		}
	}
	return defaultValue
}

func GetServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{
		Address:         getEnvOrDefaultString("ADDRESS", "localhost:8080"),
		LogLevel:        getEnvOrDefaultString("LOG_LEVEL", "info"),
		StoreInterval:   getEnvOrDefaultInt("STORE_INTERVAL", 300),
		FileStoragePath: getEnvOrDefaultString("FILE_STORAGE_PATH", "tmp/metrics.json"),
		Restore:         getEnvOrDefaultBool("RESTORE", true),
	}

	serverAddress := flag.String("a", cfg.Address, "server address")
	logLevel := flag.String("l", cfg.LogLevel, "log level")
	storeInterval := flag.Int("i", cfg.StoreInterval, "store interval")
	fileStoragePath := flag.String("f", cfg.FileStoragePath, "file storage path")
	restore := flag.Bool("r", cfg.Restore, "restore")

	flag.Parse()

	// Валидация
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		fmt.Fprintf(os.Stderr, "Usage options:\n")
		flag.PrintDefaults()
		return nil, fmt.Errorf("unknown arguments provided")
	}

	cfg.Address = *serverAddress
	cfg.LogLevel = *logLevel
	cfg.StoreInterval = *storeInterval
	cfg.FileStoragePath = *fileStoragePath
	cfg.Restore = *restore

	fmt.Println("Server Address:", cfg.Address)
	fmt.Println("Log Level:", cfg.LogLevel)
	fmt.Println("Store Interval:", cfg.StoreInterval)
	fmt.Println("File Storage Path:", cfg.FileStoragePath)
	fmt.Println("Restore:", cfg.Restore)

	return cfg, nil
}
