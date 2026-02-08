package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	Address         string
	LogLevel        string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DataBaseDSN     string
	HashKey         string
}

type AgentConfig struct {
	Address        string
	PollInterval   time.Duration
	ReportInterval time.Duration
	HashKey        string
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

func getEnvOrDefaultTimeDuration(envVar string, defaultValue time.Duration) time.Duration {
	if value, ok := os.LookupEnv(envVar); ok {
		if parsedValue, err := strconv.Atoi(value); err == nil {
			return time.Duration(parsedValue) * time.Second
		}
	}
	return defaultValue
}

func GetServerConfig() (*ServerConfig, error) {
	// Настройки из переменных окружения
	cfg := &ServerConfig{
		Address:         getEnvOrDefaultString("ADDRESS", "localhost:8080"),
		LogLevel:        getEnvOrDefaultString("LOG_LEVEL", "info"),
		StoreInterval:   getEnvOrDefaultInt("STORE_INTERVAL", 300),
		FileStoragePath: getEnvOrDefaultString("FILE_STORAGE_PATH", "tmp/metrics.json"),
		Restore:         getEnvOrDefaultBool("RESTORE", true),
		DataBaseDSN:     getEnvOrDefaultString("DATABASE_DSN", ""),
		HashKey:         getEnvOrDefaultString("KEY", ""),
	}

	// Настройки из командной строки
	serverAddress := flag.String("a", cfg.Address, "server address")
	logLevel := flag.String("l", cfg.LogLevel, "log level")
	storeInterval := flag.Int("i", cfg.StoreInterval, "store interval")
	fileStoragePath := flag.String("f", cfg.FileStoragePath, "file storage path")
	restore := flag.Bool("r", cfg.Restore, "restore")
	dataBaseDSN := flag.String("d", cfg.DataBaseDSN, "database dsn")
	hashKey := flag.String("k", cfg.HashKey, "hash key")
	flag.Parse()

	// Валидация командной строки
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		fmt.Fprintf(os.Stderr, "Usage options:\n")
		flag.PrintDefaults()
		return nil, fmt.Errorf("unknown arguments provided")
	}

	// Сохраняем настройки
	cfg.Address = *serverAddress
	cfg.LogLevel = *logLevel
	cfg.StoreInterval = *storeInterval
	cfg.FileStoragePath = *fileStoragePath
	cfg.Restore = *restore
	cfg.DataBaseDSN = *dataBaseDSN
	cfg.HashKey = *hashKey

	// Отображение настроек
	fmt.Println("Server Address:", cfg.Address)
	fmt.Println("Log Level:", cfg.LogLevel)
	fmt.Println("Store Interval:", cfg.StoreInterval)
	fmt.Println("File Storage Path:", cfg.FileStoragePath)
	fmt.Println("Restore:", cfg.Restore)
	fmt.Println("DataBaseDSN:", cfg.DataBaseDSN)
	fmt.Println("HashKey:", cfg.HashKey)
	fmt.Println("---------------")

	return cfg, nil
}

func GetAgentConfig() (*AgentConfig, error) {
	// Настройки из переменных окружения
	cfg := &AgentConfig{
		Address:        getEnvOrDefaultString("ADDRESS", "localhost:8080"),
		PollInterval:   getEnvOrDefaultTimeDuration("POLL_INTERVAL", 2*time.Second),
		ReportInterval: getEnvOrDefaultTimeDuration("REPORT_INTERVAL", 10*time.Second),
		HashKey:        getEnvOrDefaultString("KEY", ""),
	}

	// Настройки из командной строки
	serverAddress := flag.String("a", cfg.Address, "server address")
	pollInterval := flag.Int("p", int(cfg.PollInterval.Seconds()), "poll interval")
	reportInterval := flag.Int("r", int(cfg.ReportInterval.Seconds()), "report interval")
	hashKey := flag.String("k", cfg.HashKey, "hash key")
	flag.Parse()

	// Валидация командной строки
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		fmt.Fprintf(os.Stderr, "Usage options:\n")
		flag.PrintDefaults()
		return nil, fmt.Errorf("unknown arguments provided")
	}
	if *pollInterval <= 0 {
		fmt.Fprintf(os.Stderr, "Error: poll interval must be positive, got %d\n", *pollInterval)
		return nil, fmt.Errorf("incorrect pollInterval")
	}
	if *reportInterval <= 0 {
		fmt.Fprintf(os.Stderr, "Error: report interval must be positive, got %d\n", *reportInterval)
		return nil, fmt.Errorf("incorrect reportInterval")
	}

	// Сохраняем настройки
	cfg.Address = *serverAddress
	cfg.PollInterval = time.Duration(*pollInterval) * time.Second
	cfg.ReportInterval = time.Duration(*reportInterval) * time.Second
	cfg.HashKey = *hashKey

	// Отображение настроек
	fmt.Println("Server Address:", cfg.Address)
	fmt.Println("Poll Level:", cfg.PollInterval)
	fmt.Println("Report Interval:", cfg.ReportInterval)
	fmt.Println("HashKey:", cfg.HashKey)
	fmt.Println("---------------")

	return cfg, nil
}
