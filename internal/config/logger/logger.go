package logger

import (
	"fmt"

	"go.uber.org/zap"
)

func Initialize(level string) (*zap.Logger, error) {
	var Log *zap.Logger = zap.NewNop()

	// Преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, fmt.Errorf("не удалось определить уровень логирования: %v", err)
	}

	// Создаём новую конфигурацию логера
	cfg := zap.NewDevelopmentConfig()

	// Устанавливаем уровень
	cfg.Level = lvl

	// Создаём логер на основе конфигурации
	Log, err = cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать логгер: %v", err)
	}

	return Log, nil
}
