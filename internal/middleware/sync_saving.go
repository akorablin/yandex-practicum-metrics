package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/akorablin/yandex-practicum-metrics/internal/repository/file"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func SyncSaving(next http.Handler, file *file.Files) http.Handler {
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
			if err := file.Save(); err != nil {
				log.Printf("Failed to save metrics: %v", err)
			} else {
				log.Println("Metrics saved synchronously")
			}
		}
	})
}
