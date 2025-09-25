package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Guardar valores originales
	originalEnvVars := map[string]string{
		"PORT":                os.Getenv("PORT"),
		"METRICS_PORT":        os.Getenv("METRICS_PORT"),
		"TARGET_URL":          os.Getenv("TARGET_URL"),
		"REDIS_URL":           os.Getenv("REDIS_URL"),
		"LOG_LEVEL":           os.Getenv("LOG_LEVEL"),
		"DEFAULT_RPS":         os.Getenv("DEFAULT_RPS"),
		"IP_RATE_LIMITS":      os.Getenv("IP_RATE_LIMITS"),
		"PATH_RATE_LIMITS":    os.Getenv("PATH_RATE_LIMITS"),
		"IP_PATH_RATE_LIMITS": os.Getenv("IP_PATH_RATE_LIMITS"),
	}

	// Limpiar variables de entorno
	for key := range originalEnvVars {
		os.Unsetenv(key)
	}

	// Restaurar al final del test
	defer func() {
		for key, value := range originalEnvVars {
			if value != "" {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("default values", func(t *testing.T) {
		cfg := Load()

		if cfg.Port != "8080" {
			t.Errorf("expected Port to be '8080', got '%s'", cfg.Port)
		}
		if cfg.MetricsPort != "9090" {
			t.Errorf("expected MetricsPort to be '9090', got '%s'", cfg.MetricsPort)
		}
		if cfg.TargetURL != "https://api.mercadolibre.com" {
			t.Errorf("expected TargetURL to be 'https://api.mercadolibre.com', got '%s'", cfg.TargetURL)
		}
		if cfg.DefaultRPS != 100 {
			t.Errorf("expected DefaultRPS to be 100, got %d", cfg.DefaultRPS)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		os.Setenv("PORT", "3000")
		os.Setenv("METRICS_PORT", "9091")
		os.Setenv("TARGET_URL", "https://custom.api.com")
		os.Setenv("DEFAULT_RPS", "200")
		os.Setenv("IP_RATE_LIMITS", "192.168.1.1:300,10.0.0.1:150")
		os.Setenv("PATH_RATE_LIMITS", "/categories/*:500,/items/*:400")

		cfg := Load()

		if cfg.Port != "3000" {
			t.Errorf("expected Port to be '3000', got '%s'", cfg.Port)
		}
		if cfg.MetricsPort != "9091" {
			t.Errorf("expected MetricsPort to be '9091', got '%s'", cfg.MetricsPort)
		}
		if cfg.TargetURL != "https://custom.api.com" {
			t.Errorf("expected TargetURL to be 'https://custom.api.com', got '%s'", cfg.TargetURL)
		}
		if cfg.DefaultRPS != 200 {
			t.Errorf("expected DefaultRPS to be 200, got %d", cfg.DefaultRPS)
		}

		// Test rate limit maps
		if cfg.IPRateLimit["192.168.1.1"] != 300 {
			t.Errorf("expected IP rate limit for 192.168.1.1 to be 300, got %d", cfg.IPRateLimit["192.168.1.1"])
		}
		if cfg.PathRateLimit["/categories/*"] != 500 {
			t.Errorf("expected Path rate limit for /categories/* to be 500, got %d", cfg.PathRateLimit["/categories/*"])
		}
	})
}

func TestParseRateLimitMap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]int{},
		},
		{
			name:  "single pair",
			input: "key1:100",
			expected: map[string]int{
				"key1": 100,
			},
		},
		{
			name:  "multiple pairs",
			input: "key1:100,key2:200,key3:300",
			expected: map[string]int{
				"key1": 100,
				"key2": 200,
				"key3": 300,
			},
		},
		{
			name:  "with spaces",
			input: " key1 : 100 , key2 : 200 ",
			expected: map[string]int{
				"key1": 100,
				"key2": 200,
			},
		},
		{
			name:  "invalid format",
			input: "invalid,key2:200",
			expected: map[string]int{
				"key2": 200,
			},
		},
		{
			name:  "invalid number",
			input: "key1:invalid,key2:200",
			expected: map[string]int{
				"key2": 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRateLimitMap(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected key '%s' to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected value for key '%s' to be %d, got %d", key, expectedValue, actualValue)
				}
			}
		})
	}
}
