package telemetry

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the service.
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	CarrierErrors   *prometheus.CounterVec
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// NewMetrics creates and registers Prometheus metrics.
// It uses sync.Once to ensure metrics are only registered once to avoid
// duplicate registration panics when called multiple times (e.g., in tests).
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = &Metrics{
			RequestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "delivro_requests_total",
					Help: "Total number of requests by operation, carrier, and status",
				},
				[]string{"operation", "carrier", "status"},
			),
			RequestDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "delivro_request_duration_seconds",
					Help:    "Request duration in seconds by operation and carrier",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"operation", "carrier"},
			),
			CarrierErrors: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "delivro_carrier_errors_total",
					Help: "Total carrier API errors by carrier and error type",
				},
				[]string{"carrier", "error_type"},
			),
		}
	})
	return globalMetrics
}

// RecordRequest records a request metric.
func (m *Metrics) RecordRequest(operation, carrier, status string, duration float64) {
	m.RequestsTotal.WithLabelValues(operation, carrier, status).Inc()
	m.RequestDuration.WithLabelValues(operation, carrier).Observe(duration)
}

// RecordError records a carrier error metric.
func (m *Metrics) RecordError(carrier, errorType string) {
	m.CarrierErrors.WithLabelValues(carrier, errorType).Inc()
}
