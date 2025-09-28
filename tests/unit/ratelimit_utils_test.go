package unit

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/andress1014/meli-proxy/internal/ratelimit"
)

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100, 10.0.0.1, 172.16.0.1"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "203.0.113.10"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "203.0.113.10",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "198.51.100.5:54321",
			expectedIP: "198.51.100.5",
		},
		{
			name:       "Invalid X-Forwarded-For, fallback to RemoteAddr",
			headers:    map[string]string{"X-Forwarded-For": "invalid-ip"},
			remoteAddr: "203.0.113.20:12345",
			expectedIP: "203.0.113.20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header:     make(http.Header),
				RemoteAddr: tt.remoteAddr,
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := ratelimit.ExtractIP(req)
			if ip != tt.expectedIP {
				t.Errorf("ExtractIP() = %v, want %v", ip, tt.expectedIP)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Categories path",
			path:     "/categories/MLA1234",
			expected: "/categories/*",
		},
		{
			name:     "Items path",
			path:     "/items/MLA123456789",
			expected: "/items/*",
		},
		{
			name:     "Users path",
			path:     "/users/123456",
			expected: "/users/*",
		},
		{
			name:     "Sites path",
			path:     "/sites/MLA",
			expected: "/sites/*",
		},
		{
			name:     "Generic path",
			path:     "/api/health",
			expected: "/api/health",
		},
		{
			name:     "Path with query params",
			path:     "/categories/MLA1234?limit=10&offset=0",
			expected: "/categories/*",
		},
		{
			name:     "Path with trailing slash",
			path:     "/categories/MLA1234/",
			expected: "/categories/*",
		},
		{
			name:     "Nested categories path",
			path:     "/categories/MLA1234/attributes",
			expected: "/categories/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ratelimit.NormalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("NormalizePath(%v) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetLimitKeys(t *testing.T) {
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "192.168.1.100:12345",
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.10")

	// Simular URL
	req.URL = &url.URL{Path: "/categories/MLA1234"}

	keys := ratelimit.GetLimitKeys(req)

	expectedKeys := map[string]string{
		"ip":      "ip::203.0.113.10",
		"path":    "path::/categories/*",
		"ip_path": "ip_path::203.0.113.10::/categories/*",
	}

	for keyType, expectedKey := range expectedKeys {
		if keys[keyType] != expectedKey {
			t.Errorf("GetLimitKeys()[%v] = %v, want %v", keyType, keys[keyType], expectedKey)
		}
	}
}
