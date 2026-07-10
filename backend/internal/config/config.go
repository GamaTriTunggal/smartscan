package config

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// MinJWTSecretLength is the minimum recommended length for JWT secret
// OWASP recommends at least 256 bits (32 bytes) for HMAC-SHA256
const MinJWTSecretLength = 32

type Config struct {
	AppEnv              string
	ServerPort          string
	DB                  DatabaseConfig
	Redis               RedisConfig
	JWT                 JWTConfig
	QRGeneration        QRGenerationConfig
	Log                 LogConfig
	Sentry              SentryConfig
	R2                  R2Config
	Geocoding           GeocodingConfig
	UploadPath          string
	FrontendURL         string
	ScanSignatureSecret string // HMAC secret for scan URL signature verification
	// TrustedProxies is the list of proxy CIDRs/IPs Gin will trust when deriving
	// the client IP from X-Forwarded-For / X-Real-IP. Empty means trust NONE (the
	// app is exposed directly, so ClientIP() uses the real socket RemoteAddr and
	// the header is ignored). This prevents rate-limit keys from being spoofed via
	// a forged X-Forwarded-For header. Configure via TRUSTED_PROXIES (comma-separated)
	// to the CIDR of your reverse proxy when deployed behind one.
	TrustedProxies []string
}

// IsProduction reports whether the app is running in a production-like environment.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production" || c.AppEnv == "prod"
}

// R2Config holds configuration for Cloudflare R2 storage
type R2Config struct {
	Enabled     bool   // Enable R2 storage (vs local filesystem)
	AccountID   string // Cloudflare Account ID
	AccessKeyID string // R2 API Access Key ID
	SecretKey   string // R2 API Secret Access Key
	BucketName  string // R2 Bucket name
	PublicURL   string // Public URL for the bucket (e.g., https://cdn-dev.smartscan.com)
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret                   string
	ExpirationHours          int
	RefreshHours             int
	InactivityTimeoutMinutes int // Session timeout after inactivity (0 = disabled)
}

// QRGenerationConfig holds configuration for the async QR generation system
type QRGenerationConfig struct {
	Enabled           bool          // Enable async QR generation (vs sync fallback)
	NumWorkers        int           // Number of worker goroutines (default: 2)
	MaxBatchLimit     int           // Maximum QR codes per batch (default: 5,000,000)
	ChunkSize         int           // QR codes per INSERT (default: 1000)
	MaxRetries        int           // Max retries before DLQ (default: 5)
	ScannerInterval   time.Duration // How often scanner checks for stuck batches (default: 30s)
	StuckThreshold    time.Duration // How long a processing batch can be idle before considered stuck (default: 10m)
	VisibilityTimeout time.Duration // Redis stream visibility timeout (default: 10m)
	PollInterval      time.Duration // How often workers poll for jobs (default: 1s)
	TenantLockTTL     time.Duration // TTL for per-tenant concurrent lock (default: 15m)
	MaxStreamLength   int64         // Max jobs in Redis stream (default: 10000)
	PDFExportMaxCodes int           // Max codes per PDF export file (default 10000)
}

// GeocodingConfig holds configuration for reverse geocoding services
type GeocodingConfig struct {
	BigDataCloudAPIKey string // BigDataCloud API key for server-side reverse geocoding
}

// SentryConfig holds configuration for Sentry/GlitchTip error tracking
type SentryConfig struct {
	DSN              string  // Sentry/GlitchTip DSN URL
	Environment      string  // Environment name (development, staging, production)
	Release          string  // Application version/release
	Debug            bool    // Enable debug mode
	SampleRate       float64 // Error sample rate (1.0 = 100%)
	TracesSampleRate float64 // Performance traces sample rate (0-1.0)
	GlitchTipDomain  string  // GlitchTip domain for Host header (e.g., http://localhost:8001)
}

