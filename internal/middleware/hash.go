package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

type newResponseWriter struct {
	http.ResponseWriter
	body   []byte
	status int
}

func (w *newResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *newResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func CheckHash(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Проверяем подпись тела запроса
			got := r.Header.Get("HashSHA256")
			if got != "" {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Cannot read body", http.StatusBadRequest)
					return
				}
				_ = r.Body.Close()
				/*
					computed := GetHash(body, key)
					if !strings.EqualFold(got, computed) {
							http.Error(w, "Invalid hash sum", http.StatusBadRequest)
							return
					}
				*/
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			// Создаем подпись тела ответа
			rw := &newResponseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r)
			if len(rw.body) > 0 {
				hash := GetHash(rw.body, key)
				rw.ResponseWriter.Header().Set("HashSHA256", hash)
			}
		})
	}
}

func GetHash(data []byte, key string) string {
	if key == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
