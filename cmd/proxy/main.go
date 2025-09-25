package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/logger"
	"github.com/andress1014/meli-proxy/internal/metrics"
	"github.com/andress1014/meli-proxy/internal/proxy"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
	"go.uber.org/zap"
)

func main() {
	// Optimizaciones de runtime para 50K RPS
	if gomaxprocs := os.Getenv("GOMAXPROCS"); gomaxprocs != "" {
		if procs, err := strconv.Atoi(gomaxprocs); err == nil {
			runtime.GOMAXPROCS(procs)
		}
	}

	// Configuración
	cfg := config.Load()

	// Logger
	log := logger.New(cfg.LogLevel)
	defer log.Sync()

	log.Info("starting meli-proxy optimized for high load",
		zap.Int("gomaxprocs", runtime.GOMAXPROCS(0)),
		zap.String("version", "1.0.0-optimized"))

	// Métricas
	metricsServer := metrics.NewServer(cfg.MetricsPort)

	// Rate limiter
	rateLimiter, err := ratelimit.NewRedisLimiter(cfg.RedisURL)
	if err != nil {
		log.Error("failed to create rate limiter", zap.Error(err))
		os.Exit(1)
	}
	defer rateLimiter.Close()

	// Proxy server
	proxyServer := proxy.NewServer(cfg, rateLimiter, log)

	// HTTP Server optimizado para alta carga
	mainServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: proxyServer.Handler(),

		// Timeouts optimizados para 50K RPS
		ReadTimeout:       5 * time.Second,  // Reducido para alta velocidad
		WriteTimeout:      10 * time.Second, // Timeout de escritura
		IdleTimeout:       30 * time.Second, // Conexiones idle
		ReadHeaderTimeout: 2 * time.Second,  // Timeout para headers

		// Buffers optimizados
		MaxHeaderBytes: 1 << 16, // 64KB max headers
	}

	// Iniciar servidores
	go func() {
		log.Info("starting metrics server", zap.String("port", cfg.MetricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server failed", zap.Error(err))
		}
	}()

	go func() {
		log.Info("starting main server", zap.String("port", cfg.Port))
		if err := mainServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("main server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info("shutting down servers...")

	// Context con timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown servers
	if err := mainServer.Shutdown(ctx); err != nil {
		log.Error("main server shutdown error", zap.Error(err))
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		log.Error("metrics server shutdown error", zap.Error(err))
	}

	log.Info("servers shutdown complete")
}
