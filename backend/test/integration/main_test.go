package integration

import (
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=postgres port=5432 user=smartscan password=smartscan dbname=smartscan_test sslmode=disable TimeZone=UTC"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Exit(0) // Skip if DB not available
	}

	testDB = db
	os.Exit(m.Run())
}

func getTestConfig() *config.Config {
	return &config.Config{
		AppEnv:      "test",
		FrontendURL: "http://localhost:3000",
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-signing",
			ExpirationHours: 24,
			RefreshHours:    168,
		},
	}
}
