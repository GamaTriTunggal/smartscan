package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// QR generation stream and consumer group names
const (
	QRGenStreamQueue         = "qr:generation:queue"
	QRGenStreamDLQ           = "qr:generation:dlq"
	QRGenConsumerGroup       = "qr-generation-workers"
	QRGenTenantLockPrefix    = "qr:gen:lock:tenant:"
	QRGenJobDedupPrefix      = "qr:gen:processed:"
	QRGenJobDedupTTL         = 24 * time.Hour
)

// RedisQRGenerationQueue implements QR generation queue using Redis Streams
type RedisQRGenerationQueue struct {
	client         *redis.Client
	db             *gorm.DB
	circuitBreaker *CircuitBreaker
	maxLen         int64
	tenantLockTTL  time.Duration
}

// RedisQRGenerationQueueConfig holds configuration for the queue
type RedisQRGenerationQueueConfig struct {
	MaxStreamLength      int64
	TenantLockTTL        time.Duration
	CircuitBreakerConfig CircuitBreakerConfig
}

// DefaultRedisQRGenerationQueueConfig returns default configuration
func DefaultRedisQRGenerationQueueConfig() RedisQRGenerationQueueConfig {
	return RedisQRGenerationQueueConfig{
		MaxStreamLength:      10000, // 10K jobs in stream max
		TenantLockTTL:        15 * time.Minute,
		CircuitBreakerConfig: DefaultCircuitBreakerConfig(),
	}
}

// NewRedisQRGenerationQueue creates a new Redis-based QR generation queue
func NewRedisQRGenerationQueue(client *redis.Client, db *gorm.DB, cfg RedisQRGenerationQueueConfig) (*RedisQRGenerationQueue, error) {
	if client == nil {
		return nil, errors.New("redis client is required")
	}
	if cfg.MaxStreamLength <= 0 {
		cfg.MaxStreamLength = 10000
	}
	if cfg.TenantLockTTL <= 0 {
		cfg.TenantLockTTL = 15 * time.Minute
	}

	q := &RedisQRGenerationQueue{
		client:         client,
		db:             db,
		circuitBreaker: NewCircuitBreaker(cfg.CircuitBreakerConfig),
		maxLen:         cfg.MaxStreamLength,
		tenantLockTTL:  cfg.TenantLockTTL,
	}

	// Initialize consumer group
	ctx := context.Background()
	if err := q.ensureConsumerGroup(ctx); err != nil {
		log.Printf("Warning: failed to create QR generation consumer group: %v", err)
	}

	return q, nil
}

// ensureConsumerGroup creates the consumer group if it doesn't exist
func (q *RedisQRGenerationQueue) ensureConsumerGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, QRGenStreamQueue, QRGenConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// Enqueue adds a QR generation job to the queue
func (q *RedisQRGenerationQueue) Enqueue(ctx context.Context, job *QRGenerationJob) error {
	if job.ID == "" {
		return errors.New("job ID is required")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}

	// Check circuit breaker
	if !q.circuitBreaker.Allow() {
		return ErrCircuitOpen
	}

	data := job.ToMap()

	args := &redis.XAddArgs{
		Stream: QRGenStreamQueue,
		MaxLen: q.maxLen,
		Approx: true,
		Values: data,
	}

	id, err := q.client.XAdd(ctx, args).Result()
	if err != nil {
		q.circuitBreaker.RecordFailure()
		log.Printf("Failed to enqueue QR generation job %s to Redis: %v", job.ID, err)
		return fmt.Errorf("%w: %v", ErrQueueUnavailable, err)
	}

	q.circuitBreaker.RecordSuccess()
	job.StreamID = id
	log.Printf("[QRGenQueue] Enqueued job %s for batch %s with stream ID %s", job.ID, job.BatchID, id)
	return nil
}

