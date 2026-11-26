package config

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

// validEnv returns the minimum set of environment variables that produce a
// valid development Config. Each test overrides specific keys and restores
// them via t.Cleanup.
func validEnv(t *testing.T) {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xAB
	}
	env := map[string]string{
		"APP_ENV":        "development",
		"PUBLIC_BASE_URL": "http://localhost:8080",
		"ALLOWED_ORIGIN": "http://localhost:3000",
		"DATABASE_URL":   "postgres://user:pass@localhost:5432/rbac",
		"SMTP_HOST":      "localhost",
		"SMTP_PORT":      "1025",
		"SMTP_FROM":      "noreply@example.com",
		"OTP_HMAC_KEY":   base64.StdEncoding.EncodeToString(key),
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestLoad_ValidDevelopment(t *testing.T) {
	validEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != EnvDevelopment {
		t.Errorf("AppEnv = %q; want %q", cfg.AppEnv, EnvDevelopment)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q; want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure should be false in development by default")
	}
	if cfg.GoogleEnabled {
		t.Error("GoogleEnabled should be false by default")
	}
	if cfg.SMTPTLSMode != TLSModeNone {
		t.Errorf("SMTPTLSMode = %q; want %q", cfg.SMTPTLSMode, TLSModeNone)
	}
	if len(cfg.OTPHMACKey) < 32 {
		t.Errorf("OTPHMACKey length = %d; want >= 32", len(cfg.OTPHMACKey))
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	required := []string{
		"DATABASE_URL",
		"PUBLIC_BASE_URL",
		"ALLOWED_ORIGIN",
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_FROM",
		"OTP_HMAC_KEY",
	}
	for _, key := range required {
		t.Run("missing_"+key, func(t *testing.T) {
			validEnv(t)
			t.Setenv(key, "")
			_, err := Load()
			if err == nil {
				t.Fatalf("expected error when %s is empty", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("error should mention %q; got: %v", key, err)
			}
		})
	}
}

func TestLoad_InvalidAppEnv(t *testing.T) {
	validEnv(t)
	t.Setenv("APP_ENV", "staging")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid APP_ENV")
	}
}

func TestLoad_OTPKeyTooShort(t *testing.T) {
	validEnv(t)
	short := make([]byte, 16)
	t.Setenv("OTP_HMAC_KEY", base64.StdEncoding.EncodeToString(short))
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for OTP_HMAC_KEY < 32 bytes")
	}
	if !strings.Contains(err.Error(), "OTP_HMAC_KEY") {
		t.Errorf("error should mention OTP_HMAC_KEY; got: %v", err)
	}
}

func TestLoad_OTPKeyURLSafeBase64(t *testing.T) {
	validEnv(t)
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xDE
	}
	t.Setenv("OTP_HMAC_KEY", base64.URLEncoding.EncodeToString(key))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error with URL-safe base64: %v", err)
	}
	if len(cfg.OTPHMACKey) != 32 {
		t.Errorf("OTPHMACKey length = %d; want 32", len(cfg.OTPHMACKey))
	}
}

func TestLoad_InvalidSMTPPort(t *testing.T) {
	validEnv(t)
	t.Setenv("SMTP_PORT", "99999")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for SMTP_PORT out of range")
	}
}

func TestLoad_InvalidSMTPTLSMode(t *testing.T) {
	validEnv(t)
	t.Setenv("SMTP_TLS_MODE", "tls13")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unknown SMTP_TLS_MODE")
	}
}

func TestLoad_ProductionRequiresCookieSecure(t *testing.T) {
	validEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("SMTP_TLS_MODE", "starttls")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for COOKIE_SECURE=false in production")
	}
}

func TestLoad_ProductionForbidsNoTLS(t *testing.T) {
	validEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "true")
	// SMTP_TLS_MODE defaults to "none" — should fail in production
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for SMTP_TLS_MODE=none in production")
	}
}

func TestLoad_GoogleEnabled_MissingCredentials(t *testing.T) {
	validEnv(t)
	t.Setenv("GOOGLE_ENABLED", "true")
	// GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL not set
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when Google enabled but credentials missing")
	}
	if !strings.Contains(err.Error(), "GOOGLE_CLIENT_ID") {
		t.Errorf("error should mention GOOGLE_CLIENT_ID; got: %v", err)
	}
}

func TestLoad_CustomHTTPAddr(t *testing.T) {
	validEnv(t)
	t.Setenv("HTTP_ADDR", ":9090")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q; want %q", cfg.HTTPAddr, ":9090")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	validEnv(t)
	t.Setenv("LOG_LEVEL", "verbose")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestMain(m *testing.M) {
	// Ensure we start with a clean environment for each test run.
	// Individual tests use t.Setenv which is automatically restored.
	os.Exit(m.Run())
}
