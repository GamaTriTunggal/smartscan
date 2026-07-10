package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/gamatritunggal/smartscan/backend/internal/migrations"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func main() {
	// Load .env
	_ = godotenv.Load("../../.env") // Root .env (when running from backend/cmd/migrate/)
	_ = godotenv.Load("../.env")    // backend/.env
	_ = godotenv.Load()             // Current directory

	dsn := buildDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	goose.SetBaseFS(migrations.EmbedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set dialect: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := "status"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		err = goose.UpContext(ctx, db, ".")
	case "down":
		err = goose.DownContext(ctx, db, ".")
	case "status":
		err = goose.StatusContext(ctx, db, ".")
	case "version":
		err = goose.VersionContext(ctx, db, ".")
	case "redo":
		if err = goose.DownContext(ctx, db, "."); err == nil {
			err = goose.UpByOneContext(ctx, db, ".")
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		fmt.Fprintf(os.Stderr, "Available commands: up, down, status, version, redo\n")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "smartscan")
	password := getEnv("DB_PASSWORD", "smartscan")
	dbname := getEnv("DB_NAME", "smartscan")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
