package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/akorablin/yandex-practicum-metrics/internal/middleware"
	models "github.com/akorablin/yandex-practicum-metrics/internal/model"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type Handlers struct {
	storage storage.Storage
	db      *sql.DB
	logger  *zap.Logger
}

func NewHandlers(repo storage.Storage, db *sql.DB, logger *zap.Logger) *Handlers {
	return &Handlers{
		storage: repo,
		db:      db,
		logger:  logger,
	}
}

func (h *Handlers) GetRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logging(*h.logger))

	r.Post("/update/{type}/{name}/{value}", h.updateHandler)
	r.Get("/value/{type}/{name}", h.valueHandler)
	r.Post("/update/", h.updateMetricJSONHandler)
	r.Post("/updates/", h.UpdateMetricsBatch)
	r.Post("/value/", h.valueMetricJSONHandler)
	r.Get("/ping", h.pingHandler)
	r.Get("/", h.rootHandler)

	return r
}

func (h *Handlers) updateHandler(res http.ResponseWriter, req *http.Request) {
	// Проверяем HTTP-метод
	if req.Method != http.MethodPost {
		http.Error(res, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
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

	metricType := parts[0]
	metricName := parts[1]
	value := parts[2]

	switch metricType {
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

func (h *Handlers) valueHandler(res http.ResponseWriter, req *http.Request) {
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "name")

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")

	switch metricType {
	case "gauge":
		value, err := h.storage.GetGauge(metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.Error(res, "Gauge metric not found", http.StatusNotFound)
			return
		}

		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "%g", value)

	case "counter":
		value, err := h.storage.GetCounter(metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.Error(res, "Counter metric not found", http.StatusNotFound)
			return
		}
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "%d", value)

	default:
		http.Error(res, "Unknown metric type. Use 'gauge' or 'counter'", http.StatusBadRequest)
	}
}

func (h *Handlers) rootHandler(res http.ResponseWriter, req *http.Request) {
	var gaugesCopy, countersCopy = h.storage.GetAllMetrics()

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Metrics Server</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            margin: 40px; 
            background-color: #f5f5f5; 
        }
        .container {
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        table { 
            border-collapse: collapse; 
            width: 100%; 
            margin-bottom: 20px; 
        }
        th, td { 
            border: 1px solid #ddd; 
            padding: 12px; 
            text-align: left; 
        }
        th { 
            background-color: #4CAF50; 
            color: white;
        }
        tr:nth-child(even) {
            background-color: #f2f2f2;
        }
        h1 { 
            color: #333; 
            text-align: center;
        }
        h2 { 
            color: #4CAF50; 
            border-bottom: 2px solid #4CAF50;
            padding-bottom: 10px;
        }
        .count {
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Metrics Server Dashboard</h1>
        
        <h2>Gauges <span class="count">({{len .Gauges}})</span></h2>
        <table>
            <tr><th>Name</th><th>Value</th></tr>
            {{range $name, $value := .Gauges}}
            <tr><td><strong>{{$name}}</strong></td><td>{{printf "%.6f" $value}}</td></tr>
            {{else}}
            <tr><td colspan="2" style="text-align: center; color: #666;">No gauges available</td></tr>
            {{end}}
        </table>
        
        <h2>Counters <span class="count">({{len .Counters}})</span></h2>
        <table>
            <tr><th>Name</th><th>Value</th></tr>
            {{range $name, $value := .Counters}}
            <tr><td><strong>{{$name}}</strong></td><td>{{$value}}</td></tr>
            {{else}}
            <tr><td colspan="2" style="text-align: center; color: #666;">No counters available</td></tr>
            {{end}}
        </table>
        
        <div style="margin-top: 30px; padding: 15px; background-color: #e7f3ff; border-left: 4px solid #2196F3;">
            <h3>API Endpoints:</h3>
            <ul>
                <li><code>POST /update/{type}/{name}/{value}- Update metric</code> </li>
                <li><code>GET /value/{type}/{name} - Get metric value</code></li>
				<li><code>POST /update - Update metric (JSON)</code></li>
                <li><code>GET /value - Get metric value (JSON)</code></li>
				<li><code>GET /ping - Ping DB</code></li>
				<li><code>GET / - This dashboard</code></li>
            </ul>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("metrics").Parse(tmpl)
	if err != nil {
		http.Error(res, "Template error", http.StatusInternalServerError)
		log.Printf("Template parse error: %v", err)
		return
	}

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   gaugesCopy,
		Counters: countersCopy,
	}

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.WriteHeader(http.StatusOK)

	if err := t.Execute(res, data); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func (h *Handlers) updateMetricJSONHandler(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "cannot read body", http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	var m models.Metrics
	if err := json.Unmarshal(body, &m); err != nil {
		http.Error(res, "invalid json", http.StatusBadRequest)
		return
	}
	if m.ID == "" || (m.MType != "gauge" && m.MType != "counter") {
		http.Error(res, "invalid metric id or type", http.StatusBadRequest)
		return
	}

	switch m.MType {
	case "gauge":
		if m.Value == nil {
			http.Error(res, "missing value for gauge", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(m.ID, *m.Value)
	case "counter":
		if m.Delta == nil {
			http.Error(res, "missing delta for counter", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(m.ID, *m.Delta)
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`{"status":"ok"}`))
}

func (h *Handlers) UpdateMetricsBatch(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Получаем метрики из тела запроса
	var metrics []models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(map[string]string{"error": "Invalid JSON format"})
		return
	}

	// Проверяем количество метрик на пустоту
	if len(metrics) == 0 {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(map[string]string{"error": "Empty batch"})
		return
	}

	// Валидация метрик
	var validationErrors []string
	for i, metric := range metrics {
		if metric.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("metric[%d]: ID is required", i))
			continue
		}

		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				validationErrors = append(validationErrors, fmt.Sprintf("metric[%d]: gauge value is required", i))
			}
		case models.Counter:
			if metric.Delta == nil {
				validationErrors = append(validationErrors, fmt.Sprintf("metric[%d]: counter delta is required", i))
			}
		default:
			validationErrors = append(validationErrors, fmt.Sprintf("metric[%d]: unknown metric type: %s", i, metric.MType))
		}
	}
	if len(validationErrors) > 0 {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(map[string]any{
			"error":   "validation failed",
			"details": validationErrors,
		})
		return
	}

	// Удаляем дубликаты метрик и сразу суммируем counters
	uniqueMetrics := []models.Metrics{}
	uniqueKeys := make(map[string]int)
	uniqueGaugeValues := make(map[string]float64)
	uniqueCounterValues := make(map[string]int64)
	for i, metric := range metrics {
		uniqueKeys[metric.ID] = i
		switch metric.MType {
		case models.Gauge:
			uniqueGaugeValues[metric.ID] = *metric.Value
		case models.Counter:
			uniqueCounterValues[metric.ID] += *metric.Delta
		}
	}
	for i, metric := range metrics {
		ui := uniqueKeys[metric.ID]
		if ui == i {
			switch metric.MType {
			case models.Gauge:
				*metric.Value = uniqueGaugeValues[metric.ID]
			case models.Counter:
				*metric.Delta = uniqueCounterValues[metric.ID]
			}
			uniqueMetrics = append(uniqueMetrics, metric)
		}
	}

	// Сохранение метрик
	ctx := context.Background()
	err := h.storage.UpdateMetricsBatch(ctx, uniqueMetrics)
	if err != nil {
		log.Printf("Failed to update mectrics after retries: %v", err)
	}

	// Ответ
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`{"status":"ok"}`))
}

func (h *Handlers) valueMetricJSONHandler(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "cannot read body", http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	var m models.Metrics
	if err := json.Unmarshal(body, &m); err != nil {
		http.Error(res, "invalid json", http.StatusBadRequest)
		return
	}

	if m.ID == "" || (m.MType != "gauge" && m.MType != "counter") {
		http.Error(res, "invalid metric id or type", http.StatusBadRequest)
		return
	}

	resp := models.Metrics{ID: m.ID, MType: m.MType}

	switch m.MType {
	case "gauge":
		value, err := h.storage.GetGauge(m.ID)
		if errors.Is(err, storage.ErrMetricNotFound) {
			value = 0
		}
		resp.Value = &value
	case "counter":
		value, err := h.storage.GetCounter(m.ID)
		if errors.Is(err, storage.ErrMetricNotFound) {
			value = 0
		}
		resp.Delta = &value
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(res, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(jsonResp)
}

func (h *Handlers) pingHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/html")

	if h.db == nil {
		http.Error(res, "БД недоступна!", http.StatusInternalServerError)
		return
	}

	if err := h.db.Ping(); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}
