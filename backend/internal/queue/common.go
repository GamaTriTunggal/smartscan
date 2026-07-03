package queue

import "errors"

// Shared queue errors
var (
	ErrQueueUnavailable = errors.New("queue unavailable")
	ErrInvalidJobData   = errors.New("invalid job data")
	ErrCircuitOpen      = errors.New("circuit breaker is open")
	ErrJobNotFound      = errors.New("job not found")
)

// WorkerState represents the current state of a worker
type WorkerState string

const (
	WorkerStateIdle       WorkerState = "idle"
	WorkerStateProcessing WorkerState = "processing"
	WorkerStateStopped    WorkerState = "stopped"
)

// WorkerStats holds statistics about workers
type WorkerStats struct {
	ActiveWorkers   int   `json:"active_workers"`
	IdleWorkers     int   `json:"idle_workers"`
	TotalProcessed  int64 `json:"total_processed"`
	TotalErrors     int64 `json:"total_errors"`
	AvgProcessingMs int64 `json:"avg_processing_ms"`
}
