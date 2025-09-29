package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/metrics"
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
	client     *http.Client
}

func NewServer(cfg *config.Config, rateLimiter ratelimit.Limiter, logger *zap.Logger) *Server {
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
		client:     client,
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
			"status":     "healthy",
			"service":    "meli-proxy",
			"version":    "v1.5.1", // Testing webhook only!
			"uptime":     uptime.String(),
			"target_url": s.config.TargetURL,
			"system": map[string]interface{}{
				"goroutines": runtime.NumGoroutine(),
				"memory_mb":  m.Alloc / 1024 / 1024,
				"gc_cycles":  m.NumGC,
				"cpu_count":  runtime.NumCPU(),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"deployment": map[string]interface{}{
				"environment": "production",
				"region":      "sfo3",
				"instances":   4,
			},
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

		// Handle status endpoint (simplified health check)
		if r.URL.Path == "/status" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			statusInfo := map[string]interface{}{
				"status":  "ok",
				"version": "v1.5.1",
				"uptime":  time.Since(s.startTime).String(),
				"service": "meli-proxy",
			}
			json.NewEncoder(w).Encode(statusInfo)
			return
		}

		// Handle no-rate-limit routes
		if s.isNoRateLimitRoute(r.URL.Path) {
			s.ServeNoRateLimit(w, r)
			return
		}

		// Handle everything else through proxy with rate limiting
		s.ServeHTTP(w, r)
	})
}

// isNoRateLimitRoute checks if the request path is a no-rate-limit route
func (s *Server) isNoRateLimitRoute(path string) bool {
	return strings.HasPrefix(path, "/no-ratelimit/")
}

// ServeNoRateLimit handles requests without rate limiting
func (s *Server) ServeNoRateLimit(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Remove the no-ratelimit prefix
	originalPath := strings.TrimPrefix(r.URL.Path, "/no-ratelimit")

	// Create a new request with the modified path
	targetURL := s.config.TargetURL + originalPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create proxy request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		s.logger.Error("Error creating proxy request",
			zap.Error(err),
			zap.String("target_url", targetURL))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		// Record metrics for error
		duration := time.Since(startTime)
		s.recordMetrics(r.Method, "/no-ratelimit/*", "500", duration)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set X-Forwarded headers
	clientIP := r.RemoteAddr
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		clientIP = ip
	} else if ip := r.Header.Get("X-Real-IP"); ip != "" {
		clientIP = ip
	}
	proxyReq.Header.Set("X-Forwarded-For", clientIP)
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)
	proxyReq.Header.Set("X-Forwarded-Proto", "http")

	// Execute request
	resp, err := s.client.Do(proxyReq)
	if err != nil {
		s.logger.Error("Error executing proxy request",
			zap.Error(err),
			zap.String("target_url", targetURL))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)

		// Record metrics for error
		duration := time.Since(startTime)
		s.recordMetrics(r.Method, "/no-ratelimit/*", "502", duration)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set response status
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		s.logger.Error("Error copying response body", zap.Error(err))
	}

	// Record metrics for successful request
	duration := time.Since(startTime)
	statusCode := resp.StatusCode
	s.recordMetrics(r.Method, "/no-ratelimit/*", strconv.Itoa(statusCode), duration)

	// Log request without rate limit info
	s.logger.Info("No-rate-limit request served",
		zap.String("method", r.Method),
		zap.String("original_path", originalPath),
		zap.String("target_url", targetURL),
		zap.Int("status", resp.StatusCode),
		zap.String("client_ip", clientIP),
		zap.Duration("duration", duration),
	)
}

// recordMetrics registers metrics for a request
func (s *Server) recordMetrics(method, path, statusStr string, duration time.Duration) {
	metrics.RecordRequest(method, path, statusStr, duration)
}