func Load() *Config {
	return &Config{
		AppEnv:     getEnv("APP_ENV", "development"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "smartscan"),
			Password: getDatabasePassword(),
			DBName:   getEnv("DB_NAME", "smartscan"),
			SSLMode:  getDatabaseSSLMode(),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:                   getJWTSecret(),
			ExpirationHours:          getEnvInt("JWT_EXPIRATION_HOURS", 24),
			RefreshHours:             getEnvInt("JWT_REFRESH_HOURS", 168),
			InactivityTimeoutMinutes: getEnvInt("INACTIVITY_TIMEOUT_MINUTES", 30),
		},
		QRGeneration: QRGenerationConfig{
			Enabled:           getEnv("QR_GENERATION_ENABLED", "true") == "true",
			NumWorkers:        getEnvInt("QR_GENERATION_WORKERS", 2),
			MaxBatchLimit:     getEnvInt("QR_BATCH_MAX_LIMIT", 5000000),
			ChunkSize:         getEnvInt("QR_GENERATION_CHUNK_SIZE", 1000),
			MaxRetries:        getEnvInt("QR_GENERATION_MAX_RETRIES", 5),
			ScannerInterval:   getEnvDuration("QR_GENERATION_SCANNER_INTERVAL", 30*time.Second),
			StuckThreshold:    getEnvDuration("QR_GENERATION_STUCK_THRESHOLD", 10*time.Minute),
			VisibilityTimeout: getEnvDuration("QR_GENERATION_VISIBILITY_TIMEOUT", 10*time.Minute),
			PollInterval:      getEnvDuration("QR_GENERATION_POLL_INTERVAL", time.Second),
			TenantLockTTL:     getEnvDuration("QR_GENERATION_TENANT_LOCK_TTL", 15*time.Minute),
			MaxStreamLength:   int64(getEnvInt("QR_GENERATION_MAX_STREAM_LENGTH", 10000)),
			PDFExportMaxCodes: getEnvInt("PDF_EXPORT_MAX_CODES", 10000),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"), // json for production, text for development readability
		},
		Sentry: SentryConfig{
			DSN:              getEnv("SENTRY_DSN", ""),
			Environment:      getEnv("APP_ENV", "development"),
			Release:          getEnv("SENTRY_RELEASE", "smartscan@0.1.0"),
			Debug:            getEnv("SENTRY_DEBUG", "false") == "true",
			SampleRate:       getEnvFloat("SENTRY_SAMPLE_RATE", 1.0),
			TracesSampleRate: getEnvFloat("SENTRY_TRACES_SAMPLE_RATE", 0.1),
			GlitchTipDomain:  getEnv("GLITCHTIP_DOMAIN", ""),
		},
		Geocoding: GeocodingConfig{
			BigDataCloudAPIKey: getEnv("BIGDATACLOUD_API_KEY", ""),
		},
		R2: R2Config{
			Enabled:     getEnv("R2_ENABLED", "false") == "true",
			AccountID:   getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID: getEnv("R2_ACCESS_KEY_ID", ""),
			SecretKey:   getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:  getEnv("R2_BUCKET_NAME", ""),
			PublicURL:   getEnv("R2_PUBLIC_URL", ""),
		},
		UploadPath:          getEnv("UPLOAD_PATH", "./uploads"),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:3000"),
		ScanSignatureSecret: getScanSignatureSecret(),
		TrustedProxies:      getEnvList("TRUSTED_PROXIES"),
	}
}

// getEnvList parses a comma-separated env var into a trimmed, non-empty slice.
// Returns nil when unset/empty.
func getEnvList(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getJWTSecret retrieves and validates the JWT secret
// In production, it requires a strong secret from environment variable
// In development, it can use a default or generate one
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	env := getEnv("APP_ENV", "development")
	isProduction := env == "production" || env == "prod"

	// Check if JWT_SECRET is set
	if secret == "" {
		if isProduction {
			// In production, JWT_SECRET is required
			log.Fatal("SECURITY ERROR: JWT_SECRET environment variable is required in production")
		}
		// In development, generate a random secret
		secret = generateRandomSecret()
		log.Printf("WARNING: No JWT_SECRET set, using generated secret (not suitable for production)")
		return secret
	}

	// Validate secret strength
	if len(secret) < MinJWTSecretLength {
		if isProduction {
			log.Fatalf("SECURITY ERROR: JWT_SECRET must be at least %d characters in production (got %d)", MinJWTSecretLength, len(secret))
		}
		log.Printf("WARNING: JWT_SECRET is too short (%d chars). Minimum %d chars recommended for security", len(secret), MinJWTSecretLength)
	}

	// Check for known weak secrets
	weakSecrets := []string{
		"smartscan-dev-jwt-secret-change-in-production",
		"secret",
		"jwt-secret",
		"your-secret-key",
	}
	for _, weak := range weakSecrets {
		if secret == weak {
			if isProduction {
				log.Fatal("SECURITY ERROR: Using a known weak JWT_SECRET in production is not allowed")
			}
			log.Printf("WARNING: Using a weak JWT_SECRET. Change this before deploying to production")
			break
		}
	}

	return secret
}

// generateRandomSecret generates a cryptographically secure random secret
func generateRandomSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate random secret")
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// getDatabasePassword retrieves and validates the database password
// In production, it requires a strong password from environment variable
// In development, it can use a default
func getDatabasePassword() string {
	password := os.Getenv("DB_PASSWORD")
	env := getEnv("APP_ENV", "development")
	isProduction := env == "production" || env == "prod"

	// Check if DB_PASSWORD is set
	if password == "" {
		if isProduction {
			log.Fatal("SECURITY ERROR: DB_PASSWORD environment variable is required in production")
		}
		password = "smartscan"
		log.Printf("WARNING: No DB_PASSWORD set, using default (not suitable for production)")
		return password
	}

	// Check for known weak passwords
	weakPasswords := []string{
		"smartscan",
		"password",
		"postgres",
		"admin",
		"123456",
	}
	for _, weak := range weakPasswords {
		if password == weak {
			if isProduction {
				log.Fatal("SECURITY ERROR: Using a known weak DB_PASSWORD in production is not allowed")
			}
			log.Printf("WARNING: Using a weak DB_PASSWORD. Change this before deploying to production")
			break
		}
	}

	return password
}

