package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Token expiry constants
const (
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 30 * 24 * time.Hour // 30 days
)

// generateAccessToken creates a short-lived JWT access token
func (s *userService) generateAccessToken(userID, platformRole string) (string, int64, error) {
	expiresAt := time.Now().Add(AccessTokenExpiry)

	claims := jwt.MapClaims{
		"user_id":       userID,
		"platform_role": platformRole,
		"type":          "access",
		"exp":           expiresAt.Unix(),
		"iat":           time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, expiresAt.Unix(), nil
}

// generateRefreshToken creates a random refresh token (UUID format)
func (s *userService) generateRefreshToken() string {
	return uuid.New().String()
}

// hashToken creates a SHA256 hash of a token for database storage
func (s *userService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// verifyRefreshToken checks if a refresh token matches its hash
func (s *userService) verifyRefreshToken(token, tokenHash string) bool {
	return s.hashToken(token) == tokenHash
}

// generateRandomBytes creates cryptographically secure random bytes
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
