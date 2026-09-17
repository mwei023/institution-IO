package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	Port        string
	Environment string // development | staging | production

	// Database
	DatabaseURL     string
	DBMaxConns      int32
	DBMinConns      int32
	DBMaxConnLife   time.Duration
	DBMaxConnIdle   time.Duration

	// Redis
	RedisURL      string
	RedisPassword string

	// JWT
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTAccessTTL      time.Duration  // 15 minutes
	JWTRefreshTTL     time.Duration  // 7 days

	// Encryption
	EncryptionKey string // 32-byte hex key for AES-256

	// Object Storage (MinIO / S3)
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool

	// M-Pesa Daraja
	MpesaConsumerKey    string
	MpesaConsumerSecret string
	MpesaShortcode      string
	MpesaPasskey        string
	MpesaCallbackURL    string
	MpesaCallbackSecret string // shared secret Daraja callback must send (X-Callback-Secret)
	MpesaEnvironment    string // sandbox | production

	// SMS (Africa's Talking)
	SMSAPIKey   string
	SMSUsername string
	SMSSenderID string

	// Email (SendGrid)
	SendGridAPIKey    string
	EmailFromAddress  string
	EmailFromName     string

	// Backup
	BackupEncryptionKey string
	BackupBucket        string

	// Logging
	LogLevel string

	// CORS
	AllowedOrigins []string
}

// Load reads configuration from environment variables.
// Call with an optional path to a .env file (ignored if file doesn't exist).
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		// Ignore error — .env file is optional in production
		_ = godotenv.Load(envFile)
	}

	// Pilot P0-4: missing required vars return an error instead of panic,
	// so main can log + exit(1) cleanly without a stacktrace.
	var missing []string
	req := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		return val
	}

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		DatabaseURL:       req("DATABASE_URL"),
		DBMaxConns:        int32(getEnvInt("DB_MAX_CONNS", 25)),
		DBMinConns:        int32(getEnvInt("DB_MIN_CONNS", 5)),
		DBMaxConnLife:     getEnvDuration("DB_MAX_CONN_LIFE", time.Hour),
		DBMaxConnIdle:     getEnvDuration("DB_MAX_CONN_IDLE", 30*time.Minute),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTPrivateKeyPath: req("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  req("JWT_PUBLIC_KEY_PATH"),
		JWTAccessTTL:      getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:     getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		EncryptionKey:     req("ENCRYPTION_KEY"),
		StorageEndpoint:   getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:  getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:  getEnv("STORAGE_SECRET_KEY", "minioadmin"),
		StorageBucket:     getEnv("STORAGE_BUCKET", "institution-os"),
		StorageUseSSL:     getEnvBool("STORAGE_USE_SSL", false),
		MpesaConsumerKey:    getEnv("MPESA_CONSUMER_KEY", ""),
		MpesaConsumerSecret: getEnv("MPESA_CONSUMER_SECRET", ""),
		MpesaShortcode:      getEnv("MPESA_SHORTCODE", "174379"),
		MpesaPasskey:        getEnv("MPESA_PASSKEY", ""),
		MpesaCallbackURL:    getEnv("MPESA_CALLBACK_URL", ""),
		MpesaCallbackSecret: getEnv("MPESA_CALLBACK_SECRET", ""),
		MpesaEnvironment:    getEnv("MPESA_ENVIRONMENT", "sandbox"),
		SMSAPIKey:           getEnv("SMS_API_KEY", ""),
		SMSUsername:         getEnv("SMS_USERNAME", "sandbox"),
		SMSSenderID:         getEnv("SMS_SENDER_ID", "InstitutionOS"),
		SendGridAPIKey:      getEnv("SENDGRID_API_KEY", ""),
		EmailFromAddress:    getEnv("EMAIL_FROM_ADDRESS", "noreply@institution-os.com"),
		EmailFromName:       getEnv("EMAIL_FROM_NAME", "Institution OS"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		AllowedOrigins:      []string{getEnv("ALLOWED_ORIGINS", "http://localhost:3000")},
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
