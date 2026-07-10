package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// HTTPMetrics is a Gin middleware that records HTTP metrics
func HTTPMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics endpoint to avoid self-referencing
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// Track in-flight requests
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get normalized path for metrics (avoid high cardinality)
		path := normalizePath(c.FullPath())
		if path == "" {
			path = "unknown"
		}

		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		// Record metrics
		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
		HTTPResponseSize.WithLabelValues(method, path).Observe(float64(c.Writer.Size()))
	}
}

// normalizePath normalizes the path to prevent high cardinality
// Replaces path parameters with placeholders
func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	return path
}

// RecordQRScan records a QR scan metric
func RecordQRScan(scanType, result string) {
	QRScansTotal.WithLabelValues(scanType, result).Inc()
}

// RecordEmailSent records an email sent metric
func RecordEmailSent(emailType, status string) {
	EmailsSentTotal.WithLabelValues(emailType, status).Inc()
}

// RecordAuthEvent records an authentication event metric
func RecordAuthEvent(event, result string) {
	AuthEventsTotal.WithLabelValues(event, result).Inc()
}

// RecordDatabaseQuery records a database query metric
func RecordDatabaseQuery(operation string, duration time.Duration) {
	DatabaseQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordQueueMessage records a queue message metric
func RecordQueueMessage(queue, status string, duration time.Duration) {
	QueueMessagesTotal.WithLabelValues(queue, status).Inc()
	QueueMessageDuration.WithLabelValues(queue).Observe(duration.Seconds())
}

// SetDatabaseConnections sets the current database connection count
func SetDatabaseConnections(count float64) {
	DatabaseConnectionsActive.Set(count)
}
