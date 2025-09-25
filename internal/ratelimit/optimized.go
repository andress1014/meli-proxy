package ratelimit

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LocalCache para reducir calls a Redis en alta carga
type LocalCache struct {
	data   sync.Map
	ttl    time.Duration
	logger *zap.Logger
}

type cacheEntry struct {
	allowed   bool
	remaining int
	timestamp time.Time
}

func NewLocalCache(ttl time.Duration, logger *zap.Logger) *LocalCache {
	cache := &LocalCache{
		ttl:    ttl,
		logger: logger,
	}

	// Limpieza periódica del cache
	go cache.cleanup()

	return cache
}

func (lc *LocalCache) Get(key string) (bool, int, bool) {
	if val, ok := lc.data.Load(key); ok {
		entry := val.(cacheEntry)
		if time.Since(entry.timestamp) < lc.ttl {
			return entry.allowed, entry.remaining, true
		}
		// Expired, remove it
		lc.data.Delete(key)
	}
	return false, 0, false
}

func (lc *LocalCache) Set(key string, allowed bool, remaining int) {
	lc.data.Store(key, cacheEntry{
		allowed:   allowed,
		remaining: remaining,
		timestamp: time.Now(),
	})
}

func (lc *LocalCache) cleanup() {
	ticker := time.NewTicker(lc.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		lc.data.Range(func(key, value interface{}) bool {
			entry := value.(cacheEntry)
			if now.Sub(entry.timestamp) > lc.ttl {
				lc.data.Delete(key)
			}
			return true
		})
	}
}

// OptimizedRedisLimiter con cache local para alta carga
type OptimizedRedisLimiter struct {
	*RedisLimiter
	localCache *LocalCache
	logger     *zap.Logger
}

func NewOptimizedRedisLimiter(redisURL string, logger *zap.Logger) (*OptimizedRedisLimiter, error) {
	baseLimiter, err := NewRedisLimiter(redisURL)
	if err != nil {
		return nil, err
	}

	// Cache local de 1 segundo para reducir carga en Redis
	localCache := NewLocalCache(1*time.Second, logger)

	return &OptimizedRedisLimiter{
		RedisLimiter: baseLimiter,
		localCache:   localCache,
		logger:       logger,
	}, nil
}

func (orl *OptimizedRedisLimiter) CheckLimitOptimized(ctx context.Context, key string, limit int, window time.Duration) (*LimitResult, error) {
	// Verificar cache local primero (evita Redis)
	if allowed, remaining, found := orl.localCache.Get(key); found {
		return &LimitResult{
			Allowed:   allowed,
			Remaining: remaining,
			ResetTime: time.Now().Add(window),
		}, nil
	}

	// Si no está en cache, ir a Redis
	result, err := orl.CheckLimit(ctx, key, limit, window)
	if err != nil {
		orl.logger.Error("redis check failed", zap.String("key", key), zap.Error(err))
		// Fail open: permitir request si Redis falla (crítico para alta carga)
		return &LimitResult{
			Allowed:   true,
			Remaining: limit - 1,
			ResetTime: time.Now().Add(window),
		}, nil
	}

	// Guardar en cache local
	orl.localCache.Set(key, result.Allowed, result.Remaining)

	return result, nil
}

// Verificar múltiples límites optimizado
func (orl *OptimizedRedisLimiter) CheckMultipleLimitsOptimized(ctx context.Context, limits map[string]LimitConfig) (map[string]*LimitResult, error) {
	results := make(map[string]*LimitResult)

	// Procesar en paralelo para alta carga
	type resultPair struct {
		key    string
		result *LimitResult
		err    error
	}

	resultChan := make(chan resultPair, len(limits))

	// Lanzar goroutines para cada verificación
	for key, config := range limits {
		go func(k string, cfg LimitConfig) {
			result, err := orl.CheckLimitOptimized(ctx, k, cfg.Limit, cfg.Window)
			resultChan <- resultPair{key: k, result: result, err: err}
		}(key, config)
	}

	// Recopilar resultados
	for i := 0; i < len(limits); i++ {
		pair := <-resultChan
		if pair.err != nil {
			orl.logger.Warn("limit check failed",
				zap.String("key", pair.key),
				zap.Error(pair.err))
			// Fail open para alta disponibilidad
			continue
		}
		results[pair.key] = pair.result
	}

	return results, nil
}
