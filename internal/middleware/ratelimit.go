package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/andress1014/meli-proxy/internal/config"
	"github.com/andress1014/meli-proxy/internal/metrics"
	"github.com/andress1014/meli-proxy/internal/ratelimit"
	"go.uber.org/zap"
)

type RateLimitMiddleware struct {
	limiter ratelimit.Limiter
	config  *config.Config
	logger  *zap.Logger
}

func NewRateLimitMiddleware(limiter ratelimit.Limiter, cfg *config.Config, logger *zap.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		config:  cfg,
		logger:  logger,
	}
}

func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		// Obtener keys para rate limiting
		keys := ratelimit.GetLimitKeys(r)
		ip := ratelimit.ExtractIP(r)
		path := ratelimit.NormalizePath(r.URL.Path)

		// Configurar límites
		limits := m.buildLimitConfigs(keys, ip, path)

		// Verificar límites
		results, err := m.limiter.CheckMultipleLimits(ctx, limits)
		if err != nil {
			m.logger.Error("rate limit check failed",
				zap.Error(err),
				zap.String("ip", ip),
				zap.String("path", path))
			// En caso de error, permitir el request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		// Verificar si algún límite fue excedido
		for limitType, result := range results {
			if !result.Allowed {
				// Registrar métrica de bloqueo
				metrics.RecordRateLimitBlocked(limitType, keys[limitType])

				// Log del bloqueo
				m.logger.Warn("rate limit exceeded",
					zap.String("limit_type", limitType),
					zap.String("key", keys[limitType]),
					zap.String("ip", ip),
					zap.String("path", path))

				// Responder con 429
				m.writeRateLimitResponse(w, result)
				return
			}
		}

		// Agregar headers informativos
		m.addRateLimitHeaders(w, results)

		// Continuar con el próximo handler
		next.ServeHTTP(w, r)
	})
}

func (m *RateLimitMiddleware) buildLimitConfigs(keys map[string]string, ip, path string) map[string]ratelimit.LimitConfig {
	window := 60 * time.Second // 1 minuto por defecto
	limits := make(map[string]ratelimit.LimitConfig)

	// Límite por IP
	ipLimit := m.config.DefaultRPS
	if customLimit, exists := m.config.IPRateLimit[ip]; exists {
		ipLimit = customLimit
	}
	limits["ip"] = ratelimit.LimitConfig{Limit: ipLimit, Window: window}

	// Límite por Path
	pathLimit := m.config.DefaultRPS
	if customLimit, exists := m.config.PathRateLimit[path]; exists {
		pathLimit = customLimit
	}
	limits["path"] = ratelimit.LimitConfig{Limit: pathLimit, Window: window}

	// Límite por IP+Path
	ipPathKey := ip + "::" + path
	ipPathLimit := m.config.DefaultRPS / 2 // Más restrictivo
	if customLimit, exists := m.config.IPPathRateLimit[ipPathKey]; exists {
		ipPathLimit = customLimit
	}
	limits["ip_path"] = ratelimit.LimitConfig{Limit: ipPathLimit, Window: window}

	return limits
}

func (m *RateLimitMiddleware) writeRateLimitResponse(w http.ResponseWriter, result *ratelimit.LimitResult) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
	w.WriteHeader(http.StatusTooManyRequests)

	response := `{"error":"rate_limit_exceeded","message":"Too many requests"}`
	w.Write([]byte(response))
}

func (m *RateLimitMiddleware) addRateLimitHeaders(w http.ResponseWriter, results map[string]*ratelimit.LimitResult) {
	// Usar el límite más restrictivo para los headers
	minRemaining := -1
	var earliestReset time.Time

	for _, result := range results {
		if minRemaining == -1 || result.Remaining < minRemaining {
			minRemaining = result.Remaining
		}
		if earliestReset.IsZero() || result.ResetTime.Before(earliestReset) {
			earliestReset = result.ResetTime
		}
	}

	// Add standard rate limit headers
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(m.config.DefaultRPS))
	if minRemaining >= 0 {
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(minRemaining))
	}
	if !earliestReset.IsZero() {
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(earliestReset.Unix(), 10))
	}
}
