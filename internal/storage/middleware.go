package storage

import (
	"log"
	"net/http"
	"strings"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (m *MemStorage) SyncMetricSaving(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Запускаем следующий обработчик с оберткой
		next.ServeHTTP(rw, r)

		// Сохраняем после успешного POST запроса к /update
		if r.Method == http.MethodPost &&
			(strings.HasPrefix(r.URL.Path, "/update/") || r.URL.Path == "/update") &&
			rw.statusCode == http.StatusOK {
			if err := m.SaveToFile(); err != nil {
				log.Printf("Failed to save metrics: %v", err)
			} else {
				log.Println("Metrics saved synchronously")
			}
		}
	})
}
