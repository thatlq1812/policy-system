package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TokenBlacklistRepository defines operations for token blacklist
type TokenBlacklistRepository interface {
	// Add adds a token to the blacklist
	Add(ctx context.Context, jti, userID string, expiresAt time.Time, reason string) error

	// IsBlacklisted checks if a token is blacklisted
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	// CleanupExpired removes expired tokens from blacklist
	CleanupExpired(ctx context.Context) (int64, error)

	// RevokeAllUserTokens adds all active tokens for a user to blacklist
	// This is used when user changes password or requests security logout
	RevokeAllUserTokens(ctx context.Context, userID string, reason string) (int64, error)
}

// postgresTokenBlacklistRepository implements TokenBlacklistRepository
type postgresTokenBlacklistRepository struct {
	db *pgxpool.Pool
}

// NewPostgresTokenBlacklistRepository creates a new token blacklist repository
func NewPostgresTokenBlacklistRepository(db *pgxpool.Pool) TokenBlacklistRepository {
	return &postgresTokenBlacklistRepository{db: db}
}

// Add adds a token to the blacklist
func (r *postgresTokenBlacklistRepository) Add(ctx context.Context, jti, userID string, expiresAt time.Time, reason string) error {
	query := `
		INSERT INTO token_blacklist (jti, user_id, expires_at, reason)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (jti) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, jti, userID, expiresAt, reason)
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	return nil
}

// IsBlacklisted checks if a token is blacklisted
func (r *postgresTokenBlacklistRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM token_blacklist 
			WHERE jti = $1 AND expires_at > CURRENT_TIMESTAMP
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, jti).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	return exists, nil
}

// CleanupExpired removes expired tokens from blacklist
func (r *postgresTokenBlacklistRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM token_blacklist 
		WHERE expires_at <= CURRENT_TIMESTAMP
	`
	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return result.RowsAffected(), nil
}

// RevokeAllUserTokens is a placeholder for future implementation
// This would require tracking active JTIs per user, which is complex
// For now, we rely on individual token revocation on logout
func (r *postgresTokenBlacklistRepository) RevokeAllUserTokens(ctx context.Context, userID string, reason string) (int64, error) {
	// NOTE: This is not fully implemented because access tokens are stateless
	// We don't track which access tokens are currently active for a user
	//
	// Options:
	// 1. Track all issued access tokens (adds DB write on every login/refresh)
	// 2. When password changes, blacklist a "user revoke timestamp" and check in middleware
	// 3. Accept that password change only revokes refresh tokens (current approach)

	// For now, this is a no-op
	// In practice, password change revokes all refresh tokens,
	// preventing generation of new access tokens
	return 0, nil
}
