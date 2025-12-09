package domain

import "time"

// RefreshToken represents a refresh token entity

type RefreshToken struct {
	ID            string     `db:"id"`
	UserID        string     `db:"user_id"`
	TokenHash     string     `db:"token_hash"`
	ExpiresAt     time.Time  `db:"expires_at"`
	CreatedAt     time.Time  `db:"created_at"`
	RevokedAt     *time.Time `db:"revoked_at"`
	RevokedReason *string    `db:"revoked_reason"`
	DeviceInfo    *string    `db:"device_info"`
	IPAddress     *string    `db:"ip_address"`
}

// CreateRefreshTokenParams contains parameters for creating a refresh token
type CreateRefreshTokenParams struct {
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	DeviceInfo string
	IPAddress  string
}

// IsRevoked checks if the refresh token has been revoked
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

// IsExpired checks if the refresh token has expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if token is valid (not revoked and not expired)
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsRevoked() && !rt.IsExpired()
}
