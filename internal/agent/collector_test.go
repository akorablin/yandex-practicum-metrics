package agent

import (
	"testing"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector()

	if collector == nil {
		t.Fatal("NewCollector() returned nil")
	}

	if collector.gauge == nil {
		t.Error("gauges map is nil")
	}

	if collector.counter == nil {
		t.Error("counters map is nil")
	}
}

func TestUpdateMetrics(t *testing.T) {
	collector := NewCollector()

	// Проверяем начальное состояние
	gauge := collector.GetGauges()
	counter := collector.GetCounters()

	if len(gauge) != 0 {
		t.Errorf("Expected 0 initial gauges, got %d", len(gauge))
	}

	if len(counter) != 0 {
		t.Errorf("Expected 0 initial counters, got %d", len(counter))
	}

	// Обновляем метрики
	collector.UpdateMetrics()

	gauge = collector.GetGauges()
	counter = collector.GetCounters()

	// Проверяем что метрики собрались
	if len(gauge) == 0 {
		t.Error("No gauge metrics collected")
	}

	if len(counter) == 0 {
		t.Error("No counter metrics collected")
	}

	// Проверяем наличие обязательных метрик
	if _, exists := gauge["RandomValue"]; !exists {
		t.Error("RandomValue gauge metric not found")
	}

	if pollCount, exists := counter["PollCount"]; !exists {
		t.Error("PollCount counter metric not found")
	} else if pollCount != 1 {
		t.Errorf("Expected PollCount = 1, got %d", pollCount)
	}

	// Проверяем некоторые runtime метрики
	requiredGauges := []string{
		"Alloc", "Sys", "HeapAlloc", "HeapSys", "NumGC", "GCCPUFraction",
	}

	for _, name := range requiredGauges {
		if _, exists := gauge[name]; !exists {
			t.Errorf("Required gauge metric %s not found", name)
		}
	}
}
