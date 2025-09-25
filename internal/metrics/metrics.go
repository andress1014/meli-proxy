package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Contador de requests totales
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "meli_proxy_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// Contador de rate limits
	rateLimitBlocked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "meli_proxy_rate_limit_blocked_total",
			Help: "Total number of requests blocked by rate limiting",
		},
		[]string{"limit_type", "key"},
	)

	// Histograma de latencias
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "meli_proxy_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// Gauge de requests en progreso
	requestsInProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "meli_proxy_requests_in_progress",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"method", "path"},
	)

	// Requests por segundo
	requestsPerSecond = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "meli_proxy_requests_per_second",
			Help: "Current requests per second",
		},
		[]string{"path"},
	)
)

func init() {
	// Registrar métricas
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(rateLimitBlocked)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(requestsInProgress)
	prometheus.MustRegister(requestsPerSecond)
}

type Server struct {
	server *http.Server
}

func NewServer(port string) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return &Server{
		server: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
	}
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx interface{}) error {
	if httpCtx, ok := ctx.(interface {
		Done() <-chan struct{}
		Deadline() (time.Time, bool)
		Err() error
		Value(key interface{}) interface{}
	}); ok {
		return s.server.Shutdown(httpCtx)
	}
	return s.server.Close()
}

// Métodos para incrementar métricas
func RecordRequest(method, path, status string, duration time.Duration) {
	requestsTotal.WithLabelValues(method, path, status).Inc()
	requestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
}

func RecordRateLimitBlocked(limitType, key string) {
	rateLimitBlocked.WithLabelValues(limitType, key).Inc()
}

func IncRequestsInProgress(method, path string) {
	requestsInProgress.WithLabelValues(method, path).Inc()
}

func DecRequestsInProgress(method, path string) {
	requestsInProgress.WithLabelValues(method, path).Dec()
}

func UpdateRequestsPerSecond(path string, rps float64) {
	requestsPerSecond.WithLabelValues(path).Set(rps)
}
