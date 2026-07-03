package queue

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"gorm.io/gorm"
)

// QRGenerationScanner periodically scans for stuck batches and re-enqueues them.
// This handles recovery scenarios:
//  1. Redis was down when user submitted batch (status = pending_queue)
//  2. Server restarted while batch was processing (status = processing, no active worker)
//  3. Redis crashed mid-generation and job was lost from stream
type QRGenerationScanner struct {
	queue    *RedisQRGenerationQueue
	db       *gorm.DB
	interval time.Duration
	// Stuck threshold: how long a batch can be in "processing" without progress
	// before scanner considers it stuck and re-enqueues.
	stuckThreshold time.Duration

	mu      sync.Mutex
	stopCh  chan struct{}
	doneCh  chan struct{}
	started bool
	stopped bool
}

// QRGenerationScannerConfig holds scanner configuration
type QRGenerationScannerConfig struct {
	Interval       time.Duration
	StuckThreshold time.Duration
}

// DefaultQRGenerationScannerConfig returns default configuration
func DefaultQRGenerationScannerConfig() QRGenerationScannerConfig {
	return QRGenerationScannerConfig{
		Interval:       30 * time.Second,
		StuckThreshold: 10 * time.Minute,
	}
}

// NewQRGenerationScanner creates a new scanner
func NewQRGenerationScanner(queue *RedisQRGenerationQueue, db *gorm.DB, cfg QRGenerationScannerConfig) *QRGenerationScanner {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.StuckThreshold <= 0 {
		cfg.StuckThreshold = 10 * time.Minute
	}
	return &QRGenerationScanner{
		queue:          queue,
		db:             db,
		interval:       cfg.Interval,
		stuckThreshold: cfg.StuckThreshold,
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}
}

// Start begins the scanner in a background goroutine
func (s *QRGenerationScanner) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("scanner already started")
	}
	s.started = true
	s.mu.Unlock()

	go func() {
		defer close(s.doneCh)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[QRGenScanner] Panic recovered: %v", r)
			}
		}()
		s.run(ctx)
	}()

	log.Printf("[QRGenScanner] Started (interval=%s, stuck_threshold=%s)", s.interval, s.stuckThreshold)
	return nil
}

// Stop gracefully stops the scanner
func (s *QRGenerationScanner) Stop() {
	s.mu.Lock()
	if !s.started || s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh
	log.Printf("[QRGenScanner] Stopped")
}

// run is the main scanner loop
func (s *QRGenerationScanner) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run once immediately on start (handles server restart recovery)
	s.scan(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.scan(ctx)
		}
	}
}

// scan queries DB for stuck batches and re-enqueues them.
//
// A batch is considered stuck when:
//  - status = pending_queue (Redis was down when user submitted)
//  - status = queued AND older than queuedGrace (enqueued but never picked up; grace period
//    avoids racing with freshly created batches whose status update is still in flight)
//  - status = processing AND qr_batches.updated_at is older than stuckThreshold
//    (worker touches updated_at per chunk, so a stuck batch really is stuck)
func (s *QRGenerationScanner) scan(ctx context.Context) {
	// Give freshly created batches a grace period before the scanner touches them.
	// This avoids racing with CreateQRBatch which enqueues-then-updates-status.
	queuedGrace := 60 * time.Second
	stuckTime := time.Now().UTC().Add(-s.stuckThreshold)
	queuedStaleTime := time.Now().UTC().Add(-queuedGrace)

	var batches []models.QRBatch

	// Query 1: pending_queue batches older than the grace period
	var pendingQueue []models.QRBatch
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("status = ? AND updated_at < ?", models.QRBatchStatusPendingQueue, queuedStaleTime).
		Limit(100).
		Find(&pendingQueue).Error; err != nil {
		log.Printf("[QRGenScanner] Failed to query pending_queue batches: %v", err)
		return
	}
	batches = append(batches, pendingQueue...)

	// Query 2: queued batches that have sat around too long (Redis stream flushed?)
	var queued []models.QRBatch
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("status = ? AND updated_at < ?", models.QRBatchStatusQueued, queuedStaleTime).
		Limit(100).
		Find(&queued).Error; err != nil {
		log.Printf("[QRGenScanner] Failed to query queued batches: %v", err)
	} else {
		batches = append(batches, queued...)
	}

	// Query 3: processing batches whose updated_at hasn't advanced in stuckThreshold.
	// Worker touches qr_batches.updated_at per chunk (in the same tx as the INSERT),
	// so a truly stuck batch is easy to identify.
	var stuckProcessing []models.QRBatch
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("status = ? AND updated_at < ?", models.QRBatchStatusProcessing, stuckTime).
		Limit(100).
		Find(&stuckProcessing).Error; err != nil {
		log.Printf("[QRGenScanner] Failed to query stuck processing batches: %v", err)
	} else {
		batches = append(batches, stuckProcessing...)
	}

	if len(batches) == 0 {
		return
	}

	log.Printf("[QRGenScanner] Found %d batches needing re-enqueue", len(batches))

	for _, batch := range batches {
		s.reenqueueBatch(ctx, &batch)
	}
}

// reenqueueBatch re-enqueues a single batch to the Redis queue.
//
// Safety:
//   - Re-verifies current status from DB (may have changed since scan query)
//   - Uses an atomic conditional UPDATE to transition state so concurrent reenqueues are serialized
//   - Uses the queue record's generated_count as-is so worker resumes from last chunk
func (s *QRGenerationScanner) reenqueueBatch(ctx context.Context, batch *models.QRBatch) {
	// Re-check current status to avoid re-enqueuing a batch that just completed
	var current models.QRBatch
	if err := s.db.WithContext(ctx).First(&current, "id = ? AND deleted_at IS NULL", batch.ID).Error; err != nil {
		return
	}

	// Skip if reached terminal state
	if current.Status == models.QRBatchStatusCompleted || current.Status == models.QRBatchStatusFailed {
		return
	}

	// Build job. We intentionally reset retry count here — the scanner is a last-resort
	// recovery mechanism, not a normal retry path. The Nack/DLQ flow owns retry accounting.
	// Tenant-lock contention will ensure only one worker actually processes at a time.
	job := NewQRGenerationJob(
		current.ID.String(),
		current.TenantID.String(),
		current.QRCount,
		current.Prefix,
		current.Suffix,
		5, // max retries
	)
	// Attempt to enqueue
	if err := s.queue.Enqueue(ctx, job); err != nil {
		log.Printf("[QRGenScanner] Failed to re-enqueue batch %s: %v", batch.ID, err)
		return
	}

	// Atomic status transition. Only update if status is still one we expect.
	// This prevents flipping a freshly-completed batch back to queued.
	res := s.db.WithContext(ctx).Model(&models.QRBatch{}).
		Where("id = ? AND deleted_at IS NULL AND status IN ?", batch.ID, []models.QRBatchStatus{
			models.QRBatchStatusPendingQueue,
			models.QRBatchStatusQueued,
			models.QRBatchStatusProcessing,
		}).
		Updates(map[string]interface{}{
			"status":     models.QRBatchStatusQueued,
			"updated_at": time.Now().UTC(),
		})
	if res.Error != nil {
		log.Printf("[QRGenScanner] Failed to update batch %s status: %v", batch.ID, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		log.Printf("[QRGenScanner] Batch %s state changed mid-reenqueue; skipping status update", batch.ID)
		return
	}

	log.Printf("[QRGenScanner] Re-enqueued batch %s (was %s, %d QR codes)",
		batch.ID, current.Status, current.QRCount)
}
