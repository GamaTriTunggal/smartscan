package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts total HTTP requests by method, path, and status
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smartscan_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration measures HTTP request latency
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "smartscan_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// HTTPRequestsInFlight tracks concurrent requests
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "smartscan_http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	// HTTPResponseSize measures response body size
	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "smartscan_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "path"},
	)

	// DatabaseQueryDuration measures database query latency
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "smartscan_database_query_duration_seconds",
			Help:    "Database query latency in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation"},
	)

	// DatabaseConnectionsActive tracks active database connections
	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "smartscan_database_connections_active",
			Help: "Number of active database connections",
		},
	)

	// QueueMessagesTotal counts queue messages by queue and status
	QueueMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smartscan_queue_messages_total",
			Help: "Total number of queue messages processed",
		},
		[]string{"queue", "status"},
	)

	// QueueMessageDuration measures queue message processing time
	QueueMessageDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "smartscan_queue_message_duration_seconds",
			Help:    "Queue message processing time in seconds",
			Buckets: []float64{.01, .05, .1, .5, 1, 5, 10, 30},
		},
		[]string{"queue"},
	)

	// QRScansTotal counts QR scans by type
	QRScansTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smartscan_qr_scans_total",
			Help: "Total number of QR code scans",
		},
		[]string{"type", "result"},
	)

	// EmailsSentTotal counts emails sent by status
	EmailsSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smartscan_emails_sent_total",
			Help: "Total number of emails sent",
		},
		[]string{"type", "status"},
	)

	// AuthEventsTotal counts authentication events
	AuthEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smartscan_auth_events_total",
			Help: "Total number of authentication events",
		},
		[]string{"event", "result"},
	)
)
