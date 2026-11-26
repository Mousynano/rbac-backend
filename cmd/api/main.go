// Command api is the composition root for the RBAC Blueprint Designer API.
// It loads configuration, connects to PostgreSQL, registers routes, and runs
// the HTTP server with graceful shutdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Mousynano/rbac-backend/internal/auth"
	"github.com/Mousynano/rbac-backend/internal/auth/httpapi"
	"github.com/Mousynano/rbac-backend/internal/config"
	"github.com/Mousynano/rbac-backend/internal/health"
	"github.com/Mousynano/rbac-backend/internal/platform/database"
	"github.com/Mousynano/rbac-backend/internal/platform/httpserver"
	"github.com/Mousynano/rbac-backend/internal/platform/mail"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Structured logger.
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Database pool.
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	slog.Info("database pool ready")

	// Routes.
	mux := http.NewServeMux()
	httpserver.HealthRoutes(mux, health.New(database.Readiness{Pool: pool}))
	mailer := mail.SMTP{Config: mail.Config{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, TLSMode: string(cfg.SMTPTLSMode)}}
	authService := auth.Service{DB: pool, Mailer: mailer, HMACKey: cfg.OTPHMACKey}
	httpapi.Routes(mux, authService, cfg.CookieSecure)
	sessionAuth := database.SessionAuthenticator{Pool: pool}
	httpserver.CurrentUserRoute(mux, sessionAuth, sessionAuth)

	// HTTP server.
	srv := httpserver.New(cfg.HTTPAddr, httpserver.Middleware(mux, cfg.AllowedOrigin))

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	if err := httpserver.GracefulShutdown(srv); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	slog.Info("shutdown complete")
	return nil
}
