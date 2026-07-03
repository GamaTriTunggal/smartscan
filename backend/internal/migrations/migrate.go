package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/gamatritunggal/smartscan/backend/internal/logger"
)

// migrationLockID is a PostgreSQL advisory lock ID used to prevent
// concurrent migration execution (e.g., during rolling deployments).
const migrationLockID int64 = 5737100 // arbitrary constant

// Run applies all pending database migrations. Uses a PostgreSQL advisory
// lock so multiple backend instances starting simultaneously don't race.
func Run(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if _, err := db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		if _, err := db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", migrationLockID); err != nil {
			logger.Warn("Failed to release migration lock", "error", err)
		}
	}()

	before, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return fmt.Errorf("get db version: %w", err)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	after, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return fmt.Errorf("get db version after migrate: %w", err)
	}

	if after != before {
		logger.Info("Database migrations applied", "from_version", before, "to_version", after)
	} else {
		logger.Info("Database schema up to date", "version", after)
	}
	return nil
}
