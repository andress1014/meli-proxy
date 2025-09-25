package ratelimit

import (
	"net"
	"net/http"
	"regexp"
	"strings"
)

// ExtractIP extrae la IP real del request
func ExtractIP(r *http.Request) string {
	// 1. Intentar X-Forwarded-For (puede contener múltiples IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Tomar la primera IP (la del cliente original)
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// 2. Intentar X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// 3. Fallback a RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

var (
	// Patrones para normalizar paths
	categoriesPattern = regexp.MustCompile(`^/categories/[^/]+(/.*)?$`)
	itemsPattern      = regexp.MustCompile(`^/items/[^/]+(/.*)?$`)
	usersPattern      = regexp.MustCompile(`^/users/[^/]+(/.*)?$`)
	sitesPattern      = regexp.MustCompile(`^/sites/[^/]+(/.*)?$`)
)

// NormalizePath convierte paths específicos a patrones generales
// Ejemplos:
// /categories/MLA1234 -> /categories/*
// /items/MLA123456789 -> /items/*
// /users/123456 -> /users/*
func NormalizePath(path string) string {
	// Limpiar query parameters
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Normalizar trailing slash
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	// Aplicar patrones de normalización
	switch {
	case categoriesPattern.MatchString(path):
		return "/categories/*"
	case itemsPattern.MatchString(path):
		return "/items/*"
	case usersPattern.MatchString(path):
		return "/users/*"
	case sitesPattern.MatchString(path):
		return "/sites/*"
	default:
		return path
	}
}

// GetLimitKeys genera todas las keys necesarias para rate limiting
func GetLimitKeys(r *http.Request) map[string]string {
	ip := ExtractIP(r)
	normalizedPath := NormalizePath(r.URL.Path)

	return map[string]string{
		"ip":      IPKey(ip),
		"path":    PathKey(normalizedPath),
		"ip_path": IPPathKey(ip, normalizedPath),
	}
}
