package ratelimit

import (
	"context"
	"time"
)

// DummyLimiter es un rate limiter que permite todo para pruebas de carga
type DummyLimiter struct{}

// NewDummyLimiter crea un nuevo dummy limiter
func NewDummyLimiter() *DummyLimiter {
	return &DummyLimiter{}
}

// CheckLimit siempre permite el request
func (d *DummyLimiter) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (*LimitResult, error) {
	return &LimitResult{
		Allowed:   true,
		Remaining: 999999,
		ResetTime: time.Now().Add(time.Hour),
	}, nil
}

// CheckMultipleLimits siempre permite todos los requests
func (d *DummyLimiter) CheckMultipleLimits(ctx context.Context, limits map[string]LimitConfig) (map[string]*LimitResult, error) {
	results := make(map[string]*LimitResult)
	for key := range limits {
		results[key] = &LimitResult{
			Allowed:   true,
			Remaining: 999999,
			ResetTime: time.Now().Add(time.Hour),
		}
	}
	return results, nil
}

// Close no hace nada
func (d *DummyLimiter) Close() error {
	return nil
}
