package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/middleware"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
	"go.uber.org/zap"
)

func TestNewRateLimitMiddleware(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 100,
	}
	logger, _ := zap.NewDevelopment()
	limiter := ratelimit.NewDummyLimiter()
	
	middleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	if middleware == nil {
		t.Fatal("NewRateLimitMiddleware returned nil")
	}
}

func TestRateLimitMiddleware_AllowedRequest(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 100,
	}
	logger, _ := zap.NewDevelopment()
	limiter := ratelimit.NewDummyLimiter()
	
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	called := false
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if !called {
		t.Error("Next handler should have been called for allowed request")
	}
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRateLimitMiddleware_BlockedRequest(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 10,
	}
	logger, _ := zap.NewDevelopment()
	
	// Create a limiter that will block after 1 request
	limiter := &mockLimiter{
		shouldAllow: false,
		remaining:   0,
		resetTime:   time.Now().Add(60 * time.Second),
	}
	
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	called := false
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if called {
		t.Error("Next handler should NOT have been called for blocked request")
	}
	
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr.Code)
	}
}

func TestRateLimitMiddleware_ErrorHandling(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 100,
	}
	logger, _ := zap.NewDevelopment()
	
	// Create a limiter that returns errors
	limiter := &mockLimiter{
		shouldError: true,
	}
	
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	called := false
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	// Should fail open - allow request when rate limiter errors
	if !called {
		t.Error("Should fail open - next handler should be called when rate limiter errors")
	}
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 (fail open), got %d", rr.Code)
	}
}

func TestRateLimitMiddleware_Headers(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 100,
	}
	logger, _ := zap.NewDevelopment()
	
	limiter := &mockLimiter{
		shouldAllow: true,
		remaining:   50,
		resetTime:   time.Now().Add(30 * time.Second),
	}
	
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	// Check rate limit headers are present
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Missing X-RateLimit-Limit header")
	}
	
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Missing X-RateLimit-Remaining header")
	}
	
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Missing X-RateLimit-Reset header")
	}
}

func TestRateLimitMiddleware_DifferentPaths(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 100,
		PathRateLimit: map[string]int{
			"/categories/*": 50,
			"/items/*":      200,
		},
	}
	logger, _ := zap.NewDevelopment()
	limiter := ratelimit.NewDummyLimiter()
	
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	tests := []struct {
		path string
	}{
		{"/categories/MLA1234"},
		{"/items/MLA123456789"},
		{"/users/12345"},
	}
	
	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for path %s, got %d", tt.path, rr.Code)
		}
	}
}

func TestRateLimitMiddleware_IPExtraction(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 100,
	}
	logger, _ := zap.NewDevelopment()
	limiter := ratelimit.NewDummyLimiter()
	
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
	}{
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			remoteAddr: "10.0.0.1:12345",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "203.0.113.2"},
			remoteAddr: "10.0.0.1:12345",
		},
		{
			name:       "RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "203.0.113.3:54321",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestRateLimitMiddleware_Concurrency(t *testing.T) {
	cfg := &config.Config{
		DefaultRPS: 1000,
	}
	logger, _ := zap.NewDevelopment()
	limiter := ratelimit.NewDummyLimiter()
	
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, cfg, logger)
	
	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond) // Simulate some processing
		w.WriteHeader(http.StatusOK)
	}))
	
	const numRequests = 100
	results := make(chan int, numRequests)
	
	// Send concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			rr := httptest.NewRecorder()
			
			handler.ServeHTTP(rr, req)
			results <- rr.Code
		}(i)
	}
	
	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		}
	}
	
	// All should succeed with dummy limiter
	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}
}

// Mock limiter for testing
type mockLimiter struct {
	shouldAllow bool
	shouldError bool
	remaining   int
	resetTime   time.Time
}

func (m *mockLimiter) CheckMultipleLimits(ctx context.Context, limits map[string]ratelimit.LimitConfig) (map[string]*ratelimit.LimitResult, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}
	
	results := make(map[string]*ratelimit.LimitResult)
	for key := range limits {
		results[key] = &ratelimit.LimitResult{
			Allowed:   m.shouldAllow,
			Remaining: m.remaining,
			ResetTime: m.resetTime,
		}
	}
	
	return results, nil
}

func (m *mockLimiter) Close() error {
	return nil
}
