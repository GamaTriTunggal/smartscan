package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

// QRGenerationWorker processes QR generation jobs
type QRGenerationWorker struct {
	id             string
	queue          *RedisQRGenerationQueue
	db             *gorm.DB
	chunkSize      int
	state          atomic.Value // WorkerState
	processedCount int64
	errorCount     int64
	stopCh         chan struct{}
	doneCh         chan struct{}

	pollInterval   time.Duration
	claimInterval  time.Duration
	visibilityTime time.Duration
}

// QRGenerationWorkerConfig holds worker configuration
type QRGenerationWorkerConfig struct {
	ChunkSize      int
	PollInterval   time.Duration
	ClaimInterval  time.Duration
	VisibilityTime time.Duration
}

// DefaultQRGenerationWorkerConfig returns sensible defaults
func DefaultQRGenerationWorkerConfig() QRGenerationWorkerConfig {
	return QRGenerationWorkerConfig{
		ChunkSize:      1000,
		PollInterval:   time.Second,
		ClaimInterval:  time.Minute,
		VisibilityTime: 10 * time.Minute,
	}
}

// NewQRGenerationWorker creates a new worker
func NewQRGenerationWorker(id string, queue *RedisQRGenerationQueue, db *gorm.DB, cfg QRGenerationWorkerConfig) *QRGenerationWorker {
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 1000
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.ClaimInterval == 0 {
		cfg.ClaimInterval = time.Minute
	}
	if cfg.VisibilityTime == 0 {
		cfg.VisibilityTime = 10 * time.Minute
	}

	w := &QRGenerationWorker{
		id:             id,
		queue:          queue,
		db:             db,
		chunkSize:      cfg.ChunkSize,
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
		pollInterval:   cfg.PollInterval,
		claimInterval:  cfg.ClaimInterval,
		visibilityTime: cfg.VisibilityTime,
	}
	w.state.Store(WorkerStateIdle)
	return w
}

// Start begins processing jobs
func (w *QRGenerationWorker) Start(ctx context.Context) {
	log.Printf("[QRGenWorker %s] Starting", w.id)
	defer close(w.doneCh)

	claimTicker := time.NewTicker(w.claimInterval)
	defer claimTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[QRGenWorker %s] Context cancelled, shutting down", w.id)
			w.state.Store(WorkerStateStopped)
			return

		case <-w.stopCh:
			log.Printf("[QRGenWorker %s] Stop signal received, shutting down", w.id)
			w.state.Store(WorkerStateStopped)
			return

		case <-claimTicker.C:
			w.claimStaleJobs(ctx)

		default:
			if err := w.processNext(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Printf("[QRGenWorker %s] Error processing job: %v", w.id, err)
				}
			}
		}
	}
}

// Stop gracefully stops the worker
func (w *QRGenerationWorker) Stop() {
	close(w.stopCh)
}

// Wait waits for the worker to finish
func (w *QRGenerationWorker) Wait() {
	<-w.doneCh
}

