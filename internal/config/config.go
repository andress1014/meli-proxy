package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	MetricsPort string
	TargetURL   string
	RedisURL    string
	LogLevel    string
	RedisEnabled bool

	// Rate limiting configuration
	DefaultRPS      int
	IPRateLimit     map[string]int
	PathRateLimit   map[string]int
	IPPathRateLimit map[string]int
}

func Load() *Config {
	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		MetricsPort:  getEnv("METRICS_PORT", "9090"),
		TargetURL:    getEnv("TARGET_URL", "https://api.mercadolibre.com"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		RedisEnabled: getEnvBool("REDIS_ENABLED", true),
		DefaultRPS:   getEnvInt("DEFAULT_RPS", 100),
	}

	// Cargar configuraciones de rate limiting desde variables de entorno
	cfg.IPRateLimit = parseRateLimitMap(getEnv("IP_RATE_LIMITS", ""))
	cfg.PathRateLimit = parseRateLimitMap(getEnv("PATH_RATE_LIMITS", ""))
	cfg.IPPathRateLimit = parseRateLimitMap(getEnv("IP_PATH_RATE_LIMITS", ""))

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// parseRateLimitMap parsea strings como "key1:100,key2:200"
func parseRateLimitMap(input string) map[string]int {
	result := make(map[string]int)
	if input == "" {
		return result
	}

	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		kv := strings.Split(strings.TrimSpace(pair), ":")
		if len(kv) == 2 {
			if limit, err := strconv.Atoi(strings.TrimSpace(kv[1])); err == nil {
				result[strings.TrimSpace(kv[0])] = limit
			}
		}
	}
	return result
}
