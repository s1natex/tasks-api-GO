package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

// Register metrics with Prometheus's default registry
func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// MetricsMiddleware records Prometheus metrics for each HTTP request
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		duration := time.Since(start).Seconds()
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}

		labels := prometheus.Labels{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": strconv.Itoa(status),
		}
		httpRequestsTotal.With(labels).Inc()
		httpRequestDuration.With(labels).Observe(duration)
	})
}

// MetricsHandler returns an HTTP handler for Prometheus metrics exposition
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
