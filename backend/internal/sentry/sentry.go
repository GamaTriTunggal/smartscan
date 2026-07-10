package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// storeTransport is a custom transport that sends events to the /store endpoint
// instead of /envelope for compatibility with GlitchTip
type storeTransport struct {
	dsn      string
	client   *http.Client
	key      string
	storeURL string
	hostHeader string // Host header value for ALLOWED_HOSTS validation
}

func newStoreTransport(dsn, glitchtipDomain string) (*storeTransport, error) {
	// Parse DSN: http://key@host:port/project_id
	// Extract key and construct store URL
	// Example: http://abc123@glitchtip:8000/1

	// Find key (between :// and @)
	start := 7 // len("http://")
	if len(dsn) > 8 && dsn[:8] == "https://" {
		start = 8
	}
	atIdx := -1
	for i := start; i < len(dsn); i++ {
		if dsn[i] == '@' {
			atIdx = i
			break
		}
	}
	if atIdx == -1 {
		return nil, fmt.Errorf("invalid DSN: no @ found")
	}

	key := dsn[start:atIdx]
	hostPath := dsn[atIdx+1:] // host:port/project_id

	// Construct store URL
	proto := "http://"
	if start == 8 {
		proto = "https://"
	}
	storeURL := proto + hostPath
	if storeURL[len(storeURL)-1] != '/' {
		// Remove project_id and add /api/PROJECT/store/
		lastSlash := -1
		for i := len(storeURL) - 1; i >= 0; i-- {
			if storeURL[i] == '/' {
				lastSlash = i
				break
			}
		}
		if lastSlash > 0 {
			projectID := storeURL[lastSlash+1:]
			storeURL = storeURL[:lastSlash] + "/api/" + projectID + "/store/"
		}
	}

	// Extract host from GLITCHTIP_DOMAIN for Host header
	// e.g., "http://localhost:8001" -> "localhost:8001"
	hostHeader := glitchtipDomain
	if strings.HasPrefix(hostHeader, "http://") {
		hostHeader = hostHeader[7:]
	} else if strings.HasPrefix(hostHeader, "https://") {
		hostHeader = hostHeader[8:]
	}

	return &storeTransport{
		dsn:        dsn,
		client:     &http.Client{Timeout: 10 * time.Second},
		key:        key,
		storeURL:   storeURL,
		hostHeader: hostHeader,
	}, nil
}

func (t *storeTransport) Configure(options sentry.ClientOptions) {}

func (t *storeTransport) SendEvent(event *sentry.Event) {
	// Build minimal JSON payload that GlitchTip accepts
	// Event ID must be 32 hex chars without hyphens
	eventID := strings.ReplaceAll(string(event.EventID), "-", "")
	if len(eventID) < 32 {
		eventID = eventID + strings.Repeat("0", 32-len(eventID))
	}

	payload := map[string]interface{}{
		"event_id":  eventID,
		"timestamp": event.Timestamp.Format("2006-01-02T15:04:05"),
		"platform":  "go",
		"level":     string(event.Level),
	}

	// Add message
	if event.Message != "" {
		payload["message"] = event.Message
	} else if event.Exception != nil && len(event.Exception) > 0 {
		payload["message"] = event.Exception[0].Value
		payload["exception"] = map[string]interface{}{
			"values": []map[string]interface{}{
				{
					"type":  event.Exception[0].Type,
					"value": event.Exception[0].Value,
				},
			},
		}
	}

	// Add environment and release
	if event.Environment != "" {
		payload["environment"] = event.Environment
	}
	if event.Release != "" {
		payload["release"] = event.Release
	}

	// Add tags
	if len(event.Tags) > 0 {
		payload["tags"] = event.Tags
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Sentry] Failed to marshal event: %v", err)
		return
	}

	req, err := http.NewRequest("POST", t.storeURL, bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("[Sentry] Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sentry-Auth", fmt.Sprintf("Sentry sentry_version=7, sentry_key=%s", t.key))
	// Set Host header to match GLITCHTIP_DOMAIN for ALLOWED_HOSTS validation
	if t.hostHeader != "" {
		req.Host = t.hostHeader
	}

	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("[Sentry] Failed to send event: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Sentry] Server returned %d: %s", resp.StatusCode, string(body))
		return
	}

	log.Printf("[Sentry] Event sent successfully: %s", event.EventID)
}