// getDatabaseSSLMode retrieves and validates the database SSL mode
// In production, it requires SSL to be enabled (not "disable")
// In development, it can use "disable" for convenience
func getDatabaseSSLMode() string {
	sslMode := os.Getenv("DB_SSLMODE")
	env := getEnv("APP_ENV", "development")
	isProduction := env == "production" || env == "prod"

	// Default to disable in development
	if sslMode == "" {
		if isProduction {
			// In production, default to require SSL
			sslMode = "require"
			log.Printf("WARNING: No DB_SSLMODE set, defaulting to 'require' for production security")
		} else {
			sslMode = "disable"
		}
		return sslMode
	}

	// In production, SSL must be enabled unless explicitly using internal Docker network
	if isProduction && sslMode == "disable" {
		// Allow disable for internal Docker networks where traffic never leaves the host
		// This is safe because containers communicate over isolated Docker bridge network
		if os.Getenv("DB_SSL_INTERNAL_NETWORK") == "true" {
			log.Printf("INFO: DB SSL disabled for internal Docker network (DB_SSL_INTERNAL_NETWORK=true)")
		} else {
			log.Fatal("SECURITY ERROR: DB_SSLMODE cannot be 'disable' in production. Use 'require', 'verify-ca', or 'verify-full'. Set DB_SSL_INTERNAL_NETWORK=true if using internal Docker network.")
		}
	}

	return sslMode
}

// GetAllowedOrigins returns validated CORS origins based on environment
// In production: only FRONTEND_URL is allowed (no localhost)
// In development: localhost:3000 + FRONTEND_URL are allowed
func (c *Config) GetAllowedOrigins() []string {
	isProduction := c.AppEnv == "production" || c.AppEnv == "prod"
	origins := []string{}

	// Validate and add FRONTEND_URL
	frontendURL := c.FrontendURL
	if frontendURL != "" {
		// Basic URL validation
		if !isValidOrigin(frontendURL) {
			if isProduction {
				log.Fatalf("SECURITY ERROR: FRONTEND_URL '%s' is not a valid origin", frontendURL)
			}
			log.Printf("WARNING: FRONTEND_URL '%s' is not a valid origin, skipping", frontendURL)
		} else {
			origins = append(origins, frontendURL)
		}
	} else if isProduction {
		log.Fatal("SECURITY ERROR: FRONTEND_URL is required in production for CORS")
	}

	// In development, also allow localhost for convenience
	if !isProduction {
		// Add common development origins
		devOrigins := []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
		for _, devOrigin := range devOrigins {
			// Don't add duplicates
			isDuplicate := false
			for _, existing := range origins {
				if existing == devOrigin {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				origins = append(origins, devOrigin)
			}
		}
	}

	if len(origins) == 0 {
		log.Fatal("SECURITY ERROR: No valid CORS origins configured")
	}

	log.Printf("CORS: Allowed origins: %v", origins)
	return origins
}

// isValidOrigin checks if a string is a valid HTTP(S) origin
func isValidOrigin(origin string) bool {
	// Must start with http:// or https://
	if len(origin) < 8 {
		return false
	}
	if origin[:7] != "http://" && origin[:8] != "https://" {
		return false
	}
	// Must not end with a trailing slash
	if origin[len(origin)-1] == '/' {
		return false
	}
	// Must not contain path (only scheme://host:port allowed)
	// Count slashes - should be exactly 2 (for ://)
	slashCount := 0
	for _, c := range origin {
		if c == '/' {
			slashCount++
		}
	}
	if slashCount != 2 {
		return false
	}
	return true
}

// getScanSignatureSecret retrieves the HMAC secret for scan URL signatures
// In production, it requires the secret from environment variable
// In development, it can use a default
func getScanSignatureSecret() string {
	secret := os.Getenv("SCAN_SIGNATURE_SECRET")
	env := getEnv("APP_ENV", "development")
	isProduction := env == "production" || env == "prod"

	if secret == "" {
		if isProduction {
			log.Fatal("SECURITY ERROR: SCAN_SIGNATURE_SECRET environment variable is required in production")
		}
		// Generate a random secret for development
		secret = generateRandomSecret()
		log.Printf("WARNING: No SCAN_SIGNATURE_SECRET set, using generated secret (not suitable for production)")
	}

	return secret
}