// processNext retrieves and processes the next job
func (w *QRGenerationWorker) processNext(ctx context.Context) error {
	// Check circuit breaker
	cb := w.queue.GetCircuitBreaker()
	if !cb.Allow() {
		time.Sleep(w.pollInterval * 5)
		return nil
	}

	w.state.Store(WorkerStateIdle)
	job, err := w.queue.Dequeue(ctx, w.id, w.pollInterval)
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}

	w.state.Store(WorkerStateProcessing)

	// Check for duplicate (already processed successfully before)
	if w.queue.IsDuplicate(ctx, job.ID) {
		log.Printf("[QRGenWorker %s] Skipping duplicate job %s", w.id, job.ID)
		return w.queue.Ack(ctx, job)
	}

	// Acquire per-tenant lock (max 1 concurrent generation per tenant)
	lockAcquired, err := w.queue.AcquireTenantLock(ctx, job.TenantID, w.id)
	if err != nil {
		log.Printf("[QRGenWorker %s] Failed to check tenant lock for %s: %v", w.id, job.TenantID, err)
		// Nack and re-enqueue (transient Redis issue)
		return w.queue.Nack(ctx, job, true)
	}
	if !lockAcquired {
		// Another worker is processing a job for this tenant
		// Re-enqueue so it goes to the back of the queue (will try again later).
		log.Printf("[QRGenWorker %s] Tenant %s already has active generation, re-queuing", w.id, job.TenantID)
		// Small delay to avoid tight loop when lots of jobs are blocked on the same tenant
		time.Sleep(500 * time.Millisecond)
		// Enqueue-then-ack: only drop the original stream message after the
		// re-enqueue is confirmed, so a transient Enqueue failure (e.g. circuit
		// breaker open) cannot lose the job from both the stream and the PEL.
		oldStreamID := job.StreamID
		job.StreamID = ""
		if err := w.queue.Enqueue(ctx, job); err != nil {
			job.StreamID = oldStreamID
			return err
		}
		w.queue.client.XAck(ctx, QRGenStreamQueue, QRGenConsumerGroup, oldStreamID)
		w.queue.client.XDel(ctx, QRGenStreamQueue, oldStreamID)
		return nil
	}

	// Ensure lock is released when we're done. Use a fresh context because the worker-pool
	// context may already be cancelled during graceful shutdown — we still want the release
	// call to reach Redis so the next worker can acquire the lock immediately.
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()
		if err := w.queue.ReleaseTenantLock(releaseCtx, job.TenantID, w.id); err != nil {
			log.Printf("[QRGenWorker %s] Failed to release tenant lock: %v", w.id, err)
		}
	}()

	// Process the job
	err = w.processJob(ctx, job)
	if err != nil {
		atomic.AddInt64(&w.errorCount, 1)
		cb.RecordFailure()

		// Increment BEFORE the retry decision. CanRetry() is a post-increment
		// check (RetryCount < MaxRetries) and Nack re-runs it internally; branching
		// on the pre-increment value made the exhaustion path below unreachable, so
		// the last attempt went to the DLQ without ever marking the batch failed
		// (leaving it stuck in "processing" for the scanner to re-enqueue forever).
		job.IncrementRetry(err)
		if job.CanRetry() {
			log.Printf("[QRGenWorker %s] Job %s (batch %s) failed (attempt %d/%d): %v, requeuing",
				w.id, job.ID, job.BatchID, job.RetryCount, job.MaxRetries, err)
			return w.queue.Nack(ctx, job, true)
		}

		// Max retries exceeded, mark batch as failed and move to DLQ
		log.Printf("[QRGenWorker %s] Job %s (batch %s) failed after max retries, moving to DLQ: %v",
			w.id, job.ID, job.BatchID, err)
		if mbErr := w.markBatchFailed(ctx, job.BatchID, fmt.Sprintf("Max retries exceeded: %v", err)); mbErr != nil {
			log.Printf("[QRGenWorker %s] Also failed to mark batch as failed: %v", w.id, mbErr)
		}
		return w.queue.MoveToDLQ(ctx, job, fmt.Sprintf("Max retries exceeded: %v", err))
	}

	// Success
	atomic.AddInt64(&w.processedCount, 1)
	cb.RecordSuccess()

	return w.queue.Ack(ctx, job)
}