func (t *storeTransport) Flush(timeout time.Duration) bool {
	return true
}

// Config holds Sentry configuration
type Config struct {
	DSN              string
	Environment      string
	Release          string
	Debug            bool
	SampleRate       float64
	TracesSampleRate float64
	GlitchTipDomain  string // e.g., "http://localhost:8001" - used for Host header
}

// Init initializes the Sentry SDK
func Init(cfg Config) error {
	if cfg.DSN == "" {
		log.Println("Sentry DSN not configured, error tracking disabled")
		return nil
	}

	// Create custom transport that uses /store endpoint for GlitchTip compatibility
	transport, err := newStoreTransport(cfg.DSN, cfg.GlitchTipDomain)
	if err != nil {
		return fmt.Errorf("failed to create store transport: %w", err)
	}

	log.Printf("[Sentry] Using store URL: %s (Host: %s)", transport.storeURL, transport.hostHeader)

	err = sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		Debug:            cfg.Debug,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		Transport:        transport,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Filter out certain errors if needed
			return event
		},
	})
	if err != nil {
		return fmt.Errorf("sentry initialization failed: %w", err)
	}

	log.Println("Sentry initialized successfully")
	return nil
}

// Flush waits for pending events to be sent to Sentry
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// GinMiddleware returns a Gin middleware for Sentry error tracking
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(c.Request)
		hub.Scope().SetTag("handler", c.FullPath())

		// Store hub in context for later use
		c.Set("sentry_hub", hub)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(c.Request.Context(), err)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()

		// Capture errors from context (set by handlers)
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				hub.CaptureException(e.Err)
			}
		}

		// Check for 5xx status codes - capture an event to track server errors
		status := c.Writer.Status()
		if status >= 500 {
			hub.Scope().SetTag("http.status_code", fmt.Sprintf("%d", status))
			// Only capture if no errors were already captured above
			if len(c.Errors) == 0 {
				hub.CaptureMessage(fmt.Sprintf("HTTP %d: %s %s", status, c.Request.Method, c.Request.URL.Path))
			}
		}
	}
}

// CaptureError captures an error with additional context
func CaptureError(c *gin.Context, err error, tags map[string]string) {
	hub := GetHub(c)
	if hub == nil {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		hub.CaptureException(err)
	})
}

// CaptureMessage captures a message with optional tags
func CaptureMessage(c *gin.Context, message string, level sentry.Level, tags map[string]string) {
	hub := GetHub(c)
	if hub == nil {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		hub.CaptureMessage(message)
	})
}

// GetHub retrieves the Sentry hub from Gin context
func GetHub(c *gin.Context) *sentry.Hub {
	if hub, exists := c.Get("sentry_hub"); exists {
		if h, ok := hub.(*sentry.Hub); ok {
			return h
		}
	}
	return sentry.CurrentHub()
}

// SetUser sets user information for the current scope
func SetUser(c *gin.Context, userID string, email string, tenantID string) {
	hub := GetHub(c)
	if hub == nil {
		return
	}

	hub.Scope().SetUser(sentry.User{
		ID:    userID,
		Email: email,
	})
	hub.Scope().SetTag("tenant_id", tenantID)
}

// SetTags sets multiple tags on the current scope
func SetTags(c *gin.Context, tags map[string]string) {
	hub := GetHub(c)
	if hub == nil {
		return
	}

	for k, v := range tags {
		hub.Scope().SetTag(k, v)
	}
}

// SetExtra sets extra context data
func SetExtra(c *gin.Context, key string, value interface{}) {
	hub := GetHub(c)
	if hub == nil {
		return
	}

	hub.Scope().SetExtra(key, value)
}

// Severity levels for error categorization
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

// Error types for categorization
const (
	ErrorTypeDatabase    = "database"
	ErrorTypeValidation  = "validation"
	ErrorTypeAuth        = "auth"
	ErrorTypeExternal    = "external_api"
	ErrorTypeInternal    = "internal"
	ErrorTypeConfig      = "configuration"
)

// CaptureHandlerError is a helper function for handlers to capture errors with standard tags
func CaptureHandlerError(c *gin.Context, err error, handler string, errorType string, severity string) {
	CaptureError(c, err, map[string]string{
		"handler":    handler,
		"error_type": errorType,
		"severity":   severity,
	})
}
