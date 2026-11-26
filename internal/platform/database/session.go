package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/Mousynano/rbac-backend/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionInvalid = errors.New("session is invalid")

// SessionAuthenticator validates session tokens and current account eligibility.
type SessionAuthenticator struct{ Pool *pgxpool.Pool }

// Authenticate returns the user ID only for a live session on an active account.
func (a SessionAuthenticator) Authenticate(ctx context.Context, token string) (string, error) {
	if len(token) < 40 || len(token) > 128 {
		return "", ErrSessionInvalid
	}
	digest := sha256.Sum256([]byte(token))
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var userID string
	err := a.Pool.QueryRow(ctx, `
		UPDATE sessions s SET last_seen_at = GREATEST(s.last_seen_at, now())
		FROM users u
		WHERE s.token_hash = $1 AND s.user_id = u.id
		  AND s.revoked_at IS NULL AND s.expires_at > now()
		  AND s.last_seen_at > now() - interval '30 minutes'
		  AND u.status = 'active'
		  AND NOT EXISTS (
			SELECT 1 FROM email_blocklist b
			WHERE (b.kind = 'email' AND b.value = u.email)
			   OR (b.kind = 'domain' AND split_part(u.email, '@', 2) = b.value)
			   OR (b.kind = 'domain' AND split_part(u.email, '@', 2) LIKE '%.' || b.value)
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM disposable_email_domains d
			WHERE split_part(u.email, '@', 2) = d.domain
			   OR split_part(u.email, '@', 2) LIKE '%.' || d.domain
		  )
		RETURNING u.id::text`, digest[:]).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSessionInvalid
		}
		return "", fmt.Errorf("%w: verify session: %v", auth.ErrUnavailable, err)
	}
	return userID, nil
}
