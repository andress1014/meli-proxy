package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/middleware"
	"github.com/andress1014/meli-proxy/pkg/httpclient"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.Load()
	
	// Override port for test server
	cfg.Port = "8081"
	
	// Initialize logger
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()

	zapLogger.Info("Starting meli-proxy test server (NO RATE LIMITING)", 
		zap.String("port", cfg.Port),
		zap.String("target_url", cfg.TargetURL))

	// Parse target URL
	targetURL, err := url.Parse(cfg.TargetURL)
	if err != nil {
		zapLogger.Fatal("invalid target URL", zap.Error(err))
	}

	// Create optimized HTTP client
	client := httpclient.NewOptimizedClient()

	// Create reverse proxy (sin rate limiting)
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host

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
			zapLogger.Error("proxy error",
				zap.Error(err),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method))

			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"service_unavailable","message":"Upstream service error"}`))
		},
	}

	// Solo middlewares básicos (sin rate limiting)
	handler := http.Handler(proxy)
	handler = middleware.NewMetricsMiddleware().Handler(handler)

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		zapLogger.Info("Test server starting", zap.String("address", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		zapLogger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("Server exited")
}
