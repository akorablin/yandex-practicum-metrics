package main

import (
	"net/http"
	"strconv"
	"strings"
)

type MetricsStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAllMetrics() (map[string]float64, map[string]int64)
}

func updateHandler(res http.ResponseWriter, req *http.Request) {
	// Проверяем HTTP-метод
	if req.Method != http.MethodPost {
		http.Error(res, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем заголовок "Content-Type"
	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(res, "\"Content-Type\" header error!", http.StatusBadRequest)
		return
	}

	// Разбиваем URL на части
	path := strings.TrimPrefix(req.URL.Path, "/update/")
	parts := strings.Split(path, "/")

	// Проверяем URL
	if len(parts) != 3 {
		invalidUrlTextError := "Invalid URL format. Expected: /update/type/name/value"
		if parts[0] == "gauge" || parts[0] == "counter" {
			http.Error(res, invalidUrlTextError, http.StatusNotFound)
		} else {
			http.Error(res, invalidUrlTextError, http.StatusBadRequest)
		}
		return
	}

	switch parts[0] {
	case "gauge":
		_, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			http.Error(res, "Invalid gauge value", http.StatusBadRequest)
			return
		}

	case "counter":
		_, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			http.Error(res, "Invalid counter value", http.StatusBadRequest)
			return
		}

	default:
		http.Error(res, "Unknown metric type. Use 'gauge' or 'counter'", http.StatusBadRequest)
		return
	}

	responseText := "OK\n"
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.Header().Set("Content-Length", strconv.Itoa(len(responseText)))
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(responseText))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", updateHandler)

	err := http.ListenAndServe(`localhost:8080`, mux)
	if err != nil {
		panic(err)
	}
}
