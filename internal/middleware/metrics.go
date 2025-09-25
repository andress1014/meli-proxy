package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/andress1014/meli-proxy/internal/metrics"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
)

type MetricsMiddleware struct{}

func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{}
}

// ResponseWriter wrapper para capturar el status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func (m *MetricsMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := ratelimit.NormalizePath(r.URL.Path)
		method := r.Method

		// Incrementar requests en progreso
		metrics.IncRequestsInProgress(method, path)
		defer metrics.DecRequestsInProgress(method, path)

		// Wrapper para capturar status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Ejecutar request
		next.ServeHTTP(rw, r)

		// Registrar métricas
		duration := time.Since(start)
		status := strconv.Itoa(rw.statusCode)

		metrics.RecordRequest(method, path, status, duration)
	})
}
