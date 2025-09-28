package unit

import (
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/internal/metrics"
)

func TestMetricsInitialization(t *testing.T) {
	// Test que el sistema de métricas se inicializa correctamente
	server := metrics.NewServer("0") // puerto 0 para auto-asignar
	
	if server == nil {
		t.Fatal("metrics server is nil")
	}
}

func TestRecordRequest(t *testing.T) {
	// Verificar que no hay errores al grabar métricas
	tests := []struct {
		name     string
		method   string
		path     string
		status   string
		duration time.Duration
	}{
		{
			name:     "GET request success",
			method:   "GET",
			path:     "/categories/MLA1234",
			status:   "200",
			duration: 50 * time.Millisecond,
		},
		{
			name:     "POST request",
			method:   "POST",
			path:     "/items",
			status:   "201",
			duration: 100 * time.Millisecond,
		},
		{
			name:     "Error response",
			method:   "GET",
			path:     "/invalid",
			status:   "404",
			duration: 25 * time.Millisecond,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No debería causar panic
			metrics.RecordRequest(tt.method, tt.path, tt.status, tt.duration)
		})
	}
}

func TestRecordRateLimit(t *testing.T) {
	// Test con diferentes tipos de rate limit
	tests := []struct {
		limitType string
		key       string
	}{
		{"ip", "192.168.1.1"},
		{"path", "/categories/*"}, 
		{"ip_path", "192.168.1.1::/items/*"},
		{"unknown_type", "test"},
	}
	
	for _, tt := range tests {
		t.Run("rate_limit_"+tt.limitType, func(t *testing.T) {
			// No debería causar panic
			metrics.RecordRateLimitBlocked(tt.limitType, tt.key)
		})
	}
}

func TestMetricsAsync(t *testing.T) {
	// Test concurrente - simular múltiples requests simultáneas
	done := make(chan bool)
	
	// Goroutine que hace requests concurrentes
	for i := 0; i < 10; i++ {
		go func(id int) {
			metrics.RecordRequest("GET", "/concurrent", "200", time.Duration(id)*time.Millisecond)
			metrics.RecordRateLimitBlocked("ip", "192.168.1.1")
			done <- true
		}(i)
	}
	
	// Esperar a que todas terminen
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// El test pasa si no hay panic o race conditions
	t.Log("Concurrent metrics recording completed successfully")
}

func TestRequestsInProgress(t *testing.T) {
	// Test de métricas de requests en progreso
	method := "GET"
	path := "/test"
	
	// Incrementar
	metrics.IncRequestsInProgress(method, path)
	
	// Decrementar
	metrics.DecRequestsInProgress(method, path)
	
	// No debería causar panic
	t.Log("Requests in progress metrics work correctly")
}

func TestUpdateRPS(t *testing.T) {
	// Test de métrica de RPS
	path := "/categories/*"
	rps := 150.5
	
	metrics.UpdateRequestsPerSecond(path, rps)
	
	// No debería causar panic
	t.Log("RPS metric update works correctly")
}
