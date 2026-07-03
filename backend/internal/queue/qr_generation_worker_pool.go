package queue

import (
	"context"
	"fmt"
	"log"
	"sync"

	"gorm.io/gorm"
)

// QRGenerationWorkerPool manages a pool of QR generation workers
type QRGenerationWorkerPool struct {
	mu           sync.RWMutex
	workers      []*QRGenerationWorker
	queue        *RedisQRGenerationQueue
	db           *gorm.DB
	numWorkers   int
	workerConfig QRGenerationWorkerConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	started bool
	stopped bool
}

// QRGenerationWorkerPoolConfig holds configuration for the worker pool
type QRGenerationWorkerPoolConfig struct {
	NumWorkers   int
	WorkerConfig QRGenerationWorkerConfig
}

// DefaultQRGenerationWorkerPoolConfig returns default configuration
func DefaultQRGenerationWorkerPoolConfig() QRGenerationWorkerPoolConfig {
	return QRGenerationWorkerPoolConfig{
		NumWorkers:   2,
		WorkerConfig: DefaultQRGenerationWorkerConfig(),
	}
}

// NewQRGenerationWorkerPool creates a new worker pool
func NewQRGenerationWorkerPool(queue *RedisQRGenerationQueue, db *gorm.DB, cfg QRGenerationWorkerPoolConfig) *QRGenerationWorkerPool {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 2
	}

	return &QRGenerationWorkerPool{
		queue:        queue,
		db:           db,
		numWorkers:   cfg.NumWorkers,
		workerConfig: cfg.WorkerConfig,
		workers:      make([]*QRGenerationWorker, 0, cfg.NumWorkers),
	}
}

// Start starts all workers in the pool
func (wp *QRGenerationWorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	if wp.started {
		wp.mu.Unlock()
		return fmt.Errorf("QR generation worker pool already started")
	}
	wp.started = true
	wp.ctx, wp.cancel = context.WithCancel(ctx)
	wp.mu.Unlock()

	log.Printf("[QRGenWorkerPool] Starting %d workers", wp.numWorkers)

	for i := 0; i < wp.numWorkers; i++ {
		workerID := fmt.Sprintf("qrgen-worker-%d", i+1)
		worker := NewQRGenerationWorker(workerID, wp.queue, wp.db, wp.workerConfig)

		wp.mu.Lock()
		wp.workers = append(wp.workers, worker)
		wp.mu.Unlock()

		wp.wg.Add(1)
		go func(w *QRGenerationWorker) {
			defer wp.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[QRGenWorkerPool] Worker %s panic recovered: %v", w.ID(), r)
				}
			}()
			w.Start(wp.ctx)
		}(worker)
	}

	log.Printf("[QRGenWorkerPool] All %d workers started", wp.numWorkers)
	return nil
}

// Shutdown gracefully shuts down all workers
func (wp *QRGenerationWorkerPool) Shutdown(ctx context.Context) error {
	wp.mu.Lock()
	if !wp.started || wp.stopped {
		wp.mu.Unlock()
		return nil
	}
	wp.stopped = true
	wp.mu.Unlock()

	log.Printf("[QRGenWorkerPool] Shutting down %d workers...", len(wp.workers))

	wp.cancel()

	wp.mu.RLock()
	for _, worker := range wp.workers {
		worker.Stop()
	}
	wp.mu.RUnlock()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[QRGenWorkerPool] All workers stopped gracefully")
		return nil
	case <-ctx.Done():
		log.Printf("[QRGenWorkerPool] Shutdown timeout, some workers may not have finished")
		return ctx.Err()
	}
}

// IsHealthy checks if the worker pool is healthy
func (wp *QRGenerationWorkerPool) IsHealthy() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if !wp.started || wp.stopped {
		return false
	}

	healthy := 0
	for _, worker := range wp.workers {
		state := worker.State()
		if state == WorkerStateIdle || state == WorkerStateProcessing {
			healthy++
		}
	}

	return healthy >= wp.numWorkers/2
}
