package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds RED-method counters and duration histograms for all notification channels.
type Metrics struct {
	// NotificationsSent counts successful sends, labelled by channel and provider.
	NotificationsSent *prometheus.CounterVec
	// NotificationsFailed counts failed sends, labelled by channel, provider, and reason.
	NotificationsFailed *prometheus.CounterVec
	// NotificationDuration tracks end-to-end send latency per channel.
	NotificationDuration *prometheus.HistogramVec
	// WorkflowRuns tracks workflow execution outcomes.
	WorkflowRuns *prometheus.CounterVec
	// QueueDepth is a gauge for approximate pending jobs per channel.
	QueueDepth *prometheus.GaugeVec
}

// NewMetrics registers and returns all application metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		NotificationsSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "qeet_notify",
			Name:      "notifications_sent_total",
			Help:      "Total notifications delivered successfully.",
		}, []string{"channel", "provider"}),

		NotificationsFailed: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "qeet_notify",
			Name:      "notifications_failed_total",
			Help:      "Total notification send failures.",
		}, []string{"channel", "provider", "reason"}),

		NotificationDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "qeet_notify",
			Name:      "notification_duration_seconds",
			Help:      "End-to-end notification send latency.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"channel"}),

		WorkflowRuns: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "qeet_notify",
			Name:      "workflow_runs_total",
			Help:      "Workflow execution outcomes.",
		}, []string{"status"}),

		QueueDepth: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "qeet_notify",
			Name:      "queue_depth",
			Help:      "Approximate number of pending jobs per channel.",
		}, []string{"channel"}),
	}
}

// Handler returns an http.Handler that exposes the Prometheus metrics endpoint.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
