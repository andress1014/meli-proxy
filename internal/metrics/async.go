package metrics

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// AsyncCollector para recopilar métricas sin bloquear requests
type AsyncCollector struct {
	buffer      chan metricEvent
	batchSize   int
	flushTicker *time.Ticker
	wg          sync.WaitGroup
	logger      *zap.Logger
}

type metricEvent struct {
	Type      string
	Method    string
	Path      string
	Status    string
	Duration  time.Duration
	LimitType string
	Key       string
	Allowed   bool
	Remaining int
}

func NewAsyncCollector(logger *zap.Logger) *AsyncCollector {
	collector := &AsyncCollector{
		buffer:      make(chan metricEvent, 10000), // Buffer grande para alta carga
		batchSize:   100,
		flushTicker: time.NewTicker(100 * time.Millisecond), // Flush cada 100ms
		logger:      logger,
	}

	// Iniciar worker async
	collector.wg.Add(1)
	go collector.worker()

	return collector
}

// RecordRequestAsync - No bloquea el request
func (ac *AsyncCollector) RecordRequestAsync(method, path string, status int, duration time.Duration) {
	select {
	case ac.buffer <- metricEvent{
		Type:     "request",
		Method:   method,
		Path:     path,
		Status:   intToString(status),
		Duration: duration,
	}:
		// Enviado al buffer exitosamente
	default:
		// Buffer lleno - hacer drop silencioso para no bloquear
		ac.logger.Warn("metrics buffer full, dropping event",
			zap.String("method", method),
			zap.String("path", path))
	}
}

// RecordRateLimitAsync - No bloquea el request
func (ac *AsyncCollector) RecordRateLimitAsync(limitType, key string, allowed bool, remaining int) {
	select {
	case ac.buffer <- metricEvent{
		Type:      "ratelimit",
		LimitType: limitType,
		Key:       key,
		Allowed:   allowed,
		Remaining: remaining,
	}:
		// Enviado al buffer exitosamente
	default:
		// Buffer lleno - drop silencioso
		ac.logger.Warn("rate limit metrics buffer full")
	}
}

func (ac *AsyncCollector) worker() {
	defer ac.wg.Done()

	batch := make([]metricEvent, 0, ac.batchSize)

	for {
		select {
		case event, ok := <-ac.buffer:
			if !ok {
				// Channel cerrado, procesar batch final
				ac.processBatch(batch)
				return
			}

			batch = append(batch, event)

			// Si el batch está lleno, procesarlo inmediatamente
			if len(batch) >= ac.batchSize {
				ac.processBatch(batch)
				batch = batch[:0] // Reset slice manteniendo capacidad
			}

		case <-ac.flushTicker.C:
			// Flush periódico para evitar latencia en métricas
			if len(batch) > 0 {
				ac.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (ac *AsyncCollector) processBatch(events []metricEvent) {
	for _, event := range events {
		switch event.Type {
		case "request":
			ac.processRequestMetric(event)
		case "ratelimit":
			ac.processRateLimitMetric(event)
		}
	}
}

func (ac *AsyncCollector) processRequestMetric(event metricEvent) {
	// Usar las funciones globales existentes
	RecordRequest(event.Method, event.Path, event.Status, event.Duration)
}

func (ac *AsyncCollector) processRateLimitMetric(event metricEvent) {
	if !event.Allowed {
		RecordRateLimitBlocked(event.LimitType, event.Key)
	}
}

// Shutdown graceful del collector async
func (ac *AsyncCollector) Shutdown() {
	ac.flushTicker.Stop()
	close(ac.buffer)
	ac.wg.Wait()
	ac.logger.Info("async metrics collector shutdown complete")
}

// Helper functions
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToString(i int) string {
	switch i {
	case 200:
		return "200"
	case 404:
		return "404"
	case 429:
		return "429"
	case 500:
		return "500"
	default:
		return "other"
	}
}
