package logger

import (
	"fmt"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return fmt.Errorf("не удалось определить уровень логирования: %v", err)
	}

	// создаём новую конфигурацию логера
	cfg := zap.NewDevelopmentConfig()

	// устанавливаем уровень
	cfg.Level = lvl

	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("не удалось создать логгер: %v", err)
	}

	Log = zl
	return nil
}
