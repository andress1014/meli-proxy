package ratelimit

import (
	"context"
)

// Limiter es la interfaz común para todos los rate limiters
type Limiter interface {
	CheckMultipleLimits(ctx context.Context, configs map[string]LimitConfig) (map[string]*LimitResult, error)
	Close() error
}

// Asegurar que RedisLimiter implementa la interfaz
var _ Limiter = (*RedisLimiter)(nil)

// Asegurar que DummyLimiter implementa la interfaz  
var _ Limiter = (*DummyLimiter)(nil)
