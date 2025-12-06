package agent

import (
	"math/rand/v2"
	"runtime"
)

type Collector struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewCollector() *Collector {
	return &Collector{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (c *Collector) UpdateMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Gauge метрики из runtime
	c.gauge["Alloc"] = float64(m.Alloc)
	c.gauge["BuckHashSys"] = float64(m.BuckHashSys)
	c.gauge["Frees"] = float64(m.Frees)
	c.gauge["GCCPUFraction"] = m.GCCPUFraction
	c.gauge["GCSys"] = float64(m.GCSys)
	c.gauge["HeapAlloc"] = float64(m.HeapAlloc)
	c.gauge["HeapIdle"] = float64(m.HeapIdle)
	c.gauge["HeapInuse"] = float64(m.HeapInuse)
	c.gauge["HeapObjects"] = float64(m.HeapObjects)
	c.gauge["HeapReleased"] = float64(m.HeapReleased)
	c.gauge["HeapSys"] = float64(m.HeapSys)
	c.gauge["LastGC"] = float64(m.LastGC)
	c.gauge["Lookups"] = float64(m.Lookups)
	c.gauge["MCacheInuse"] = float64(m.MCacheInuse)
	c.gauge["MCacheSys"] = float64(m.MCacheSys)
	c.gauge["MSpanInuse"] = float64(m.MSpanInuse)
	c.gauge["MSpanSys"] = float64(m.MSpanSys)
	c.gauge["Mallocs"] = float64(m.Mallocs)
	c.gauge["NextGC"] = float64(m.NextGC)
	c.gauge["NumForcedGC"] = float64(m.NumForcedGC)
	c.gauge["NumGC"] = float64(m.NumGC)
	c.gauge["OtherSys"] = float64(m.OtherSys)
	c.gauge["PauseTotalNs"] = float64(m.PauseTotalNs)
	c.gauge["StackInuse"] = float64(m.StackInuse)
	c.gauge["StackSys"] = float64(m.StackSys)
	c.gauge["Sys"] = float64(m.Sys)
	c.gauge["TotalAlloc"] = float64(m.TotalAlloc)

	// Счетчик
	c.counter["PollCount"]++

	// Случайное значение
	c.gauge["RandomValue"] = rand.Float64()
}

func (c *Collector) GetGauges() map[string]float64 {
	result := make(map[string]float64)
	for k, v := range c.gauge {
		result[k] = float64(v)
	}
	return result
}

func (c *Collector) GetCounters() map[string]int64 {
	result := make(map[string]int64)
	for k, v := range c.counter {
		result[k] = v
	}
	return result
}

func (c *Collector) GetMetricsCount() (int, int) {
	return len(c.gauge), len(c.counter)
}