// Dequeue retrieves the next job for processing
// Returns (nil, nil) if no job is available
func (q *RedisQRGenerationQueue) Dequeue(ctx context.Context, workerID string, timeout time.Duration) (*QRGenerationJob, error) {
	args := &redis.XReadGroupArgs{
		Group:    QRGenConsumerGroup,
		Consumer: workerID,
		Streams:  []string{QRGenStreamQueue, ">"},
		Count:    1,
		Block:    timeout,
	}

	results, err := q.client.XReadGroup(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read from QR generation stream: %w", err)
	}

	if len(results) == 0 || len(results[0].Messages) == 0 {
		return nil, nil
	}

	msg := results[0].Messages[0]
	job, err := QRGenerationJobFromMap(msg.Values)
	if err != nil {
		log.Printf("Failed to parse QR generation job from message %s: %v", msg.ID, err)
		// Ack invalid message to prevent reprocessing
		q.client.XAck(ctx, QRGenStreamQueue, QRGenConsumerGroup, msg.ID)
		return nil, ErrInvalidJobData
	}

	job.StreamID = msg.ID
	return job, nil
}

// Ack acknowledges successful processing of a job
func (q *RedisQRGenerationQueue) Ack(ctx context.Context, job *QRGenerationJob) error {
	if job.StreamID == "" {
		return ErrJobNotFound
	}

	if err := q.client.XAck(ctx, QRGenStreamQueue, QRGenConsumerGroup, job.StreamID).Err(); err != nil {
		return fmt.Errorf("failed to ack QR generation job %s: %w", job.ID, err)
	}

	// Mark as processed for deduplication
	dedupKey := QRGenJobDedupPrefix + job.ID
	q.client.Set(ctx, dedupKey, "1", QRGenJobDedupTTL)

	// Remove message from stream (cleanup)
	q.client.XDel(ctx, QRGenStreamQueue, job.StreamID)

	return nil
}

// Nack handles failed processing; requeues if retries remain, else moves to DLQ
func (q *RedisQRGenerationQueue) Nack(ctx context.Context, job *QRGenerationJob, requeue bool) error {
	if job.StreamID == "" {
		return ErrJobNotFound
	}

	if !requeue || !job.CanRetry() {
		return q.MoveToDLQ(ctx, job, job.LastError)
	}

	// Enqueue-then-ack: re-add the job to the stream FIRST, and only remove the
	// original message from the stream + PEL once the re-enqueue is confirmed.
	// If Enqueue fails (e.g. ErrCircuitOpen when the breaker just tripped, or a
	// Redis error), the original message stays in the pending-entries list so
	// ClaimStaleJobs / a redelivery can still recover it with its retry state,
	// instead of being silently dropped.
	oldStreamID := job.StreamID
	job.StreamID = "" // Clear old stream ID so Enqueue assigns new one
	if err := q.Enqueue(ctx, job); err != nil {
		job.StreamID = oldStreamID // restore so the caller/PEL keeps a handle on it
		return err
	}

	// Re-enqueue succeeded — safe to drop the original message.
	q.client.XAck(ctx, QRGenStreamQueue, QRGenConsumerGroup, oldStreamID)
	q.client.XDel(ctx, QRGenStreamQueue, oldStreamID)

	return nil
}

// MoveToDLQ moves a job to the dead letter queue
func (q *RedisQRGenerationQueue) MoveToDLQ(ctx context.Context, job *QRGenerationJob, reason string) error {
	if reason != "" {
		job.LastError = reason
	}

	data := job.ToMap()
	data["dlq_reason"] = reason
	data["dlq_time"] = time.Now().UTC().Format(time.RFC3339)

	args := &redis.XAddArgs{
		Stream: QRGenStreamDLQ,
		MaxLen: q.maxLen,
		Approx: true,
		Values: data,
	}

	if _, err := q.client.XAdd(ctx, args).Result(); err != nil {
		log.Printf("Failed to move QR generation job %s to DLQ: %v", job.ID, err)
		return err
	}

	// Ack and remove from original stream
	if job.StreamID != "" {
		q.client.XAck(ctx, QRGenStreamQueue, QRGenConsumerGroup, job.StreamID)
		q.client.XDel(ctx, QRGenStreamQueue, job.StreamID)
	}

	log.Printf("[QRGenQueue] Moved job %s (batch %s) to DLQ: %s", job.ID, job.BatchID, reason)
	return nil
}

// AcquireTenantLock attempts to acquire an exclusive lock for a tenant
// Returns (true, nil) if lock acquired, (false, nil) if another job is already running
func (q *RedisQRGenerationQueue) AcquireTenantLock(ctx context.Context, tenantID, workerID string) (bool, error) {
	key := QRGenTenantLockPrefix + tenantID
	ok, err := q.client.SetNX(ctx, key, workerID, q.tenantLockTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire tenant lock: %w", err)
	}
	return ok, nil
}

