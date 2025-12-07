package handler

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
	"github.com/go-chi/chi"
)

type Handlers struct {
	storage storage.Storage
}

func NewHandlers() *Handlers {
	metricsStorage := storage.NewMemStorage()
	return &Handlers{storage: metricsStorage}
}

func (h *Handlers) GetRoutes() http.Handler {
	r := chi.NewRouter()
	r.Post("/update/*", h.updateHandler)
	r.Get("/value/{type}/{name}", h.valueHandler)
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
		invalidUrlTextError := "Invalid URL format. Expected: /update/{type}/{name}/{value}"
		if parts[0] == "gauge" || parts[0] == "counter" {
			http.Error(res, invalidUrlTextError, http.StatusNotFound)
		} else {
			http.Error(res, invalidUrlTextError, http.StatusBadRequest)
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
                <li><code>POST /update/{type}/{name}/{value}</code> - Update metric</li>
                <li><code>GET /value/{type}/{name}</code> - Get metric value</li>
                <li><code>GET /</code> - This dashboard</li>
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
