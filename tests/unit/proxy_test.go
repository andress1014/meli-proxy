package unit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/proxy"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
	"go.uber.org/zap"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Port:        "8080",
		MetricsPort: "9090",
		TargetURL:   "https://api.mercadolibre.com",
		RedisURL:    "redis://localhost:6379",
		LogLevel:    "error",
		DefaultRPS:  100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestNewServerInvalidURL(t *testing.T) {
	// This test would cause Fatal() in real code, 
	// but we can't test Fatal() directly as it calls os.Exit()
	// Instead, let's test with a valid but unusual URL
	cfg := &config.Config{
		TargetURL: "http://localhost:99999", // Valid format but unlikely to work
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	
	// This should not panic, just create the server
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	if server == nil {
		t.Error("NewServer should not return nil for valid URL format")
	}
}

func TestProxyRequest(t *testing.T) {
	// Mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "success",
			"path":    r.URL.Path,
			"method":  r.Method,
		})
	}))
	defer backend.Close()
	
	cfg := &config.Config{
		Port:        "8080",
		MetricsPort: "9090",
		TargetURL:   backend.URL,
		LogLevel:    "error",
		DefaultRPS:  100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	// Test GET request
	req := httptest.NewRequest("GET", "/categories/MLA1234", nil)
	req.Header.Set("User-Agent", "test-client")
	rr := httptest.NewRecorder()
	
	server.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if response["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", response["message"])
	}
}

func TestProxyPOSTRequest(t *testing.T) {
	// Mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"received": string(body),
			"method":   r.Method,
		})
	}))
	defer backend.Close()
	
	cfg := &config.Config{
		TargetURL:  backend.URL,
		LogLevel:   "error",
		DefaultRPS: 100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	// Test POST request with body
	payload := `{"test": "data"}`
	req := httptest.NewRequest("POST", "/items", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	
	server.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rr.Code)
	}
}

func TestProxyHeaders(t *testing.T) {
	// Mock backend server that echoes headers
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		headers := make(map[string]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"headers": headers,
		})
	}))
	defer backend.Close()
	
	cfg := &config.Config{
		TargetURL:  backend.URL,
		LogLevel:   "error",
		DefaultRPS: 100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("Authorization", "Bearer token123")
	rr := httptest.NewRecorder()
	
	server.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	headers := response["headers"].(map[string]interface{})
	
	if headers["X-Custom-Header"] != "custom-value" {
		t.Error("Custom header not forwarded correctly")
	}
}

func TestProxyErrorHandling(t *testing.T) {
	// Mock backend server that returns errors
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "error") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
		if strings.Contains(r.URL.Path, "notfound") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	
	cfg := &config.Config{
		TargetURL:  backend.URL,
		LogLevel:   "error",
		DefaultRPS: 100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	// Test 500 error
	req := httptest.NewRequest("GET", "/error", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
	
	// Test 404 error
	req = httptest.NewRequest("GET", "/notfound", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestProxyTimeout(t *testing.T) {
	// Mock slow backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "slow") {
			time.Sleep(100 * time.Millisecond) // Simulate slow response
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer backend.Close()
	
	cfg := &config.Config{
		TargetURL:  backend.URL,
		LogLevel:   "error",
		DefaultRPS: 100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	// Test normal request (should work)
	req := httptest.NewRequest("GET", "/normal", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	// Test slow request (should still work with our current timeout settings)
	req = httptest.NewRequest("GET", "/slow", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for slow request, got %d", rr.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		TargetURL:  "https://api.mercadolibre.com",
		LogLevel:   "error",
		DefaultRPS: 100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	
	// Use the Handler() method which routes to health endpoints
	handler := server.Handler()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}
	
	if response["status"] != "healthy" {
		t.Error("Expected healthy status")
	}
}

func TestStatsEndpoint(t *testing.T) {
	cfg := &config.Config{
		TargetURL:  "https://api.mercadolibre.com",
		LogLevel:   "error",
		DefaultRPS: 100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()
	
	// Use the Handler() method which routes to status endpoints
	handler := server.Handler()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse stats response: %v", err)
	}
	
	// Should have uptime and other stats
	if _, exists := response["uptime"]; !exists {
		t.Error("Expected uptime in stats response")
	}
}

func TestProxyWithContext(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer backend.Close()
	
	cfg := &config.Config{
		TargetURL:  backend.URL,
		LogLevel:   "error",
		DefaultRPS: 100,
	}
	
	logger, _ := zap.NewDevelopment()
	rateLimiter := ratelimit.NewDummyLimiter()
	server := proxy.NewServer(cfg, rateLimiter, logger)
	
	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	
	server.ServeHTTP(rr, req)
	
	// Should handle canceled context gracefully
	// The exact behavior depends on implementation, but shouldn't crash
}