// processJob generates QR codes in chunks and updates progress.
//
// Safety mechanisms:
//   - Early-exit if batch is already in a terminal state (prevents shadow-job re-processing)
//   - Verifies batch.tenant_id matches job.tenant_id (defense-in-depth)
//   - Atomic optimistic concurrency: each chunk UPDATE requires (generated_count == expected AND worker_id == w.id)
//   - Touches qr_batches.updated_at per chunk so the scanner's stuck-detection sees active progress
//   - Background goroutine refreshes tenant lock every 5 min; aborts if lock is lost
func (w *QRGenerationWorker) processJob(ctx context.Context, job *QRGenerationJob) error {
	batchUUID, err := uuid.Parse(job.BatchID)
	if err != nil {
		return fmt.Errorf("invalid batch ID: %w", err)
	}

	// Verify batch still exists and is not deleted
	var batch models.QRBatch
	if err := w.db.WithContext(ctx).First(&batch, "id = ? AND deleted_at IS NULL", batchUUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[QRGenWorker %s] Batch %s no longer exists, skipping", w.id, job.BatchID)
			return nil
		}
		return fmt.Errorf("failed to load batch: %w", err)
	}

	// Defense-in-depth: verify the batch actually belongs to the tenant named in the job payload
	if batch.TenantID.String() != job.TenantID {
		log.Printf("[QRGenWorker %s] SECURITY: batch %s tenant mismatch (job=%s db=%s), dropping job",
			w.id, job.BatchID, job.TenantID, batch.TenantID)
		return nil
	}

	// Early-exit for terminal statuses — a shadow/stale job should never reprocess a completed/failed batch
	if batch.Status == models.QRBatchStatusCompleted || batch.Status == models.QRBatchStatusFailed {
		log.Printf("[QRGenWorker %s] Batch %s is already in terminal state %s, skipping",
			w.id, job.BatchID, batch.Status)
		return nil
	}

	// Transition to processing if not already.
	if batch.Status != models.QRBatchStatusProcessing {
		if err := w.db.WithContext(ctx).Model(&batch).Update("status", models.QRBatchStatusProcessing).Error; err != nil {
			return fmt.Errorf("failed to update batch status to processing: %w", err)
		}
	}
	// Get or create qr_generation_queue record (tracks per-chunk progress)
	now := time.Now().UTC()
	var queueRecord models.QRGenerationQueue
	err = w.db.WithContext(ctx).Where("batch_id = ?", batchUUID).First(&queueRecord).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		queueRecord = models.QRGenerationQueue{
			BatchID:      batchUUID,
			TotalQRCount: job.TotalQRCount,
			Status:       models.QRGenerationQueueStatusProcessing,
			WorkerID:     w.id,
			StartedAt:    &now,
		}
		if err := w.db.WithContext(ctx).Create(&queueRecord).Error; err != nil {
			return fmt.Errorf("failed to create queue record: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to load queue record: %w", err)
	} else {
		// Claim ownership by updating worker_id. This is the key coordination point:
		// - Any subsequent chunk update uses (worker_id = w.id) as part of the WHERE clause.
		// - If another worker claims this record, our chunk updates will affect 0 rows and we abort.
		updates := map[string]interface{}{
			"status":    models.QRGenerationQueueStatusProcessing,
			"worker_id": w.id,
		}
		if queueRecord.StartedAt == nil {
			updates["started_at"] = now
		}
		if err := w.db.WithContext(ctx).Model(&queueRecord).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update queue record: %w", err)
		}
	}

	// Normalize dirty state where generated_count > total (from manual edits or previous bugs)
	if queueRecord.GeneratedCount > job.TotalQRCount {
		log.Printf("[QRGenWorker %s] WARNING: batch %s has generated_count=%d > total=%d (dirty state), capping to total",
			w.id, job.BatchID, queueRecord.GeneratedCount, job.TotalQRCount)
		queueRecord.GeneratedCount = job.TotalQRCount
		w.db.WithContext(ctx).Model(&queueRecord).Update("generated_count", job.TotalQRCount)
	}

	// Resume from last generated count
	startCount := queueRecord.GeneratedCount
	if startCount >= job.TotalQRCount {
		log.Printf("[QRGenWorker %s] Batch %s already fully generated (%d/%d), marking complete",
			w.id, job.BatchID, startCount, job.TotalQRCount)
		return w.markBatchCompleted(ctx, batchUUID, &queueRecord)
	}

	log.Printf("[QRGenWorker %s] Starting/resuming batch %s: %d/%d (%.1f%%)",
		w.id, job.BatchID, startCount, job.TotalQRCount,
		float64(startCount)/float64(job.TotalQRCount)*100)

	// Background goroutine: refresh tenant lock every 5 minutes to prevent expiry mid-job.
	// Signals via lockLostCh if the lock is lost (another worker stole it).
	lockLostCh := make(chan struct{}, 1)
	refreshStopCh := make(chan struct{})
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-refreshStopCh:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				if err := w.queue.RefreshTenantLock(ctx, job.TenantID, w.id); err != nil {
					log.Printf("[QRGenWorker %s] Failed to refresh tenant lock for batch %s: %v",
						w.id, job.BatchID, err)
					select {
					case lockLostCh <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}()
	defer close(refreshStopCh)

	// Generate in chunks
	lastProgressLog := startCount
	for generated := startCount; generated < job.TotalQRCount; generated += w.chunkSize {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-lockLostCh:
			return fmt.Errorf("tenant lock lost mid-job for batch %s", job.BatchID)
		default:
		}

		// Calculate chunk size for this iteration (might be smaller at the end)
		remaining := job.TotalQRCount - generated
		chunkSize := w.chunkSize
		if remaining < chunkSize {
			chunkSize = remaining
		}

		// Build chunk of QR codes.
		chunk := make([]models.QRCode, chunkSize)
		for i := 0; i < chunkSize; i++ {
			qrCodeStr := fmt.Sprintf("%s%s%s", job.Prefix, utils.GenerateRandomHexWithFallback(16), job.Suffix)

			chunk[i] = models.QRCode{
				BatchID: batchUUID,
				QRCode:  qrCodeStr,
				Status:  models.QRCodeStatusActive,
			}
		}

		// Insert chunk + update progress in one transaction.
		// Uses optimistic concurrency: the UPDATE requires worker_id AND generated_count to match,
		// which guarantees that if another worker has claimed the record, our INSERT is rolled back.
		expectedCount := generated
		newCount := generated + chunkSize
		err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Re-verify batch is not soft-deleted (locking read, so a parallel delete blocks)
			var b models.QRBatch
			if err := tx.Select("id", "status", "deleted_at", "tenant_id").
				First(&b, "id = ? AND deleted_at IS NULL", batchUUID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("batch no longer exists or was deleted")
				}
				return err
			}
			// Terminal state check inside the transaction (another worker may have finished it)
			if b.Status == models.QRBatchStatusCompleted || b.Status == models.QRBatchStatusFailed {
				return fmt.Errorf("batch reached terminal state %s, aborting", b.Status)
			}

			if err := tx.Create(&chunk).Error; err != nil {
				return err
			}

			// Optimistic update: only proceed if the record is still owned by this worker
			// AND generated_count matches what we expect. If not, another worker got there first.
			res := tx.Model(&models.QRGenerationQueue{}).
				Where("id = ? AND worker_id = ? AND generated_count = ?", queueRecord.ID, w.id, expectedCount).
				Update("generated_count", newCount)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("queue record ownership lost (another worker is processing batch %s)", job.BatchID)
			}

			// Touch qr_batches.updated_at so the scanner sees active progress.
			// Without this, scanner would mark long-running batches as "stuck".
			if err := tx.Model(&models.QRBatch{}).
				Where("id = ?", batchUUID).
				UpdateColumn("updated_at", time.Now().UTC()).Error; err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to insert chunk at offset %d: %w", generated, err)
		}

		// Log progress every 10%
		currentCount := generated + chunkSize
		tenPercent := job.TotalQRCount / 10
		if tenPercent > 0 && (currentCount-lastProgressLog) >= tenPercent {
			log.Printf("[QRGenWorker %s] Batch %s progress: %d/%d (%.1f%%)",
				w.id, job.BatchID, currentCount, job.TotalQRCount,
				float64(currentCount)/float64(job.TotalQRCount)*100)
			lastProgressLog = currentCount
		}
	}

	// Mark as completed
	return w.markBatchCompleted(ctx, batchUUID, &queueRecord)
}

