package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisLimiter struct {
	client *redis.Client
	script *redis.Script
}

// Script Lua para sliding window atómico
const slidingWindowScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Limpiar registros antiguos
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window * 1000)

-- Contar requests actuales
local current = redis.call('ZCARD', key)

if current < limit then
    -- Agregar el request actual
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, window + 1)
    return {1, limit - current - 1}
else
    return {0, 0}
end
`

func NewRedisLimiter(redisURL string) (*RedisLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisLimiter{
		client: client,
		script: redis.NewScript(slidingWindowScript),
	}, nil
}

func (rl *RedisLimiter) Close() error {
	return rl.client.Close()
}

type LimitResult struct {
	Allowed   bool
	Remaining int
	ResetTime time.Time
}

func (rl *RedisLimiter) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (*LimitResult, error) {
	now := time.Now().UnixMilli()
	windowSeconds := int(window.Seconds())

	result, err := rl.script.Run(ctx, rl.client, []string{key}, windowSeconds, limit, now).Result()
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	results, ok := result.([]interface{})
	if !ok || len(results) != 2 {
		return nil, fmt.Errorf("unexpected redis script result")
	}

	allowed, ok := results[0].(int64)
	if !ok {
		return nil, fmt.Errorf("invalid allowed result from redis")
	}

	remaining, ok := results[1].(int64)
	if !ok {
		return nil, fmt.Errorf("invalid remaining result from redis")
	}

	resetTime := time.Now().Add(window)

	return &LimitResult{
		Allowed:   allowed == 1,
		Remaining: int(remaining),
		ResetTime: resetTime,
	}, nil
}

// Método para múltiples límites (IP, Path, IP+Path)
func (rl *RedisLimiter) CheckMultipleLimits(ctx context.Context, limits map[string]LimitConfig) (map[string]*LimitResult, error) {
	results := make(map[string]*LimitResult)

	for key, config := range limits {
		result, err := rl.CheckLimit(ctx, key, config.Limit, config.Window)
		if err != nil {
			return nil, fmt.Errorf("failed to check limit for key %s: %w", key, err)
		}
		results[key] = result
	}

	return results, nil
}

type LimitConfig struct {
	Limit  int
	Window time.Duration
}

// Funciones helper para generar keys
func IPKey(ip string) string {
	return fmt.Sprintf("ip::%s", ip)
}

func PathKey(path string) string {
	return fmt.Sprintf("path::%s", path)
}

func IPPathKey(ip, path string) string {
	return fmt.Sprintf("ip_path::%s::%s", ip, path)
}
