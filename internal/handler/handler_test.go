package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akorablin/yandex-practicum-metrics/internal/config"
	"github.com/akorablin/yandex-practicum-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}

	tests := []struct {
		name string
		url  string
		want want
	}{
		{
			name: "Positive gauge test #1",
			url:  "/update/gauge/TestMetric/123.456",
			want: want{
				code:        200,
				response:    `Ok`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Positive counter test #1",
			url:  "/update/counter/TestMetric/123",
			want: want{
				code:        200,
				response:    `Ok`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Empty gauge test #2",
			url:  "/update/gauge/TestMetric",
			want: want{
				code:        404,
				response:    `Ok`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Negative counter test #2",
			url:  "/update/counter/TestMetric/123.123",
			want: want{
				code:        400,
				response:    `Ok`,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.url, nil)
			request.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			cfg, _ := config.GetServerConfig()
			memStorage := storage.NewMemStorage(cfg)
			h := NewHandlers(memStorage)
			h.updateHandler(w, request)

			res := w.Result()
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
