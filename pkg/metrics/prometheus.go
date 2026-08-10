package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gogate_requests_total",
			Help: "Total number of requests processed by GoGate",
		},
		[]string{"method", "path", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gogate_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	ActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gogate_active_requests",
			Help: "Number of active requests",
		},
		[]string{"path"},
	)

	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gogate_circuit_breaker_state",
			Help: "Circuit breaker state: 0=closed, 1=open, 2=half-open",
		},
		[]string{"route"},
	)

	UpstreamErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gogate_upstream_errors_total",
			Help: "Total upstream errors",
		},
		[]string{"path", "upstream"},
	)
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}

		ActiveRequests.WithLabelValues(path).Inc()
		defer ActiveRequests.WithLabelValues(path).Dec()

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.statusCode)

		RequestTotal.WithLabelValues(r.Method, path, status).Inc()
		RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

func Handler() http.Handler {
	return promhttp.Handler()
}
