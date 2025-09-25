package ratelimit

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// ClusterConfig para Redis Cluster distribuido
type ClusterConfig struct {
	Addrs              []string      `json:"addrs"`
	Password           string        `json:"password,omitempty"`
	DB                 int           `json:"db,omitempty"`
	MaxRetries         int           `json:"max_retries"`
	MinRetryBackoff    time.Duration `json:"min_retry_backoff"`
	MaxRetryBackoff    time.Duration `json:"max_retry_backoff"`
	DialTimeout        time.Duration `json:"dial_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout"`
	PoolSize           int           `json:"pool_size"`
	MinIdleConns       int           `json:"min_idle_conns"`
	MaxConnAge         time.Duration `json:"max_conn_age"`
	PoolTimeout        time.Duration `json:"pool_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	IdleCheckFrequency time.Duration `json:"idle_check_frequency"`
}

// ClusterLimiter para rate limiting distribuido
type ClusterLimiter struct {
	client *redis.ClusterClient
	script string
	logger *zap.Logger
}

func NewClusterLimiter(config ClusterConfig, logger *zap.Logger) (*ClusterLimiter, error) {
	// Configuración optimizada para 50K RPS distribuido
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    config.Addrs,
		Password: config.Password,

		// Connection pool optimizado para alta carga
		MaxRetries:      getOrDefault(config.MaxRetries, 3),
		MinRetryBackoff: getOrDefaultDuration(config.MinRetryBackoff, 8*time.Millisecond),
		MaxRetryBackoff: getOrDefaultDuration(config.MaxRetryBackoff, 512*time.Millisecond),

		// Timeouts agresivos para baja latencia
		DialTimeout:  getOrDefaultDuration(config.DialTimeout, 1*time.Second),
		ReadTimeout:  getOrDefaultDuration(config.ReadTimeout, 500*time.Millisecond),
		WriteTimeout: getOrDefaultDuration(config.WriteTimeout, 500*time.Millisecond),

		// Pool grande para concurrencia alta
		PoolSize:           getOrDefault(config.PoolSize, 1000), // 1000 conns per cluster node
		MinIdleConns:       getOrDefault(config.MinIdleConns, 100),
		MaxConnAge:         getOrDefaultDuration(config.MaxConnAge, 30*time.Minute),
		PoolTimeout:        getOrDefaultDuration(config.PoolTimeout, 1*time.Second),
		IdleTimeout:        getOrDefaultDuration(config.IdleTimeout, 5*time.Minute),
		IdleCheckFrequency: getOrDefaultDuration(config.IdleCheckFrequency, 1*time.Minute),
	})

	// Test connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	// Lua script para sliding window atomic
	luaScript := `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Limpiar entradas antiguas
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

-- Contar entradas actuales
local current = redis.call('ZCARD', key)

if current < limit then
    -- Agregar nueva entrada
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, math.ceil(window / 1000))
    return {1, limit - current - 1}
else
    return {0, 0}
end
`

	logger.Info("Redis cluster limiter initialized",
		zap.Strings("addrs", config.Addrs),
		zap.Int("pool_size", getOrDefault(config.PoolSize, 1000)))

	return &ClusterLimiter{
		client: rdb,
		script: luaScript,
		logger: logger,
	}, nil
}

func (cl *ClusterLimiter) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (*LimitResult, error) {
	now := time.Now().UnixMilli()
	windowMs := window.Milliseconds()

	// Usar hash tag para garantizar que todas las keys del mismo usuario
	// vayan al mismo shard (importante para rate limiting distribuido)
	clusterKey := cl.addHashTag(key)

	result, err := cl.client.Eval(ctx, cl.script, []string{clusterKey}, windowMs, limit, now).Result()
	if err != nil {
		return nil, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return &LimitResult{
			Allowed:   false,
			Remaining: 0,
			ResetTime: time.Now().Add(window),
		}, nil
	}

	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))

	return &LimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: time.Now().Add(window),
	}, nil
}

// addHashTag asegura que keys relacionadas vayan al mismo shard
func (cl *ClusterLimiter) addHashTag(key string) string {
	// Extraer el identificador base (IP o user) para hash tag
	parts := strings.Split(key, ":")
	if len(parts) > 0 {
		base := parts[0]
		// Usar hash tag de Redis Cluster: {base}:key
		return "{" + base + "}:" + key
	}
	return key
}

// Health check para cluster
func (cl *ClusterLimiter) HealthCheck(ctx context.Context) error {
	return cl.client.Ping(ctx).Err()
}

// Close connections
func (cl *ClusterLimiter) Close() error {
	return cl.client.Close()
}

// Helper functions
func getOrDefault(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

func getOrDefaultDuration(value, defaultValue time.Duration) time.Duration {
	if value == 0 {
		return defaultValue
	}
	return value
}
