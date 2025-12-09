package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
)

type Handlers struct {
	storage storage.Storage
}

func NewHandlers(storage storage.Storage) *Handlers {
	return &Handlers{storage: storage}
}

func (h *Handlers) UpdateHandler(res http.ResponseWriter, req *http.Request) {
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
		if parts[0] == "gauge" || parts[0] == "counter" {
			http.Error(res, "Invalid URL format for type", http.StatusNotFound)
		} else {
			http.Error(res, "Invalid URL format. Expected: /update/{type}/{name}/{value}", http.StatusBadRequest)
		}
		return
	}

	metricName := parts[0]
	value := parts[2]

	switch metricName {
	case "gauge":
		value, err := strconv.ParseFloat(value, 64)
		if err != nil {
			http.Error(res, "Invalid gauge value", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metricName, value)
		log.Printf("Updated gauge %s = %.6f", metricName, value)

	case "counter":
		value, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			http.Error(res, "Invalid counter value", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(metricName, value)
		log.Printf("Updated counter %s (added %d)", metricName, value)

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
