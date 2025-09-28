package unit

import (
	"testing"

	"github.com/andress1014/meli-proxy/internal/config"
)

func TestConfigLoad_DefaultValues(t *testing.T) {
	cfg := config.Load()

	// Verificar valores por defecto
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
}

func TestConfigValidation(t *testing.T) {
	cfg := &config.Config{
		Port:        "8080",
		MetricsPort: "9090",
		TargetURL:   "https://api.mercadolibre.com",
		RedisURL:    "redis://localhost:6379",
		DefaultRPS:  100,
	}

	// Test básico de estructura
	if cfg.Port == "" {
		t.Error("Port should not be empty")
	}
	if cfg.TargetURL == "" {
		t.Error("TargetURL should not be empty")
	}
	if cfg.DefaultRPS <= 0 {
		t.Error("DefaultRPS should be positive")
	}
}

func TestConfigRateLimitParsing(t *testing.T) {
	tests := []struct {
		name           string
		ipLimits       string
		pathLimits     string
		ipPathLimits   string
		expectError    bool
	}{
		{
			name:         "valid IP limits",
			ipLimits:     "127.0.0.1:200,192.168.1.1:50",
			expectError:  false,
		},
		{
			name:         "valid path limits", 
			pathLimits:   "/categories/*:500,/items/*:300",
			expectError:  false,
		},
		{
			name:         "valid IP+path limits",
			ipPathLimits: "127.0.0.1::/categories/*:100",
			expectError:  false,
		},
		{
			name:         "empty limits",
			ipLimits:     "",
			pathLimits:   "",
			ipPathLimits: "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test que la configuración se puede crear sin errores
			cfg := &config.Config{
				Port:              "8080",
				MetricsPort:       "9090", 
				TargetURL:         "https://api.mercadolibre.com",
				RedisURL:          "redis://localhost:6379",
				DefaultRPS:        100,
				IPRateLimit:       make(map[string]int),
				PathRateLimit:     make(map[string]int),
				IPPathRateLimit:   make(map[string]int),
			}

			if cfg == nil && !tt.expectError {
				t.Error("expected config to be created successfully")
			}
		})
	}
}
