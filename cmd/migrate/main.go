// Command migrate applies versioned SQL migrations using goose.
// It reads DATABASE_URL from the environment and exits non-zero on failure.
// Usage: migrate [up|down|status|version]  (default: up)
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx database/sql driver
	"github.com/pressly/goose/v3"
)

const migrationsDir = "migrations"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "up", "status", "version":
	default:
		return fmt.Errorf("unsupported command %q (use up, status, or version)", command)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// Migration operations are bounded and never run implicitly at API startup.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	goose.SetBaseFS(nil) // use OS filesystem
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.RunContext(ctx, command, db, migrationsDir); err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}
	return nil
}
