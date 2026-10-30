package config

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Config represents application configuration loaded from environment variables
type Config struct {
	Profile string

	AppEnv     string
	AppPort    int
	AppURL     string
	ConsoleURL string
	DataDir    string

	MasterKey   []byte
	KEKProvider string

	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	DBName         string
	DBMaxOpenConns int
	DBMaxIdleConns int

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	SMTPHostname       string
	SMTPSubmissionPort int
	SMTPSMTPSPort      int
	SMTPInboundPort    int

	SystemMailMode string
	SystemMailFrom string

	LogLevel       string
	LogFormat      string
	LogRedactEmail bool
}

// Load loads and validates configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Profile:            getEnv("MVX_PROFILE", "standard"),
		AppEnv:             getEnv("MVX_APP_ENV", "development"),
		AppPort:            getEnvInt("MVX_APP_PORT", 8080),
		AppURL:             getEnv("MVX_APP_URL", "http://localhost:8080"),
		ConsoleURL:         getEnv("MVX_CONSOLE_URL", "http://localhost:3000"),
		DataDir:            getEnv("MVX_DATA_DIR", "/var/lib/mailvex"),
		KEKProvider:        getEnv("MVX_KEK_PROVIDER", "env"),
		DBHost:             getEnv("MVX_DB_HOST", "127.0.0.1"),
		DBPort:             getEnvInt("MVX_DB_PORT", 3306),
		DBUser:             getEnv("MVX_DB_USER", "mailvex"),
		DBPassword:         getEnv("MVX_DB_PASSWORD", "mailvex_secret"),
		DBName:             getEnv("MVX_DB_NAME", "mailvex"),
		DBMaxOpenConns:     getEnvInt("MVX_DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:     getEnvInt("MVX_DB_MAX_IDLE_CONNS", 10),
		RedisAddr:          getEnv("MVX_REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:      getEnv("MVX_REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("MVX_REDIS_DB", 0),
		SMTPHostname:       getEnv("MVX_SMTP_HOSTNAME", "localhost"),
		SMTPSubmissionPort: getEnvInt("MVX_SMTP_SUBMISSION_PORT", 587),
		SMTPSMTPSPort:      getEnvInt("MVX_SMTP_SMTPS_PORT", 465),
		SMTPInboundPort:    getEnvInt("MVX_SMTP_INBOUND_PORT", 25),
		SystemMailMode:     getEnv("MVX_SYSTEM_MAIL_MODE", "direct"),
		SystemMailFrom:     getEnv("MVX_SYSTEM_MAIL_FROM", "system@mail.example.com"),
		LogLevel:           getEnv("MVX_LOG_LEVEL", "info"),
		LogFormat:          getEnv("MVX_LOG_FORMAT", "json"),
		LogRedactEmail:     getEnvBool("MVX_LOG_REDACT_EMAIL", true),
	}

	masterKey, err := parseMasterKey(cfg.AppEnv)
	if err != nil {
		return nil, err
	}
	cfg.MasterKey = masterKey

	// Parse database and redis URLs if provided and specific fields are default
	applyDatabaseURL(cfg)
	applyRedisURL(cfg)

	return cfg, nil
}

func parseMasterKey(appEnv string) ([]byte, error) {
	keyVar := "MVX_MASTER_KEY"
	val := getEnv("MVX_MASTER_KEY", "")
	if val == "" {
		keyVar = "MVX_KEK"
		val = getEnv("MVX_KEK", "")
	}
	if val == "" {
		keyVar = "MAILVEX_MASTER_KEK"
		val = getEnv("MAILVEX_MASTER_KEK", "")
	}

	if val == "" {
		if appEnv == "production" {
			return nil, fmt.Errorf("master key is mandatory in production: set MVX_MASTER_KEY (32 bytes base64 encoded) or MVX_KEK (64 hex characters)")
		}
		// Sane 32-byte fallback for test/dev environment
		return []byte("mailvex-homelab-32byte-masterkek"), nil
	}

	val = strings.TrimSpace(val)

	// 1. Try 64-char Hex (64 hex characters = 32 bytes)
	if len(val) == 64 {
		if hexBytes, err := hex.DecodeString(val); err == nil && len(hexBytes) == 32 {
			return hexBytes, nil
		}
	}

	// 2. Try Base64 (Standard or URL/Raw)
	if b64Bytes, err := base64.StdEncoding.DecodeString(val); err == nil {
		if len(b64Bytes) == 32 {
			return b64Bytes, nil
		}
		return nil, fmt.Errorf("invalid %s: base64 decoded to %d bytes, expected 32 (AES-256)", keyVar, len(b64Bytes))
	}
	if b64Bytes, err := base64.RawStdEncoding.DecodeString(val); err == nil {
		if len(b64Bytes) == 32 {
			return b64Bytes, nil
		}
		return nil, fmt.Errorf("invalid %s: base64 decoded to %d bytes, expected 32 (AES-256)", keyVar, len(b64Bytes))
	}

	// 3. Raw 32-byte string
	if len(val) == 32 {
		return []byte(val), nil
	}

	return nil, fmt.Errorf("invalid %s: expected 32-byte key (base64, 64-char hex, or 32 raw bytes), got %d characters", keyVar, len(val))
}

var dsnPattern = regexp.MustCompile(`^(?:([^:]+)(?::([^@]*))?@)?(?:tcp\(([^:)]+)(?::(\d+))?\))?\/([^?]+)`)

func applyDatabaseURL(cfg *Config) {
	dbURL := getEnv("MVX_DATABASE_URL", "")
	if dbURL == "" {
		return
	}
	matches := dsnPattern.FindStringSubmatch(dbURL)
	if matches != nil {
		if matches[1] != "" && os.Getenv("MVX_DB_USER") == "" {
			cfg.DBUser = matches[1]
		}
		if matches[2] != "" && os.Getenv("MVX_DB_PASSWORD") == "" {
			cfg.DBPassword = matches[2]
		}
		if matches[3] != "" && os.Getenv("MVX_DB_HOST") == "" {
			cfg.DBHost = matches[3]
		}
		if matches[4] != "" && os.Getenv("MVX_DB_PORT") == "" {
			if port, err := strconv.Atoi(matches[4]); err == nil {
				cfg.DBPort = port
			}
		}
		if matches[5] != "" && os.Getenv("MVX_DB_NAME") == "" {
			cfg.DBName = matches[5]
		}
	}
}

func applyRedisURL(cfg *Config) {
	redisURL := getEnv("MVX_REDIS_URL", "")
	if redisURL != "" && os.Getenv("MVX_REDIS_ADDR") == "" {
		redisURL = strings.TrimPrefix(redisURL, "redis://")
		cfg.RedisAddr = redisURL
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
