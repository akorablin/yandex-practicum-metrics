package main

import (
	"net/http"

	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
)

func main() {
	metricsStorage := storage.NewMemStorage()
	handlers := handler.NewHandlers(metricsStorage)

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handlers.UpdateHandler)

	err := http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		panic(err)
	}
}
