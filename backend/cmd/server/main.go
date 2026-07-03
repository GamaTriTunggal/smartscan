package main

// Fixed tenant_printing.go handlers - tenantID type assertion bug

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/database"
	"github.com/gamatritunggal/smartscan/backend/internal/handlers"
	"github.com/gamatritunggal/smartscan/backend/internal/logger"
	"github.com/gamatritunggal/smartscan/backend/internal/migrations"
	"github.com/gamatritunggal/smartscan/backend/internal/queue"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/storage"
)

func main() {
	// Load .env file - try root first (for local dev), then current dir
	// Docker passes env vars directly, so this is mainly for local development
	_ = godotenv.Load("../.env") // Root .env (when running from backend/)
	_ = godotenv.Load()          // Current directory .env (fallback)

	// Load configuration
	cfg := config.Load()

	// Initialize structured logger
	logger.Init(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})

	logger.Info("Starting smartscan server",
		"version", "1.0.0",
		"env", cfg.AppEnv,
		"log_level", cfg.Log.Level,
		"log_format", cfg.Log.Format,
	)

	// Initialize Sentry/GlitchTip error tracking
	if err := sentry.Init(sentry.Config{
		DSN:              cfg.Sentry.DSN,
		Environment:      cfg.Sentry.Environment,
		Release:          cfg.Sentry.Release,
		Debug:            cfg.Sentry.Debug,
		SampleRate:       cfg.Sentry.SampleRate,
		TracesSampleRate: cfg.Sentry.TracesSampleRate,
		GlitchTipDomain:  cfg.Sentry.GlitchTipDomain,
	}); err != nil {
		logger.Warn("Failed to initialize Sentry", "error", err)
	}
	defer sentry.Flush(2 * time.Second)

	// Initialize R2 storage if enabled
	if cfg.R2.Enabled {
		r2Cfg := storage.R2Config{
			AccountID:   cfg.R2.AccountID,
			AccessKeyID: cfg.R2.AccessKeyID,
			SecretKey:   cfg.R2.SecretKey,
			BucketName:  cfg.R2.BucketName,
			PublicURL:   cfg.R2.PublicURL,
		}
		r2Client, err := storage.NewR2Client(context.Background(), r2Cfg)
		if err != nil {
			logger.Fatal("Failed to initialize R2 storage", "error", err)
		}
		storage.SetGlobalR2Client(r2Client)
		logger.Info("R2 storage initialized",
			"bucket", cfg.R2.BucketName,
			"public_url", cfg.R2.PublicURL,
		)
	} else {
		logger.Info("R2 storage disabled, using local filesystem")
	}

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}

	// Run database migrations
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("Failed to get sql.DB for migrations", "error", err)
	}
	migCtx, migCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	if err := migrations.Run(migCtx, sqlDB); err != nil {
		migCancel()
		logger.Fatal("Database migration failed", "error", err)
	}
	migCancel()

	// Connect to Redis
	redisClient, err := database.ConnectRedis(cfg)
	if err != nil {
		logger.Warn("Failed to connect to Redis", "error", err)
	}

	// Initialize QR generation queue, worker pool, and scanner
	var qrGenQueue *queue.RedisQRGenerationQueue
	var qrGenWorkerPool *queue.QRGenerationWorkerPool
	var qrGenScanner *queue.QRGenerationScanner
	if redisClient != nil && cfg.QRGeneration.Enabled {
		qrGenQueueCfg := queue.RedisQRGenerationQueueConfig{
			MaxStreamLength: cfg.QRGeneration.MaxStreamLength,
			TenantLockTTL:   cfg.QRGeneration.TenantLockTTL,
			CircuitBreakerConfig: queue.CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Timeout:          time.Minute,
				HalfOpenMaxCalls: 1,
			},
		}
		q, err := queue.NewRedisQRGenerationQueue(redisClient, db, qrGenQueueCfg)
		if err != nil {
			logger.Warn("Failed to create QR generation queue", "error", err)
		} else {
			qrGenQueue = q
			handlers.SetQRGenerationQueue(q)

			// Worker pool
			poolCfg := queue.QRGenerationWorkerPoolConfig{
				NumWorkers: cfg.QRGeneration.NumWorkers,
				WorkerConfig: queue.QRGenerationWorkerConfig{
					ChunkSize:      cfg.QRGeneration.ChunkSize,
					PollInterval:   cfg.QRGeneration.PollInterval,
					VisibilityTime: cfg.QRGeneration.VisibilityTimeout,
				},
			}
			qrGenWorkerPool = queue.NewQRGenerationWorkerPool(q, db, poolCfg)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("QR generation worker pool panic recovered", "panic", r)
					}
				}()
				ctx := context.Background()
				if err := qrGenWorkerPool.Start(ctx); err != nil {
					logger.Warn("QR generation worker pool error", "error", err)
				}
			}()

			// Recovery scanner
			scannerCfg := queue.QRGenerationScannerConfig{
				Interval:       cfg.QRGeneration.ScannerInterval,
				StuckThreshold: cfg.QRGeneration.StuckThreshold,
			}
			qrGenScanner = queue.NewQRGenerationScanner(q, db, scannerCfg)
			if err := qrGenScanner.Start(context.Background()); err != nil {
				logger.Warn("Failed to start QR generation scanner", "error", err)
			}

			logger.Info("QR generation queue initialized",
				"workers", cfg.QRGeneration.NumWorkers,
				"max_batch_limit", cfg.QRGeneration.MaxBatchLimit,
			)
		}
	} else if !cfg.QRGeneration.Enabled {
		logger.Info("QR generation async queue disabled")
	}
	_ = qrGenQueue // referenced by handlers via SetQRGenerationQueue

	// Setup router
	router := setupRouter(db, cfg)

	// Create HTTP server for graceful shutdown support
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Graceful shutdown on signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful HTTP server shutdown with timeout
	shutdownHttpCtx, shutdownHttpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownHttpCancel()
	if err := srv.Shutdown(shutdownHttpCtx); err != nil {
		logger.Error("HTTP server forced to shutdown", "error", err)
	}
	logger.Info("HTTP server shutdown complete")

	// Stop QR generation scanner and worker pool
	if qrGenScanner != nil {
		logger.Info("Stopping QR generation scanner...")
		qrGenScanner.Stop()
	}
	if qrGenWorkerPool != nil {
		logger.Info("Stopping QR generation worker pool...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := qrGenWorkerPool.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error stopping QR generation worker pool", "error", err)
		}
	}


	// Close database connections
	if err := database.Close(); err != nil {
		logger.Error("Error closing database", "error", err)
	}
	if err := database.CloseRedis(); err != nil {
		logger.Error("Error closing Redis", "error", err)
	}

	logger.Info("Server stopped")
}
