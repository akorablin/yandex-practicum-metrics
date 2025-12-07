package main

import (
	"net/http"

	"github.com/akorablin/yandex-practicum-metrics/internal/handler"
)

func main() {
	handlers := handler.NewHandlers()
	err := http.ListenAndServe("localhost:8080", handlers.GetRoutes())
	if err != nil {
		panic(err)
	}
}
