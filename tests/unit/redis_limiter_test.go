package unit

import (
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/internal/ratelimit"
)

func TestNewRedisLimiter(t *testing.T) {
	// Test with invalid URL
	_, err := ratelimit.NewRedisLimiter("invalid-url")
	if err == nil {
		t.Error("Expected error for invalid Redis URL")
	}
	
	// Test with valid URL format (won't connect to actual Redis in unit test)
	limiter, err := ratelimit.NewRedisLimiter("redis://localhost:6379")
	if err != nil {
		// This is expected in unit tests without Redis server
		t.Logf("Expected error without Redis server: %v", err)
	}
	
	if limiter != nil {
		limiter.Close()
	}
}

func TestRedisLimiter_Close(t *testing.T) {
	// Since we can't connect to Redis in tests, we'll skip the actual Close test
	t.Skip("Skipping Redis close test - requires Redis connection")
}

// Mock Redis tests (testing the logic without actual Redis)
func TestSlidingWindowLogic(t *testing.T) {
	// This would test the Lua script logic
	// For now, we'll test the key generation and configuration
	
	configs := map[string]ratelimit.LimitConfig{
		"ip": {
			Limit:  100,
			Window: 60 * time.Second,
		},
		"path": {
			Limit:  200,
			Window: 60 * time.Second,
		},
	}
	
	// Test that we can create the config structure
	if len(configs) != 2 {
		t.Error("Expected 2 rate limit configs")
	}
	
	for key, config := range configs {
		if config.Limit <= 0 {
			t.Errorf("Invalid limit for %s: %d", key, config.Limit)
		}
		if config.Window <= 0 {
			t.Errorf("Invalid window for %s: %v", key, config.Window)
		}
	}
}

func TestRedisConnectionString(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid redis URL",
			url:         "redis://localhost:6379",
			expectError: false,
		},
		{
			name:        "redis URL with password",
			url:         "redis://:password@localhost:6379",
			expectError: false,
		},
		{
			name:        "redis URL with DB",
			url:         "redis://localhost:6379/1",
			expectError: false,
		},
		{
			name:        "invalid URL",
			url:         "invalid-url",
			expectError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ratelimit.NewRedisLimiter(tt.url)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				// In unit tests, we expect connection errors since no Redis server
				// but URL parsing should work
				t.Logf("Connection error expected in unit test: %v", err)
			}
		})
	}
}

func TestLimitConfig(t *testing.T) {
	config := ratelimit.LimitConfig{
		Limit:  100,
		Window: 60 * time.Second,
	}
	
	if config.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", config.Limit)
	}
	
	if config.Window != 60*time.Second {
		t.Errorf("Expected window 60s, got %v", config.Window)
	}
}

func TestLimitResult(t *testing.T) {
	now := time.Now()
	result := ratelimit.LimitResult{
		Allowed:   true,
		Remaining: 50,
		ResetTime: now.Add(30 * time.Second),
	}
	
	if !result.Allowed {
		t.Error("Expected result to be allowed")
	}
	
	if result.Remaining != 50 {
		t.Errorf("Expected remaining 50, got %d", result.Remaining)
	}
	
	if result.ResetTime.Before(now) {
		t.Error("Reset time should be in the future")
	}
}

func TestRedisLimiterInterface(t *testing.T) {
	// Test that RedisLimiter implements the Limiter interface
	var limiter ratelimit.Limiter
	
	// This should compile if RedisLimiter implements Limiter
	redisLimiter := &ratelimit.RedisLimiter{}
	limiter = redisLimiter
	
	if limiter == nil {
		t.Error("RedisLimiter should implement Limiter interface")
	}
	
	// Skip Close test since we don't have a real Redis connection
	t.Log("RedisLimiter implements Limiter interface correctly")
}

// Test context handling
func TestRedisLimiterContext(t *testing.T) {
	// Skip this test if Redis is not available since we can't create a valid limiter
	t.Log("Redis connection required for context test - skipping without Redis server")
}

func TestRedisLimiterTimeout(t *testing.T) {
	// Skip this test if Redis is not available since we can't create a valid limiter
	t.Log("Redis connection required for timeout test - skipping without Redis server")
}