// markBatchCompleted updates batch and queue record to completed state
func (w *QRGenerationWorker) markBatchCompleted(ctx context.Context, batchID uuid.UUID, queueRecord *models.QRGenerationQueue) error {
	now := time.Now().UTC()
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.QRBatch{}).
			Where("id = ?", batchID).
			Updates(map[string]interface{}{
				"status": models.QRBatchStatusCompleted,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&models.QRGenerationQueue{}).
			Where("id = ?", queueRecord.ID).
			Updates(map[string]interface{}{
				"status":       models.QRGenerationQueueStatusCompleted,
				"completed_at": now,
			}).Error
	})
	if err != nil {
		log.Printf("[QRGenWorker %s] Failed to mark batch %s completed: %v", w.id, batchID, err)
		return fmt.Errorf("failed to mark batch completed: %w", err)
	}
	log.Printf("[QRGenWorker %s] Batch %s generation completed", w.id, batchID)

	// In-app notification + optional webhook so staff know the batch is ready
	// for export. Best-effort — never fails the job.
	var batch models.QRBatch
	if err := w.db.WithContext(ctx).Preload("Product").First(&batch, "id = ?", batchID).Error; err == nil {
		productName := ""
		if batch.Product != nil {
			productName = batch.Product.ProductName
		}
		title := "QR batch ready"
		body := fmt.Sprintf("%s — batch %q (%s codes) finished generating and is ready to export.",
			productName, batch.BatchName, formatInt(batch.QRCount))
		n := models.Notification{
			TenantID: batch.TenantID,
			Type:     models.NotificationTypeQRBatchReady,
			Title:    title,
			Body:     body,
			Link:     "/tenant/qr-batches/" + batch.ID.String(),
		}
		if err := w.db.WithContext(ctx).Create(&n).Error; err != nil {
			log.Printf("[QRGenWorker %s] failed to create batch-ready notification: %v", w.id, err)
		}
		utils.SendWebhook(w.db, batch.TenantID, "qr_batch_ready", map[string]interface{}{
			"event":        "qr_batch_ready",
			"batch_id":     batch.ID.String(),
			"batch_name":   batch.BatchName,
			"product_name": productName,
			"qr_count":     batch.QRCount,
		})
	}
	return nil
}

