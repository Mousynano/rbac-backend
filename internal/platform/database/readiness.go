package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RequiredSchemaVersion is the latest migration required by this API build.
const RequiredSchemaVersion int64 = 2

// Readiness checks connectivity and the schema version installed by goose.
type Readiness struct{ Pool *pgxpool.Pool }

// Check fails closed when PostgreSQL or its expected schema is unavailable.
func (r Readiness) Check(ctx context.Context) error {
	if err := Ping(ctx, r.Pool); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	var version int64
	err := r.Pool.QueryRow(ctx, `SELECT version_id FROM goose_db_version WHERE is_applied = true ORDER BY id DESC LIMIT 1`).Scan(&version)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if version != RequiredSchemaVersion {
		return fmt.Errorf("schema version %d, want %d", version, RequiredSchemaVersion)
	}
	return nil
}
