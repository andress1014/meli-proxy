package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/middleware"
	"github.com/andress1014/meli-proxy/internal/metrics"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
)

func TestRateLimitMiddleware(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	// Create basic config for testing
	cfg := &config.Config{
		DefaultRPS: 100,
	}
	
	// Test with DummyLimiter
	dummyLimiter := ratelimit.NewDummyLimiter()
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(dummyLimiter, cfg, logger)
	
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMetricsMiddleware(t *testing.T) {
	metricsMiddleware := middleware.NewMetricsMiddleware()
	
	handler := metricsMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMetricsMiddlewareError(t *testing.T) {
	metricsMiddleware := middleware.NewMetricsMiddleware()
	
	handler := metricsMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	}))

	req := httptest.NewRequest("POST", "/error", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestMiddlewareWithHeaders(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	cfg := &config.Config{
		DefaultRPS: 100,
	}
	
	dummyLimiter := ratelimit.NewDummyLimiter()
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(dummyLimiter, cfg, logger)
	
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"X-Real-IP", "X-Real-IP", "192.168.1.1"},
		{"X-Forwarded-For", "X-Forwarded-For", "192.168.1.1, 10.0.0.1"},
		{"RemoteAddr", "", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if test.header != "" {
				req.Header.Set(test.header, test.value)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
		})
	}
}

func TestOptimizedMiddlewareRedisUnavailable(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	// Create components for OptimizedMiddleware
	asyncCollector := metrics.NewAsyncCollector(logger)
	defer asyncCollector.Shutdown()
	
	// Try to create OptimizedRedisLimiter (this will fail if Redis is not available)
	config := ratelimit.ClusterConfig{
		Addrs: []string{"localhost:6379"},
	}
	optimizedLimiter, err := ratelimit.NewClusterLimiter(config, logger)
	if err != nil {
		t.Logf("Redis not available (expected), skipping optimized middleware test: %v", err)
		return
	}
	
	// Create a mock OptimizedRedisLimiter from ClusterLimiter 
	// This is just to test the middleware structure, not the actual functionality
	optimizedMiddleware := middleware.NewOptimizedMiddleware(nil, asyncCollector, logger)
	
	handler := optimizedMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// With nil limiter, it should still pass through
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	_ = optimizedLimiter // Use the variable to avoid unused variable error
}
