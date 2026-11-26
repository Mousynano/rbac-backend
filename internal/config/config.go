// Package config loads and validates application configuration from environment
// variables. It fails fast at startup if any required value is missing or
// invalid; it never supplies secret defaults.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Env identifies the deployment environment.
type Env string

const (
	EnvDevelopment Env = "development"
	EnvProduction  Env = "production"
)

// SMTPTLSMode controls how the SMTP client establishes TLS.
type SMTPTLSMode string

const (
	TLSModeNone     SMTPTLSMode = "none"      // local/loopback only, no TLS
	TLSModeStartTLS SMTPTLSMode = "starttls"  // upgrade via STARTTLS
	TLSModeImplicit SMTPTLSMode = "implicit"  // TLS from the first byte (port 465)
)

// Config holds all runtime configuration. Fields are populated by Load; no
// field carries a secret default value. Callers must not modify Config after
// Load returns.
type Config struct {
	AppEnv         Env
	HTTPAddr       string
	PublicBaseURL  string
	AllowedOrigin  string
	DatabaseURL    string
	CookieSecure   bool

	// Google OIDC — optional; routes return 503 when disabled.
	GoogleEnabled      bool
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// SMTP
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPTLSMode  SMTPTLSMode

	// OTPHMACKey is the decoded key (>=32 bytes) used to sign OTP digests.
	OTPHMACKey []byte

	// LogLevel controls structured log verbosity: debug, info, warn, error.
	LogLevel string
}

// Load reads every required variable from the environment and returns a
// validated Config. Any missing required field or invalid value causes an
// error that names all failing fields so the operator can fix them in one
// restart.
func Load() (*Config, error) {
	var errs []string

	require := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			errs = append(errs, fmt.Sprintf("%s is required", key))
		}
		return v
	}

	optional := func(key, def string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return def
		}
		return v
	}

	cfg := &Config{}

	// Runtime environment
	rawEnv := optional("APP_ENV", string(EnvDevelopment))
	switch Env(rawEnv) {
	case EnvDevelopment, EnvProduction:
		cfg.AppEnv = Env(rawEnv)
	default:
		errs = append(errs, fmt.Sprintf("APP_ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, rawEnv))
	}

	cfg.HTTPAddr = optional("HTTP_ADDR", ":8080")
	cfg.PublicBaseURL = require("PUBLIC_BASE_URL")
	cfg.AllowedOrigin = require("ALLOWED_ORIGIN")
	cfg.DatabaseURL = require("DATABASE_URL")

	// COOKIE_SECURE defaults to true in production; must be explicitly false in development.
	rawSecure := optional("COOKIE_SECURE", "")
	if rawSecure == "" {
		cfg.CookieSecure = cfg.AppEnv == EnvProduction
	} else {
		var err error
		cfg.CookieSecure, err = strconv.ParseBool(rawSecure)
		if err != nil {
			errs = append(errs, "COOKIE_SECURE must be true or false")
		}
	}
	if cfg.AppEnv == EnvProduction && !cfg.CookieSecure {
		errs = append(errs, "COOKIE_SECURE must be true in production")
	}

	// Google OIDC
	rawGoogleEnabled := optional("GOOGLE_ENABLED", "false")
	var googleEnabledErr error
	cfg.GoogleEnabled, googleEnabledErr = strconv.ParseBool(rawGoogleEnabled)
	if googleEnabledErr != nil {
		errs = append(errs, "GOOGLE_ENABLED must be true or false")
	}
	if cfg.GoogleEnabled {
		cfg.GoogleClientID = require("GOOGLE_CLIENT_ID")
		cfg.GoogleClientSecret = require("GOOGLE_CLIENT_SECRET")
		cfg.GoogleRedirectURL = require("GOOGLE_REDIRECT_URL")
	}

	// SMTP
	cfg.SMTPHost = require("SMTP_HOST")
	rawSMTPPort := require("SMTP_PORT")
	if rawSMTPPort != "" {
		p, err := strconv.Atoi(rawSMTPPort)
		if err != nil || p < 1 || p > 65535 {
			errs = append(errs, "SMTP_PORT must be a valid TCP port number (1–65535)")
		} else {
			cfg.SMTPPort = p
		}
	}
	cfg.SMTPFrom = require("SMTP_FROM")

	// SMTP credentials are optional (Mailpit has none locally).
	cfg.SMTPUsername = optional("SMTP_USERNAME", "")
	cfg.SMTPPassword = optional("SMTP_PASSWORD", "")

	rawTLS := optional("SMTP_TLS_MODE", string(TLSModeNone))
	switch SMTPTLSMode(rawTLS) {
	case TLSModeNone, TLSModeStartTLS, TLSModeImplicit:
		cfg.SMTPTLSMode = SMTPTLSMode(rawTLS)
	default:
		errs = append(errs, fmt.Sprintf("SMTP_TLS_MODE must be none, starttls, or implicit; got %q", rawTLS))
	}
	if cfg.AppEnv == EnvProduction && cfg.SMTPTLSMode == TLSModeNone {
		errs = append(errs, "SMTP_TLS_MODE=none is not allowed in production")
	}

	// OTP HMAC key — base64-encoded, minimum 32 bytes decoded.
	rawOTPKey := require("OTP_HMAC_KEY")
	if rawOTPKey != "" {
		decoded, err := base64.StdEncoding.DecodeString(rawOTPKey)
		if err != nil {
			// Try URL-safe variant.
			decoded, err = base64.URLEncoding.DecodeString(rawOTPKey)
		}
		if err != nil {
			errs = append(errs, "OTP_HMAC_KEY must be a valid base64-encoded value")
		} else if len(decoded) < 32 {
			errs = append(errs, fmt.Sprintf("OTP_HMAC_KEY must decode to at least 32 bytes; got %d", len(decoded)))
		} else {
			cfg.OTPHMACKey = decoded
		}
	}

	// Log level
	rawLevel := strings.ToLower(optional("LOG_LEVEL", "info"))
	switch rawLevel {
	case "debug", "info", "warn", "error":
		cfg.LogLevel = rawLevel
	default:
		errs = append(errs, fmt.Sprintf("LOG_LEVEL must be debug, info, warn, or error; got %q", rawLevel))
	}

	if len(errs) > 0 {
		return nil, errors.New("configuration errors:\n  - " + strings.Join(errs, "\n  - "))
	}
	return cfg, nil
}
