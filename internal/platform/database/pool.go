// Package database provides the PostgreSQL connection pool and readiness check.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxConns        = 10
	minConns        = 0
	maxConnLifetime = 30 * time.Minute
	maxConnIdleTime = 5 * time.Minute
	connectTimeout  = 10 * time.Second
	healthTimeout   = 3 * time.Second
)

// Open creates and validates a pgxpool using the given connection URL.
// It pings the database before returning to confirm the pool is usable.
func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.MaxConnLifetime = maxConnLifetime
	cfg.MaxConnIdleTime = maxConnIdleTime
	cfg.ConnConfig.ConnectTimeout = connectTimeout

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := Ping(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("initial ping: %w", err)
	}

	return pool, nil
}

// Ping verifies that the pool can reach the database within healthTimeout.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	pingCtx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()
	return pool.Ping(pingCtx)
}
