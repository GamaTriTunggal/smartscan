package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

var (
	// Default is the default logger instance
	Default *slog.Logger
)

// Config holds logger configuration
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output io.Writer
}

// Init initializes the default logger with the given configuration
func Init(cfg Config) {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	level := parseLevel(cfg.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	Default = slog.New(handler)
	slog.SetDefault(Default)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger is the application logger interface
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// slogLogger wraps slog.Logger to implement Logger interface
type slogLogger struct {
	*slog.Logger
}

// New returns a new logger instance
func New() Logger {
	if Default == nil {
		Init(Config{Level: "info", Format: "json"})
	}
	return &slogLogger{Default}
}

// With returns a new logger with the given attributes
func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{l.Logger.With(args...)}
}

// Debug logs at debug level
func (l *slogLogger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs at info level
func (l *slogLogger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs at warn level
func (l *slogLogger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs at error level
func (l *slogLogger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Package-level convenience functions that use the default logger

// Debug logs at debug level
func Debug(msg string, args ...any) {
	if Default == nil {
		Init(Config{Level: "info", Format: "json"})
	}
	Default.Debug(msg, args...)
}

// Info logs at info level
func Info(msg string, args ...any) {
	if Default == nil {
		Init(Config{Level: "info", Format: "json"})
	}
	Default.Info(msg, args...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	if Default == nil {
		Init(Config{Level: "info", Format: "json"})
	}
	Default.Warn(msg, args...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	if Default == nil {
		Init(Config{Level: "info", Format: "json"})
	}
	Default.Error(msg, args...)
}

// Fatal logs at error level and exits
func Fatal(msg string, args ...any) {
	Error(msg, args...)
	os.Exit(1)
}

// With returns a logger with the given attributes
func With(args ...any) *slog.Logger {
	if Default == nil {
		Init(Config{Level: "info", Format: "json"})
	}
	return Default.With(args...)
}

// WithContext returns a logger with request context
func WithContext(ctx context.Context) *slog.Logger {
	if Default == nil {
		Init(Config{Level: "info", Format: "json"})
	}

	// Extract common context values if present
	logger := Default

	if requestID := ctx.Value("request_id"); requestID != nil {
		logger = logger.With("request_id", requestID)
	}

	if userID := ctx.Value("user_id"); userID != nil {
		logger = logger.With("user_id", userID)
	}

	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		logger = logger.With("tenant_id", tenantID)
	}

	return logger
}

// HTTPRequest creates log attributes for an HTTP request
func HTTPRequest(method, path string, status int, latency time.Duration, clientIP string) []any {
	return []any{
		"http.method", method,
		"http.path", path,
		"http.status", status,
		"http.latency_ms", latency.Milliseconds(),
		"client.ip", clientIP,
	}
}

// DatabaseQuery creates log attributes for a database query
func DatabaseQuery(query string, duration time.Duration, rowsAffected int64) []any {
	return []any{
		"db.query", query,
		"db.duration_ms", duration.Milliseconds(),
		"db.rows_affected", rowsAffected,
	}
}

// QueueMessage creates log attributes for a queue message
func QueueMessage(queueName, messageID, status string) []any {
	return []any{
		"queue.name", queueName,
		"queue.message_id", messageID,
		"queue.status", status,
	}
}
