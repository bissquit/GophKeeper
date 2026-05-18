// Package logging provides an HTTP middleware that logs each request using slog
package logging

import (
	"log/slog"
	"net/http"
	"time"
)

type responseData struct {
	statusCode int
	method     string
	size       int
	path       string
	duration   time.Duration
}

type responseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

// Write captures the response size and defaults the status code to 200
// when WriteHeader was never called explicitly
func (w *responseWriter) Write(b []byte) (int, error) {
	if w.responseData.statusCode == 0 {
		w.responseData.statusCode = http.StatusOK
	}
	w.responseData.size += len(b)
	return w.ResponseWriter.Write(b)
}

// WriteHeader records the status code on its first invocation
func (w *responseWriter) WriteHeader(statusCode int) {
	if w.responseData.statusCode == 0 {
		w.responseData.statusCode = statusCode
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

// Logger returns a chi-compatible middleware
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rd := &responseData{
				method: r.Method,
				path:   r.URL.Path,
			}
			rw := &responseWriter{ResponseWriter: w, responseData: rd}
			next.ServeHTTP(rw, r)
			rd.duration = time.Since(start)

			logger.Info("request completed",
				"method", rd.method,
				"path", rd.path,
				"status", rd.statusCode,
				"size", rd.size,
				"duration", rd.duration)
		})
	}
}
