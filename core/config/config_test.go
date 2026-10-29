package config

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestConfigMasterKeyFallback(t *testing.T) {
	os.Unsetenv("MVX_MASTER_KEY")
	os.Unsetenv("MVX_KEK")
	os.Unsetenv("MAILVEX_MASTER_KEK")
	os.Setenv("MVX_APP_ENV", "development")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load to succeed with dev fallback, got: %v", err)
	}
	if len(cfg.MasterKey) != 32 {
		t.Fatalf("expected 32-byte fallback master key, got %d bytes", len(cfg.MasterKey))
	}
	if !bytes.Equal(cfg.MasterKey, []byte("mailvex-homelab-32byte-masterkek")) {
		t.Fatalf("unexpected fallback key: %s", string(cfg.MasterKey))
	}
}

func TestConfigMasterKeyProductionMandatory(t *testing.T) {
	os.Unsetenv("MVX_MASTER_KEY")
	os.Unsetenv("MVX_KEK")
	os.Unsetenv("MAILVEX_MASTER_KEK")
	os.Setenv("MVX_APP_ENV", "production")

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load to fail in production when master key is missing")
	}
	if !strings.Contains(err.Error(), "mandatory in production") {
		t.Fatalf("expected error mentioning mandatory in production, got: %v", err)
	}
}

func TestConfigMasterKeyBase64(t *testing.T) {
	os.Setenv("MVX_APP_ENV", "development")
	// Valid 32-byte key base64 encoded
	os.Setenv("MVX_MASTER_KEY", "bWFpbHZleC1ob21lbGFiLTMyYnl0ZS1tYXN0ZXJrZWs=")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load to succeed with base64 key, got: %v", err)
	}
	if len(cfg.MasterKey) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(cfg.MasterKey))
	}
}

func TestConfigMasterKeyInvalidBase64Length(t *testing.T) {
	os.Setenv("MVX_APP_ENV", "development")
	// Old .env.example string which decoded to 35 bytes
	os.Setenv("MVX_MASTER_KEY", "dGhpcy1pcy1hLXRlc3QtMzItYnl0ZS1tYXN0ZXIta2V5ISE=")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error with 35-byte base64 key, got nil")
	}
	if !strings.Contains(err.Error(), "base64 decoded to 35 bytes, expected 32") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestConfigMasterKeyHex(t *testing.T) {
	os.Unsetenv("MVX_MASTER_KEY")
	os.Setenv("MVX_APP_ENV", "development")
	// 64 hex characters (32 bytes)
	os.Setenv("MVX_KEK", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load to succeed with 64-char hex MVX_KEK, got: %v", err)
	}
	if len(cfg.MasterKey) != 32 {
		t.Fatalf("expected 32 bytes from hex key, got %d", len(cfg.MasterKey))
	}
}

func TestConfigDatabaseAndRedisURL(t *testing.T) {
	os.Unsetenv("MVX_DB_USER")
	os.Unsetenv("MVX_DB_PASSWORD")
	os.Unsetenv("MVX_DB_HOST")
	os.Unsetenv("MVX_DB_PORT")
	os.Unsetenv("MVX_DB_NAME")
	os.Unsetenv("MVX_REDIS_ADDR")

	os.Setenv("MVX_DATABASE_URL", "custom_user:custom_secret@tcp(db.internal:3307)/custom_db?parseTime=true")
	os.Setenv("MVX_REDIS_URL", "redis.internal:6380")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load to succeed, got: %v", err)
	}
	if cfg.DBUser != "custom_user" {
		t.Errorf("expected DBUser custom_user, got %s", cfg.DBUser)
	}
	if cfg.DBPassword != "custom_secret" {
		t.Errorf("expected DBPassword custom_secret, got %s", cfg.DBPassword)
	}
	if cfg.DBHost != "db.internal" {
		t.Errorf("expected DBHost db.internal, got %s", cfg.DBHost)
	}
	if cfg.DBPort != 3307 {
		t.Errorf("expected DBPort 3307, got %d", cfg.DBPort)
	}
	if cfg.DBName != "custom_db" {
		t.Errorf("expected DBName custom_db, got %s", cfg.DBName)
	}
	if cfg.RedisAddr != "redis.internal:6380" {
		t.Errorf("expected RedisAddr redis.internal:6380, got %s", cfg.RedisAddr)
	}
}
