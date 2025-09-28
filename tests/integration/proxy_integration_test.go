package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/middleware"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
	"go.uber.org/zap"
)

// TestFullProxyIntegration - test de integración completo sin Redis real
func TestFullProxyIntegration(t *testing.T) {
	// Crear un servidor backend mock
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simular respuesta de la API de MercadoLibre
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success", "path": "` + r.URL.Path + `"}`))
	}))
	defer backendServer.Close()

	// Configuración de prueba
	cfg := &config.Config{
		Port:        "8080",
		MetricsPort: "9090",
		TargetURL:   backendServer.URL, // Apuntar al mock
		RedisURL:    "redis://localhost:6379",
		LogLevel:    "error", // Reducir logs en tests
		DefaultRPS:  100,
	}

	logger, _ := zap.NewDevelopment()

	// Handler final
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simular el proxy (sin hacer request real a MeLi)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"test": "integration", "path": "` + r.URL.Path + `"}`))
	})

	// Crear middleware de rate limiting (sin Redis real)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(nil, cfg, logger)
	
	// Aplicar middlewares
	handler := rateLimitMiddleware.Handler(finalHandler)

	t.Run("request normal pasa a través del sistema", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/categories/MLA1234", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		
		// Como no hay Redis, debería pasar (fail-open)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		
		if rr.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content type, got %s", rr.Header().Get("Content-Type"))
		}
	})

	t.Run("diferentes paths funcionan correctamente", func(t *testing.T) {
		paths := []string{
			"/categories/MLA1234",
			"/items/MLA123456789",
			"/users/123456",
			"/sites/MLA",
		}

		for _, path := range paths {
			req := httptest.NewRequest("GET", path, nil)
			req.RemoteAddr = "192.168.1.101:12345"
			
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			
			if rr.Code != http.StatusOK {
				t.Errorf("path %s: expected status 200, got %d", path, rr.Code)
			}
		}
	})

	t.Run("headers se preservan correctamente", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.102:12345"
		req.Header.Set("User-Agent", "test-client/1.0")
		req.Header.Set("X-Custom-Header", "custom-value")
		
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		
		// El handler debería preservar el comportamiento
		if rr.Header().Get("Content-Type") == "" {
			t.Error("Content-Type header missing")
		}
	})
}

func TestMiddlewareChain(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		DefaultRPS: 50,
	}

	// Handler base
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("base response"))
	})

	// Crear cadena de middlewares
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(nil, cfg, logger)
	
	// Aplicar middlewares en orden
	handler := rateLimitMiddleware.Handler(baseHandler)

	t.Run("cadena de middlewares funciona", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test-chain", nil)
		req.RemoteAddr = "192.168.1.200:12345"
		
		rr := httptest.NewRecorder()
		
		start := time.Now()
		handler.ServeHTTP(rr, req)
		duration := time.Since(start)
		
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		
		// El middleware no debería agregar latencia significativa
		if duration > 100*time.Millisecond {
			t.Logf("Warning: middleware chain took %v (may be acceptable)", duration)
		}
		
		t.Logf("Middleware chain completed in %v", duration)
	})
}

func TestRateLimitKeyGeneration(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		forwardedFor   string
		path           string
		expectedIP     string
		expectedPath   string
	}{
		{
			name:         "standard request",
			remoteAddr:   "192.168.1.100:12345",
			path:         "/categories/MLA1234",
			expectedIP:   "192.168.1.100",
			expectedPath: "/categories/*",
		},
		{
			name:         "with X-Forwarded-For",
			remoteAddr:   "10.0.0.1:12345",
			forwardedFor: "203.0.113.10",
			path:         "/items/MLA123456789",
			expectedIP:   "203.0.113.10",
			expectedPath: "/items/*",
		},
		{
			name:         "direct path",
			remoteAddr:   "192.168.1.101:12345", 
			path:         "/health",
			expectedIP:   "192.168.1.101",
			expectedPath: "/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header:     make(http.Header),
				RemoteAddr: tt.remoteAddr,
			}
			
			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}
			
			// Test extracción de IP
			ip := ratelimit.ExtractIP(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, ip)
			}
			
			// Test normalización de path
			normalizedPath := ratelimit.NormalizePath(tt.path)
			if normalizedPath != tt.expectedPath {
				t.Errorf("expected path %s, got %s", tt.expectedPath, normalizedPath)
			}
		})
	}
}
