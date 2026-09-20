package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
)

// Config represents application configuration loaded from environment variables
type Config struct {
	Profile string

	AppEnv     string
	AppPort    int
	AppURL     string
	ConsoleURL string

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

	masterKeyStr := getEnv("MVX_MASTER_KEY", "")
	if masterKeyStr != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(masterKeyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid MVX_MASTER_KEY: base64 decode failed: %w", err)
		}
		if len(keyBytes) != 32 {
			return nil, fmt.Errorf("invalid MVX_MASTER_KEY: expected 32 bytes, got %d", len(keyBytes))
		}
		cfg.MasterKey = keyBytes
	} else if cfg.AppEnv == "production" {
		return nil, fmt.Errorf("MVX_MASTER_KEY is mandatory in production")
	} else {
		// Sane 32-byte fallback for test/dev environment
		cfg.MasterKey = []byte("this-is-a-32-byte-dev-master-key!")
	}

	return cfg, nil
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
