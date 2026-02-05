package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

func CheckHash(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Cannot read body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			computedHash := ""
			if key != "" {
				computedHash = GetHash(body, key)
				w.Header().Set("HashSHA256", computedHash)
			}

			incomingHash := r.Header.Get("HashSHA256")
			if incomingHash != "" && computedHash != "" && !hmac.Equal([]byte(incomingHash), []byte(computedHash)) {
				http.Error(w, "Invalid hash sum", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
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