// RefreshTenantLock extends the TTL of the tenant lock (for long-running jobs)
// Only refreshes if the lock is still held by the same worker
func (q *RedisQRGenerationQueue) RefreshTenantLock(ctx context.Context, tenantID, workerID string) error {
	key := QRGenTenantLockPrefix + tenantID

	// Lua script: only refresh if value matches (we own the lock)
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	ttlSeconds := int(q.tenantLockTTL.Seconds())
	res, err := q.client.Eval(ctx, script, []string{key}, workerID, ttlSeconds).Result()
	if err != nil {
		return fmt.Errorf("failed to refresh tenant lock: %w", err)
	}
	// The script returns 0 (not an error) when the key is missing or owned by a
	// different worker — i.e. the lock expired or was stolen. Surface that as an
	// error so the worker's refresh goroutine can abort the job (see processJob's
	// lockLostCh handling) instead of continuing to generate under a lost lock.
	if n, ok := res.(int64); ok && n == 0 {
		return fmt.Errorf("tenant lock for %s no longer held by worker %s", tenantID, workerID)
	}
	return nil
}

// ReleaseTenantLock releases a tenant lock (only if we own it)
func (q *RedisQRGenerationQueue) ReleaseTenantLock(ctx context.Context, tenantID, workerID string) error {
	key := QRGenTenantLockPrefix + tenantID

	// Lua script: only delete if value matches
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := q.client.Eval(ctx, script, []string{key}, workerID).Result()
	if err != nil {
		return fmt.Errorf("failed to release tenant lock: %w", err)
	}
	return nil
}

// IsDuplicate checks if a job was already processed
func (q *RedisQRGenerationQueue) IsDuplicate(ctx context.Context, jobID string) bool {
	key := QRGenJobDedupPrefix + jobID
	exists, err := q.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

// ClaimStaleJobs reclaims jobs from workers that have died
func (q *RedisQRGenerationQueue) ClaimStaleJobs(ctx context.Context, workerID string, minIdleTime time.Duration) ([]*QRGenerationJob, error) {
	var jobs []*QRGenerationJob

	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: QRGenStreamQueue,
		Group:  QRGenConsumerGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()
	if err != nil {
		return nil, err
	}

	for _, p := range pending {
		if p.Idle < minIdleTime {
			continue
		}

		claimed, err := q.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   QRGenStreamQueue,
			Group:    QRGenConsumerGroup,
			Consumer: workerID,
			MinIdle:  minIdleTime,
			Messages: []string{p.ID},
		}).Result()
		if err != nil {
			log.Printf("Failed to claim stale QR generation message %s: %v", p.ID, err)
			continue
		}

		for _, msg := range claimed {
			job, err := QRGenerationJobFromMap(msg.Values)
			if err != nil {
				log.Printf("Failed to parse claimed QR generation message %s: %v", msg.ID, err)
				continue
			}
			job.StreamID = msg.ID
			jobs = append(jobs, job)
		}
	}

	if len(jobs) > 0 {
		log.Printf("[QRGenQueue] Claimed %d stale jobs", len(jobs))
	}

	return jobs, nil
}

// GetStats returns queue statistics
func (q *RedisQRGenerationQueue) GetStats(ctx context.Context) (*QRGenStats, error) {
	stats := &QRGenStats{}

	queueLen, err := q.client.XLen(ctx, QRGenStreamQueue).Result()
	if err == nil {
		stats.Pending = queueLen
	}

	dlqLen, err := q.client.XLen(ctx, QRGenStreamDLQ).Result()
	if err == nil {
		stats.DLQCount = dlqLen
	}

	pending, err := q.client.XPending(ctx, QRGenStreamQueue, QRGenConsumerGroup).Result()
	if err == nil {
		stats.Processing = pending.Count
	}

	return stats, nil
}

// GetCircuitBreaker returns the circuit breaker for monitoring
func (q *RedisQRGenerationQueue) GetCircuitBreaker() *CircuitBreaker {
	return q.circuitBreaker
}

// QRGenStats holds statistics about the QR generation queue
type QRGenStats struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	DLQCount   int64 `json:"dlq_count"`
}
