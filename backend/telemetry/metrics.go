package telemetry

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: "affyne", Name: "http_request_duration_seconds", Help: "HTTP request latency"},
		[]string{"method", "path", "status"},
	)
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "affyne", Name: "http_requests_total", Help: "Total HTTP requests"},
		[]string{"method", "path", "status"},
	)
	SyncEventsCounter = prometheus.NewCounter(prometheus.CounterOpts{Namespace: "affyne", Name: "sync_events_total", Help: "Total events processed in sync"})
	SyncCycleCounter  = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "affyne", Name: "sync_cycles_total", Help: "Sync cycles"},
		[]string{"result"},
	)
)

// InitMetrics registers all collectors.
func InitMetrics() {
	prometheus.MustRegister(RequestDuration, RequestCounter, SyncEventsCounter, SyncCycleCounter)
}

// Handler returns /metrics HTTP handler.
func Handler() http.Handler { return promhttp.Handler() }

// ObserveRequest records metrics for an HTTP request.
func ObserveRequest(method, path, status string, start time.Time) {
	dur := time.Since(start).Seconds()
	RequestDuration.WithLabelValues(method, path, status).Observe(dur)
	RequestCounter.WithLabelValues(method, path, status).Inc()
}
