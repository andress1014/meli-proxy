package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// NewOptimizedClient crea un cliente HTTP optimizado para alta carga
func NewOptimizedClient() *http.Client {
	transport := &http.Transport{
		// Pool masivo para alta carga
		MaxIdleConns:        10000, // Pool global gigante
		MaxIdleConnsPerHost: 1000,  // Por host (api.mercadolibre.com)
		MaxConnsPerHost:     2000,  // Conexiones totales por host

		// Timeouts agresivos para alta carga
		IdleConnTimeout:       30 * time.Second, // Más corto
		TLSHandshakeTimeout:   3 * time.Second,  // Más rápido
		ExpectContinueTimeout: 500 * time.Millisecond,
		ResponseHeaderTimeout: 10 * time.Second, // Más agresivo

		// TCP optimizations
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,  // Conexión rápida
			KeepAlive: 30 * time.Second, // Keep alive TCP
		}).DialContext,

		// Keep-alive crítico para rendimiento
		DisableKeepAlives: false,

		// HTTP/2 para multiplexing
		ForceAttemptHTTP2: true,

		// TLS optimizado
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
			// Session resumption para TLS
			ClientSessionCache: tls.NewLRUClientSessionCache(1024),
		},

		// CRÍTICO: Sin compresión para evitar overhead en alta carga
		DisableCompression: true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second, // Timeout más agresivo para alta carga

		// CRÍTICO: NO seguir redirects (transparencia total)
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Devolver redirect sin seguirlo
		},
	}
}

// NewHighPerformanceClient para casos extremos de carga
func NewHighPerformanceClient() *http.Client {
	transport := &http.Transport{
		// Configuración extrema para 50K RPS
		MaxIdleConns:        50000, // Pool masivo
		MaxIdleConnsPerHost: 5000,  // Por instancia de MercadoLibre
		MaxConnsPerHost:     10000, // Límite alto

		// Timeouts ultra-agresivos
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 100 * time.Millisecond,
		ResponseHeaderTimeout: 5 * time.Second,

		// TCP con kernel bypass (para producción extrema)
		DialContext: (&net.Dialer{
			Timeout:   500 * time.Millisecond,
			KeepAlive: 15 * time.Second,
		}).DialContext,

		DisableKeepAlives: false,
		ForceAttemptHTTP2: true,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
			ClientSessionCache: tls.NewLRUClientSessionCache(10000),
		},

		// Sin compresión para máximo rendimiento
		DisableCompression: true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   8 * time.Second, // Ultra-agresivo

		// NUNCA seguir redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
