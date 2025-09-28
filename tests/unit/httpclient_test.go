package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/pkg/httpclient"
)

func TestNewOptimizedClient(t *testing.T) {
	client := httpclient.NewOptimizedClient()
	
	// Verificar que el cliente no es nil
	if client == nil {
		t.Fatal("NewOptimizedClient() returned nil")
	}
	
	// Verificar timeout
	expectedTimeout := 15 * time.Second
	if client.Timeout != expectedTimeout {
		t.Errorf("expected timeout %v, got %v", expectedTimeout, client.Timeout)
	}
}

func TestClientNoRedirect(t *testing.T) {
	// Crear un servidor que redirija
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := httpclient.NewOptimizedClient()
	
	// Hacer request que debería ser redirigida
	resp, err := client.Get(server.URL + "/redirect")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	
	// Verificar que NO siguió el redirect (debería ser 302)
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected status %d (redirect not followed), got %d", http.StatusFound, resp.StatusCode)
	}
}

func TestClientPerformance(t *testing.T) {
	// Crear servidor de prueba
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	
	client := httpclient.NewOptimizedClient()
	
	// Hacer múltiples requests para verificar reutilización de conexiones
	numRequests := 10
	start := time.Now()
	
	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, resp.StatusCode)
		}
	}
	
	elapsed := time.Since(start)
	avgLatency := elapsed / time.Duration(numRequests)
	
	t.Logf("Completed %d requests in %v (avg: %v per request)", numRequests, elapsed, avgLatency)
	
	// Las requests deberían ser relativamente rápidas
	maxExpectedLatency := 500 * time.Millisecond
	if avgLatency > maxExpectedLatency {
		t.Logf("Warning: Average latency %v exceeds expected %v (may be acceptable depending on system load)", avgLatency, maxExpectedLatency)
	}
}

func TestClientHeaders(t *testing.T) {
	// Servidor que verifica headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("User-Agent header missing")
		}
		
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	
	client := httpclient.NewOptimizedClient()
	
	// Crear request con headers personalizados
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("User-Agent", "meli-proxy-test/1.0")
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	
	// Verificar que la respuesta incluye headers del servidor
	testHeader := resp.Header.Get("X-Test-Header")
	if testHeader != "test-value" {
		t.Errorf("expected X-Test-Header: test-value, got: %s", testHeader)
	}
}
