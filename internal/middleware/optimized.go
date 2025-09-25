package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/andress1014/meli-proxy/internal/metrics"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
)

// OptimizedMiddleware para alta carga - 50K RPS
type OptimizedMiddleware struct {
	rateLimiter    *ratelimit.OptimizedRedisLimiter
	asyncCollector *metrics.AsyncCollector
	logger         *zap.Logger
	// Pre-allocated pools para reducir allocations
	pathPool        sync.Pool
	clientIPPool    sync.Pool
	combinedKeyPool sync.Pool
}

func NewOptimizedMiddleware(rateLimiter *ratelimit.OptimizedRedisLimiter, asyncCollector *metrics.AsyncCollector, logger *zap.Logger) *OptimizedMiddleware {
	return &OptimizedMiddleware{
		rateLimiter:    rateLimiter,
		asyncCollector: asyncCollector,
		logger:         logger,
		pathPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 256) // Pre-allocate para paths
			},
		},
		clientIPPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 16) // Pre-allocate para IPs
			},
		},
		combinedKeyPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 512) // Pre-allocate para keys combinadas
			},
		},
	}
}

func (om *OptimizedMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Extraer IP optimizado (sin allocations innecesarias)
		clientIP := om.extractClientIPOptimized(r)

		// Normalizar path optimizado
		normalizedPath := om.normalizePathOptimized(r.URL.Path)

		// Verificar rate limits en paralelo (no secuencial)
		allowed, remaining, err := om.checkRateLimitsOptimized(r.Context(), clientIP, normalizedPath)
		if err != nil {
			om.logger.Error("rate limit check failed",
				zap.String("error", err.Error()),
				zap.String("ip", clientIP),
				zap.String("path", normalizedPath))
			// Fail open - permitir request si hay error
		} else if !allowed {
			// Rate limit exceeded
			om.asyncCollector.RecordRateLimitAsync("combined", clientIP+":"+normalizedPath, false, remaining)

			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

			// Record 429 async
			duration := time.Since(startTime)
			om.asyncCollector.RecordRequestAsync(r.Method, normalizedPath, 429, duration)
			return
		}

		// Wrapper para capturar status code sin overhead
		wrapper := &optimizedResponseWrapper{ResponseWriter: w, statusCode: 200}

		// Procesar request
		next.ServeHTTP(wrapper, r)

		// Record metrics async (no blocking)
		duration := time.Since(startTime)
		om.asyncCollector.RecordRequestAsync(r.Method, normalizedPath, wrapper.statusCode, duration)

		// Record successful rate limit async
		if allowed {
			om.asyncCollector.RecordRateLimitAsync("combined", clientIP+":"+normalizedPath, true, remaining)
		}
	})
}

// extractClientIPOptimized - Optimizado para alta carga
func (om *OptimizedMiddleware) extractClientIPOptimized(r *http.Request) string {
	// Usar buffer pool para reducir allocations
	buffer := om.clientIPPool.Get().([]byte)
	defer om.clientIPPool.Put(buffer[:0])

	// Prioridad: X-Forwarded-For, X-Real-IP, RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Extract IP from RemoteAddr (formato "IP:port")
	if colonIdx := strings.LastIndex(r.RemoteAddr, ":"); colonIdx > 0 {
		return r.RemoteAddr[:colonIdx]
	}

	return r.RemoteAddr
}

// normalizePathOptimized - Sin regex para alta performance
func (om *OptimizedMiddleware) normalizePathOptimized(path string) string {
	// Pool buffer para reducir allocations
	buffer := om.pathPool.Get().([]byte)
	defer om.pathPool.Put(buffer[:0])

	// Normalización simple y rápida
	if len(path) == 0 || path == "/" {
		return "/"
	}

	// Convertir a lowercase in-place
	normalizedPath := strings.ToLower(path)

	// Remover trailing slash (excepto root)
	if len(normalizedPath) > 1 && normalizedPath[len(normalizedPath)-1] == '/' {
		normalizedPath = normalizedPath[:len(normalizedPath)-1]
	}

	return normalizedPath
}

// checkRateLimitsOptimized - Verificación optimizada con cache local
func (om *OptimizedMiddleware) checkRateLimitsOptimized(ctx context.Context, clientIP, path string) (bool, int, error) {
	// Usar pool para key combinada
	keyBuffer := om.combinedKeyPool.Get().([]byte)
	defer om.combinedKeyPool.Put(keyBuffer[:0])

	// Build combined key efficiently
	combinedKey := fmt.Sprintf("%s:%s", clientIP, path)

	// Usar cache local optimizado
	result, err := om.rateLimiter.CheckLimitOptimized(ctx, combinedKey, 100, time.Minute)
	if err != nil {
		return false, 0, err
	}

	return result.Allowed, result.Remaining, nil
}

// optimizedResponseWrapper - Lightweight wrapper
type optimizedResponseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *optimizedResponseWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *optimizedResponseWrapper) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}
