package unit

import (
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/internal/metrics"
	"go.uber.org/zap"
)

func TestNewAsyncCollector(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	
	if collector == nil {
		t.Fatal("NewAsyncCollector returned nil")
	}
	
	// Should be able to shutdown without errors
	collector.Shutdown()
}

func TestAsyncCollectorRecordRequest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	defer collector.Shutdown()
	
	// Should not block or panic
	collector.RecordRequestAsync("GET", "/test", 200, 100*time.Millisecond)
	collector.RecordRequestAsync("POST", "/items", 201, 200*time.Millisecond)
	collector.RecordRequestAsync("GET", "/categories", 404, 50*time.Millisecond)
	
	// Give some time for processing
	time.Sleep(50 * time.Millisecond)
}

func TestAsyncCollectorRecordRateLimit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	defer collector.Shutdown()
	
	// Should not block or panic
	collector.RecordRateLimitAsync("ip", "192.168.1.1", true, 50)
	collector.RecordRateLimitAsync("path", "/api/items", false, 0)
	collector.RecordRateLimitAsync("ip_path", "192.168.1.1::/api/items", true, 25)
	
	// Give some time for processing
	time.Sleep(50 * time.Millisecond)
}

func TestAsyncCollectorHighLoad(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	defer collector.Shutdown()
	
	// Send many metrics quickly
	const numMetrics = 500
	for i := 0; i < numMetrics; i++ {
		collector.RecordRequestAsync("GET", "/load-test", 200, 10*time.Millisecond)
		
		if i%10 == 0 {
			collector.RecordRateLimitAsync("ip", "192.168.1.100", true, 100-i/10)
		}
	}
	
	// Give time for processing
	time.Sleep(100 * time.Millisecond)
	
	// Should handle high load without panicking
}

func TestAsyncCollectorConcurrency(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	defer collector.Shutdown()
	
	const numRoutines = 10
	const metricsPerRoutine = 50
	
	done := make(chan bool, numRoutines)
	
	// Start multiple goroutines sending metrics
	for i := 0; i < numRoutines; i++ {
		go func(routineID int) {
			defer func() { done <- true }()
			
			for j := 0; j < metricsPerRoutine; j++ {
				collector.RecordRequestAsync("GET", "/concurrent", 200, 25*time.Millisecond)
				
				if j%5 == 0 {
					remaining := 100 - j
					collector.RecordRateLimitAsync("concurrent", "test-key", j%2 == 0, remaining)
				}
				
				// Small delay to simulate realistic load
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}
	
	// Wait for all routines to complete
	for i := 0; i < numRoutines; i++ {
		<-done
	}
	
	// Give time for processing
	time.Sleep(100 * time.Millisecond)
}

func TestAsyncCollectorBufferOverflow(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	defer collector.Shutdown()
	
	// Send more metrics than buffer can handle
	// The buffer is 10000, so let's send way more
	for i := 0; i < 15000; i++ {
		collector.RecordRequestAsync("GET", "/overflow", 200, 1*time.Millisecond)
	}
	
	// Should handle overflow gracefully (drop metrics)
	time.Sleep(100 * time.Millisecond)
}

func TestAsyncCollectorShutdown(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	
	// Send some metrics
	collector.RecordRequestAsync("GET", "/shutdown", 200, 50*time.Millisecond)
	collector.RecordRateLimitAsync("test", "shutdown", true, 75)
	
	// Shutdown should process remaining metrics
	collector.Shutdown()
	
	// Multiple shutdowns would cause panic, so we don't test them
	// This is expected behavior - shutdown should only be called once
}

func TestAsyncCollectorShutdownTiming(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	
	done := make(chan bool, 1)
	
	// Send metrics continuously in background
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 20; i++ {
			select {
			case <-done:
				return
			default:
				collector.RecordRequestAsync("GET", "/timing", 200, 10*time.Millisecond)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()
	
	// Wait a bit then shutdown
	time.Sleep(50 * time.Millisecond)
	
	start := time.Now()
	collector.Shutdown()
	duration := time.Since(start)
	
	// Notify background goroutine to stop
	select {
	case done <- true:
	default:
	}
	
	// Shutdown should complete reasonably quickly
	if duration > 5*time.Second {
		t.Errorf("Shutdown took too long: %v", duration)
	}
}

func TestAsyncCollectorDifferentMetricTypes(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	defer collector.Shutdown()
	
	// Test different HTTP methods
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		collector.RecordRequestAsync(method, "/test", 200, 100*time.Millisecond)
	}
	
	// Test different status codes
	statuses := []int{200, 201, 400, 404, 429, 500, 503}
	for _, status := range statuses {
		collector.RecordRequestAsync("GET", "/status", status, 50*time.Millisecond)
	}
	
	// Test different rate limit types
	limitTypes := []string{"ip", "path", "ip_path", "user", "api_key"}
	for i, limitType := range limitTypes {
		remaining := 100 - i*10
		collector.RecordRateLimitAsync(limitType, "test-key", i%2 == 0, remaining)
	}
	
	// Give time for processing
	time.Sleep(100 * time.Millisecond)
}

func TestAsyncCollectorBatchProcessing(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewAsyncCollector(logger)
	defer collector.Shutdown()
	
	// Send metrics that should trigger batch processing
	for i := 0; i < 200; i++ {
		collector.RecordRequestAsync("GET", "/batch", 200, 10*time.Millisecond)
		
		if i%20 == 0 {
			collector.RecordRateLimitAsync("batch", "test", i%2 == 0, 100-i)
		}
	}
	
	// Give time for batch processing
	time.Sleep(200 * time.Millisecond)
}
