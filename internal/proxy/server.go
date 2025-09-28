package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/middleware"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
	"github.com/andress1014/meli-proxy/pkg/httpclient"
	"go.uber.org/zap"
)

type Server struct {
	proxy      *httputil.ReverseProxy
	config     *config.Config
	logger     *zap.Logger
	middleware []func(http.Handler) http.Handler
	startTime  time.Time
}

func NewServer(cfg *config.Config, rateLimiter *ratelimit.RedisLimiter, logger *zap.Logger) *Server {
	// Parse target URL
	targetURL, err := url.Parse(cfg.TargetURL)
	if err != nil {
		logger.Fatal("invalid target URL", zap.Error(err))
	}

	// Create optimized HTTP client
	client := httpclient.NewOptimizedClient()

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host

			// Preserve original path and query
			// req.URL.Path already contains the path

			// Add X-Forwarded headers
			if req.Header.Get("X-Forwarded-Proto") == "" {
				req.Header.Set("X-Forwarded-Proto", "https")
			}
			if req.Header.Get("X-Forwarded-Host") == "" {
				req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			}
		},
		Transport: client.Transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("proxy error",
				zap.Error(err),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method))

			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"service_unavailable","message":"Upstream service error"}`))
		},
		ModifyResponse: func(resp *http.Response) error {
			// NO modificar Location headers para evitar redirects
			// NO usar cache
			resp.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			resp.Header.Set("Pragma", "no-cache")
			resp.Header.Set("Expires", "0")

			// Agregar headers de identificación del proxy
			resp.Header.Set("X-Proxy-By", "meli-proxy")

			return nil
		},
	}

	// Setup middleware chain
	middlewares := []func(http.Handler) http.Handler{
		middleware.NewMetricsMiddleware().Handler,
		middleware.NewRateLimitMiddleware(rateLimiter, cfg, logger).Handler,
	}

	return &Server{
		proxy:      proxy,
		config:     cfg,
		logger:     logger,
		middleware: middlewares,
		startTime:  time.Now(),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	s.logger.Info("incoming request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("query", r.URL.RawQuery),
		zap.String("ip", ratelimit.ExtractIP(r)),
		zap.String("user_agent", r.Header.Get("User-Agent")))

	// Apply middleware chain
	handler := http.Handler(s.proxy)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	handler.ServeHTTP(w, r)
}

// Health check endpoint
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" && r.Method == "GET" {
		// Collect system stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		uptime := time.Since(s.startTime)
		
		healthInfo := map[string]interface{}{
			"status":      "healthy",
			"service":     "meli-proxy",
			"version":     "v1.2.0",
			"uptime":      uptime.String(),
			"target_url":  s.config.TargetURL,
			"system": map[string]interface{}{
				"goroutines": runtime.NumGoroutine(),
				"memory_mb":  m.Alloc / 1024 / 1024,
				"gc_cycles":  m.NumGC,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthInfo)
		return
	}

	// For any other path, use the proxy
	s.ServeHTTP(w, r)
}

// Wrapper to handle both health checks and proxy
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle health check
		if r.URL.Path == "/health" && r.Method == "GET" {
			s.HealthHandler(w, r)
			return
		}

		// Handle everything else through proxy
		s.ServeHTTP(w, r)
	})
}
