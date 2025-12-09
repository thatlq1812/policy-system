package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thatlq1812/policy-system/user/internal/domain"
)

// RefreshTokenRepository defines operations for refresh tokens
type RefreshTokenRepository interface {
	// Create saves a new refresh token to database
	Create(ctx context.Context, token domain.CreateRefreshTokenParams) (*domain.RefreshToken, error)

	// GetByTokenHash retrieves a refresh token by its hash
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)

	// Revoke marks a refresh token as revoked
	Revoke(ctx context.Context, tokenHash, reason string) error

	// RevokeAllUserTokens revokes all refresh tokens for a user (logout all devices)
	RevokeAllUserTokens(ctx context.Context, userID, reason string) (int64, error)

	// DeleteExpiredTokens removes tokens that have expired from the database (cleanup)
	DeleteExpired(ctx context.Context) (int64, error)

	// CountActiveTokens return number of active tokens for a user
	CountActivateTokens(ctx context.Context, userID string) (int, error)
}

// postgresRefreshTokenRepository implements RefreshTokenRepository
type postgresRefreshTokenRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRefreshTokenRepository creates a new refresh token repository
func NewPostgresRefreshTokenRepository(db *pgxpool.Pool) RefreshTokenRepository {
	return &postgresRefreshTokenRepository{db: db}
}

// Create saves a new refresh token
func (r *postgresRefreshTokenRepository) Create(ctx context.Context, params domain.CreateRefreshTokenParams) (*domain.RefreshToken, error) {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, device_info, ip_address)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token_hash, expires_at, created_at, revoked_at, revoked_reason, device_info, ip_address
	`
	var token domain.RefreshToken
	err := r.db.QueryRow(ctx, query,
		params.UserID,
		params.TokenHash,
		params.ExpiresAt,
		params.DeviceInfo,
		params.IPAddress,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
		&token.RevokedReason,
		&token.DeviceInfo,
		&token.IPAddress,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}
	return &token, nil
}

// GetByTokenHash retrieves a refresh token by its hash
func (r *postgresRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, revoked_reason, device_info, ip_address
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token domain.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
		&token.RevokedReason,
		&token.DeviceInfo,
		&token.IPAddress,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &token, nil
}

// Revoke marks a refresh token as revoked
func (r *postgresRefreshTokenRepository) Revoke(ctx context.Context, tokenHash, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP, revoked_reason = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	result, err := r.db.Exec(ctx, query, tokenHash, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found or already revoked")
	}
	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *postgresRefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID, reason string) (int64, error) {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP, revoked_reason = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	result, err := r.db.Exec(ctx, query, userID, reason)
	if err != nil {
		return 0, fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return result.RowsAffected(), nil
}

// DeleteExpired removes tokens from database
func (r *postgresRefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < CURRENT_TIMESTAMP
	`
	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	return result.RowsAffected(), nil
}

// CountActivateTokens returns number of active tokens for a user
func (r *postgresRefreshTokenRepository) CountActivateTokens(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM refresh_tokens
		WHERE user_id = $1
			AND revoked_at IS NULL
			AND expires_at > CURRENT_TIMESTAMP
	`

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active tokens: %w", err)
	}

	return count, nil
}