// formatInt renders an int with thousands separators for notification copy.
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}

// markBatchFailed updates batch and queue record to failed state.
// Returns error if the DB writes fail so the caller can escalate.
func (w *QRGenerationWorker) markBatchFailed(ctx context.Context, batchID string, errorMsg string) error {
	batchUUID, err := uuid.Parse(batchID)
	if err != nil {
		return fmt.Errorf("invalid batch ID: %w", err)
	}
	now := time.Now().UTC()

	err = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.QRBatch{}).
			Where("id = ?", batchUUID).
			Update("status", models.QRBatchStatusFailed).Error; err != nil {
			return err
		}
		return tx.Model(&models.QRGenerationQueue{}).
			Where("batch_id = ?", batchUUID).
			Updates(map[string]interface{}{
				"status":        models.QRGenerationQueueStatusFailed,
				"error_message": errorMsg,
				"completed_at":  now,
			}).Error
	})
	if err != nil {
		log.Printf("[QRGenWorker %s] Failed to mark batch %s as failed: %v (original error: %s)",
			w.id, batchID, err, errorMsg)
		return err
	}
	log.Printf("[QRGenWorker %s] Batch %s marked as failed: %s", w.id, batchID, errorMsg)
	return nil
}

// claimStaleJobs attempts to claim jobs from dead workers and processes them
// with the full retry/DLQ/ack lifecycle so no claimed message is orphaned.
func (w *QRGenerationWorker) claimStaleJobs(ctx context.Context) {
	jobs, err := w.queue.ClaimStaleJobs(ctx, w.id, w.visibilityTime)
	if err != nil {
		log.Printf("[QRGenWorker %s] Failed to claim stale jobs: %v", w.id, err)
		return
	}

	for _, job := range jobs {
		log.Printf("[QRGenWorker %s] Processing claimed stale job %s (batch %s)", w.id, job.ID, job.BatchID)

		// Try to acquire tenant lock. If another worker holds it, the lock owner is still
		// processing — re-queue so it goes to the back of the stream for a later attempt.
		lockAcquired, lockErr := w.queue.AcquireTenantLock(ctx, job.TenantID, w.id)
		if lockErr != nil {
			log.Printf("[QRGenWorker %s] Failed to check tenant lock for claimed job %s: %v",
				w.id, job.ID, lockErr)
			// Best-effort Nack to retry later
			_ = w.queue.Nack(ctx, job, true)
			continue
		}
		if !lockAcquired {
			log.Printf("[QRGenWorker %s] Tenant %s has active lock; re-queueing claimed job %s",
				w.id, job.TenantID, job.ID)
			// Enqueue-then-ack: keep the claimed message in the PEL until the
			// re-enqueue is confirmed, so a transient Enqueue failure cannot drop it.
			oldStreamID := job.StreamID
			job.StreamID = ""
			if enqErr := w.queue.Enqueue(ctx, job); enqErr != nil {
				log.Printf("[QRGenWorker %s] Failed to re-enqueue claimed job %s: %v",
					w.id, job.ID, enqErr)
				job.StreamID = oldStreamID // leave it pending for the next claim cycle
				continue
			}
			w.queue.client.XAck(ctx, QRGenStreamQueue, QRGenConsumerGroup, oldStreamID)
			w.queue.client.XDel(ctx, QRGenStreamQueue, oldStreamID)
			continue
		}

		// Process the job with proper release + ack/nack handling
		processErr := w.processJob(ctx, job)

		// Always release the tenant lock with a fresh context (in case ctx was cancelled)
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = w.queue.ReleaseTenantLock(releaseCtx, job.TenantID, w.id)
		releaseCancel()

		if processErr != nil {
			atomic.AddInt64(&w.errorCount, 1)
			log.Printf("[QRGenWorker %s] Error processing claimed job %s: %v",
				w.id, job.ID, processErr)

			// Increment before the retry decision (see processNext for why): otherwise
			// the exhaustion branch never runs and the batch is never marked failed.
			job.IncrementRetry(processErr)
			if job.CanRetry() {
				if nackErr := w.queue.Nack(ctx, job, true); nackErr != nil {
					log.Printf("[QRGenWorker %s] Failed to Nack claimed job: %v", w.id, nackErr)
				}
			} else {
				if mbErr := w.markBatchFailed(ctx, job.BatchID,
					fmt.Sprintf("Max retries exceeded (claimed): %v", processErr)); mbErr != nil {
					log.Printf("[QRGenWorker %s] Also failed to mark batch as failed: %v", w.id, mbErr)
				}
				if dlqErr := w.queue.MoveToDLQ(ctx, job,
					fmt.Sprintf("Max retries exceeded (claimed): %v", processErr)); dlqErr != nil {
					log.Printf("[QRGenWorker %s] Failed to move claimed job to DLQ: %v", w.id, dlqErr)
				}
			}
		} else {
			atomic.AddInt64(&w.processedCount, 1)
			if ackErr := w.queue.Ack(ctx, job); ackErr != nil {
				log.Printf("[QRGenWorker %s] Failed to Ack claimed job: %v", w.id, ackErr)
			}
		}
	}
}

// ID returns the worker ID
func (w *QRGenerationWorker) ID() string {
	return w.id
}

// State returns the current worker state
func (w *QRGenerationWorker) State() WorkerState {
	return w.state.Load().(WorkerState)
}

// GetStats returns worker statistics
func (w *QRGenerationWorker) GetStats() WorkerStats {
	state := w.state.Load().(WorkerState)
	stats := WorkerStats{
		TotalProcessed: atomic.LoadInt64(&w.processedCount),
		TotalErrors:    atomic.LoadInt64(&w.errorCount),
	}
	if state == WorkerStateProcessing {
		stats.ActiveWorkers = 1
	} else if state == WorkerStateIdle {
		stats.IdleWorkers = 1
	}
	return stats
}
