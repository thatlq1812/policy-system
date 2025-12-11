package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/thatlq1812/policy-system/shared/pkg/validator"
	"github.com/thatlq1812/policy-system/user/internal/domain"
	"github.com/thatlq1812/policy-system/user/internal/repository"
)

// UserService defines business operations for users
type UserService interface {
	Register(ctx context.Context, phoneNumber, password, name, platformRole string) (*domain.User, string, string, int64, int64, error)

	Login(ctx context.Context, phoneNumber, password string) (*domain.User, string, string, int64, int64, error)

	// RefreshToken generates new access token and rotates refresh token
	RefreshToken(ctx context.Context, refreshToken string) (accessToken string, newRefreshToken string, accessExpiresAt int64, refreshExpiresAt int64, err error)

	// Logout revokes a refresh token and optionally blacklists access token
	Logout(ctx context.Context, refreshToken string, accessToken string) error

	// GetUserProfile retrieves user profile by ID
	GetUserProfile(ctx context.Context, userID string) (*domain.User, error)

	// UpdateUserProfile updates user profile information
	UpdateUserProfile(ctx context.Context, userID string, name, phoneNumber *string) (*domain.User, error)

	// ChangePassword changes user's password
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error

	// Admin operations

	// ListUsers lists users with pagination and filtering
	ListUsers(ctx context.Context, page, pageSize int, platformRole string, includeDeleted bool) ([]*domain.User, int, int, error)

	// SearchUsers searches users by name or phone number with pagination
	SearchUsers(ctx context.Context, query string, limit int) ([]*domain.User, error)

	//DeleteUser deletes a user by ID (soft delete)
	DeleteUser(ctx context.Context, userId, reason string) error

	// HardDeleteUser permanently deletes a user (for rollback scenarios only)
	HardDeleteUser(ctx context.Context, userID string) error

	//UpdateUserRole updates a user's platform role
	UpdateUserRole(ctx context.Context, userID, newPlatformRole string) (*domain.User, error)

	// GetActiveSessions retrieves all active sessions (refresh tokens) for a user
	GetActiveSessions(ctx context.Context, userID string) ([]*domain.RefreshToken, int, error)

	// LogoutAllDevices revokes all refresh tokens for a user
	LogoutAllDevices(ctx context.Context, userID string) (int64, error)

	// RevokeSession revokes a specific refresh token by its ID
	RevokeSession(ctx context.Context, userID, tokenID string) error

	// GetUserStats retrieves user statistics
	GetUserStats(ctx context.Context) (map[string]int, error)

	// IsTokenBlacklisted checks if a token JTI is blacklisted
	IsTokenBlacklisted(ctx context.Context, jti string) (bool, error)
}

// userService implements UserService
type userService struct {
	repo             repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	blacklistRepo    repository.TokenBlacklistRepository // NEW: For access token revocation
	jwtSecret        string
	jwtExpiryHours   int // Deprecated, use constants in token_helper.go
}

// NewUserService creates a new service instance
func NewUserService(
	repo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	blacklistRepo repository.TokenBlacklistRepository, // NEW
	jwtSecret string,
	jwtExpiryHours int,
) UserService {
	return &userService{
		repo:             repo,
		refreshTokenRepo: refreshTokenRepo,
		blacklistRepo:    blacklistRepo,
		jwtSecret:        jwtSecret,
		jwtExpiryHours:   jwtExpiryHours,
	}
}

// Register creates a new user with dual token authentication
func (s *userService) Register(ctx context.Context, phoneNumber, password, name, platformRole string) (*domain.User, string, string, int64, int64, error) {
	// 1. Validate input
	if err := s.validateRegisterInput(phoneNumber, password, platformRole); err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}

	// 2. Check if user already exists
	existingUser, err := s.repo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, "", "", 0, 0, fmt.Errorf("%w: user with phone number %s", domain.ErrAlreadyExists, phoneNumber)
	}

	// 3. Hash password
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Create user in database
	user, err := s.repo.Create(ctx, domain.CreateUserParams{
		PhoneNumber:  phoneNumber,
		PasswordHash: passwordHash,
		Name:         name,
		PlatformRole: platformRole,
	})
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to create user: %w", err)
	}

	// 5. Generate access token
	accessToken, accessExpiresAt, err := s.generateAccessToken(user.ID, user.PlatformRole)
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 6. Generate and store refresh token
	refreshToken := s.generateRefreshToken()
	refreshTokenHash := s.hashToken(refreshToken)
	refreshExpiresAt := time.Now().Add(RefreshTokenExpiry)

	_, err = s.refreshTokenRepo.Create(ctx, domain.CreateRefreshTokenParams{
		UserID:     user.ID,
		TokenHash:  refreshTokenHash,
		ExpiresAt:  refreshExpiresAt,
		DeviceInfo: "registration", // TODO: Extract from context/metadata
		IPAddress:  "",             // TODO: Extract from context/metadata
	})
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return user, accessToken, refreshToken, accessExpiresAt, refreshExpiresAt.Unix(), nil
}

// Login authenticates user and returns dual tokens
func (s *userService) Login(ctx context.Context, phoneNumber, password string) (*domain.User, string, string, int64, int64, error) {
	// 1. Validate input
	if phoneNumber == "" || password == "" {
		return nil, "", "", 0, 0, fmt.Errorf("%w: phone number and password are required", domain.ErrInvalidInput)
	}

	// 2. Get user by phone number
	user, err := s.repo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, "", "", 0, 0, domain.ErrInvalidCredentials
	}

	// 3. Verify password
	if err := s.verifyPassword(user.PasswordHash, password); err != nil {
		return nil, "", "", 0, 0, domain.ErrInvalidCredentials
	}

	// 4. Generate access token
	accessToken, accessExpiresAt, err := s.generateAccessToken(user.ID, user.PlatformRole)
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 5. Generate and store refresh token
	refreshToken := s.generateRefreshToken()
	refreshTokenHash := s.hashToken(refreshToken)
	refreshExpiresAt := time.Now().Add(RefreshTokenExpiry)

	_, err = s.refreshTokenRepo.Create(ctx, domain.CreateRefreshTokenParams{
		UserID:     user.ID,
		TokenHash:  refreshTokenHash,
		ExpiresAt:  refreshExpiresAt,
		DeviceInfo: "login", // TODO: Extract from context/metadata
		IPAddress:  "",      // TODO: Extract from context/metadata
	})
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return user, accessToken, refreshToken, accessExpiresAt, refreshExpiresAt.Unix(), nil
}

// RefreshToken generates new access token and rotates refresh token for security
func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (string, string, int64, int64, error) {
	// 1. Hash the provided refresh token
	tokenHash := s.hashToken(refreshToken)

	// 2. Get refresh token from database
	storedToken, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if storedToken == nil {
		return "", "", 0, 0, fmt.Errorf("invalid refresh token")
	}

	// 3. Verify token is valid (not revoked, not expired)
	if !storedToken.IsValid() {
		if storedToken.IsRevoked() {
			return "", "", 0, 0, fmt.Errorf("refresh token has been revoked")
		}
		return "", "", 0, 0, fmt.Errorf("refresh token has expired")
	}

	// 4. Get user to retrieve platform_role for new access token
	user, err := s.repo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", "", 0, 0, fmt.Errorf("user not found")
	}

	// 5. Generate new access token
	accessToken, accessExpiresAt, err := s.generateAccessToken(user.ID, user.PlatformRole)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 6. TOKEN ROTATION: Generate new refresh token and revoke old one
	// This improves security - if old token is stolen, it becomes useless after one use
	newRefreshToken := s.generateRefreshToken()
	newRefreshTokenHash := s.hashToken(newRefreshToken)
	newRefreshExpiresAt := time.Now().Add(RefreshTokenExpiry)

	// Preserve device info and IP if available
	deviceInfo := "token_refresh"
	ipAddress := ""
	if storedToken.DeviceInfo != nil {
		deviceInfo = *storedToken.DeviceInfo
	}
	if storedToken.IPAddress != nil {
		ipAddress = *storedToken.IPAddress
	}

	// Store new refresh token
	_, err = s.refreshTokenRepo.Create(ctx, domain.CreateRefreshTokenParams{
		UserID:     user.ID,
		TokenHash:  newRefreshTokenHash,
		ExpiresAt:  newRefreshExpiresAt,
		DeviceInfo: deviceInfo,
		IPAddress:  ipAddress,
	})
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	// Revoke old refresh token
	err = s.refreshTokenRepo.Revoke(ctx, tokenHash, "token_rotation")
	if err != nil {
		// Log error but don't fail - new token is already issued
		log.Printf("WARNING: Failed to revoke old refresh token during rotation: %v", err)
	}

	log.Printf("INFO: Token rotated for user %s (old token revoked)", user.ID)
	return accessToken, newRefreshToken, accessExpiresAt, newRefreshExpiresAt.Unix(), nil
}

// Logout revokes a refresh token and blacklists access token
func (s *userService) Logout(ctx context.Context, refreshToken string, accessToken string) error {
	// 1. Hash the refresh token
	tokenHash := s.hashToken(refreshToken)

	// 2. Revoke the refresh token in database
	err := s.refreshTokenRepo.Revoke(ctx, tokenHash, "user_logout")
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	// 3. Blacklist the access token if provided
	if accessToken != "" && s.blacklistRepo != nil {
		// Parse access token to extract JTI and expiration
		token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSecret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				jti, jtiOk := claims["jti"].(string)
				userID, userIDOk := claims["user_id"].(string)
				exp, expOk := claims["exp"].(float64)

				if jtiOk && userIDOk && expOk {
					expiresAt := time.Unix(int64(exp), 0)

					// Add to blacklist
					err := s.blacklistRepo.Add(ctx, jti, userID, expiresAt, "user_logout")
					if err != nil {
						// Log but don't fail logout - refresh token already revoked
						log.Printf("WARNING: Failed to blacklist access token: %v", err)
					}
				}
			}
		}
		// If token parsing fails, continue - refresh token is already revoked
	}

	return nil
}

// GetUserProfile retrieves user profile by ID
func (s *userService) GetUserProfile(ctx context.Context, userID string) (*domain.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user ID is required", domain.ErrInvalidInput)
	}

	user, err := s.repo.GetByID(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if user == nil {
		return nil, domain.ErrNotFound
	}

	return user, nil
}

// UpdateUserProfile updates user profile information
func (s *userService) UpdateUserProfile(ctx context.Context, userID string, name, phoneNumber *string) (*domain.User, error) {
	// Validate
	if userID == "" {
		return nil, fmt.Errorf("%w: user ID is required", domain.ErrInvalidInput)
	}

	if name == nil && phoneNumber == nil {
		return nil, fmt.Errorf("%w: at least one field to update must be provided", domain.ErrInvalidInput)
	}

	// Validate phone number if provided
	if phoneNumber != nil {
		if !isValidPhoneNumber(*phoneNumber) {
			return nil, fmt.Errorf("%w: invalid phone number format", domain.ErrInvalidInput)
		}

		// Check for uniqueness
		existing, err := s.repo.GetByPhoneNumber(ctx, *phoneNumber)
		if err == nil && existing.ID != userID {
			return nil, fmt.Errorf("%w: phone number already in use", domain.ErrAlreadyExists)
		}
	}

	// Update in repository
	updatedUser, err := s.repo.Update(ctx, domain.UpdateUserParams{
		ID:          userID,
		Name:        name,
		PhoneNumber: phoneNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	return updatedUser, nil
}

// ChangePassword changes user's password
func (s *userService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Validate
	if userID == "" || oldPassword == "" || newPassword == "" {
		return fmt.Errorf("%w: user ID, old password, and new password are required", domain.ErrInvalidInput)
	}

	if err := validatePasswordStrength(newPassword); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}

	// Get user and verify old password
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword))
	if err != nil {
		return domain.ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in repository
	err = s.repo.UpdatePassword(ctx, userID, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to update password : %w", err)
	}

	// Revoke all refresh tokens (force re-login)
	count, err := s.refreshTokenRepo.RevokeAllUserTokens(ctx, userID, "password_changed")
	if err != nil {
		// Log but don't fail
		fmt.Printf("WARNING: Failed to revoke refresh tokens after password change for user %s: %v\n", userID, err)
	} else {
		fmt.Printf("INFO: Revoked %d refresh tokens for user %s after password change\n", count, userID)
	}
	return nil
}

// ListUsers returns paginated user list
func (s *userService) ListUsers(ctx context.Context, page, pageSize int, platformRole string, includeDeleted bool) ([]*domain.User, int, int, error) {
	// Validation pagination parameters
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 10
	}

	if pageSize > 100 {
		pageSize = 100
	}
	// Get user and total count
	users, totalCount, err := s.repo.ListUsers(ctx, domain.ListUsersParams{
		Page:           page,
		PageSize:       pageSize,
		PlatformRole:   platformRole,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Calculate total pages
	totalPage := (totalCount + pageSize - 1) / pageSize
	return users, totalCount, totalPage, nil
}

// SearchUsers searches users by name or phone number
func (s *userService) SearchUsers(ctx context.Context, query string, limit int) ([]*domain.User, error) {
	if query == "" {
		return nil, fmt.Errorf("%w: query is required", domain.ErrInvalidInput)
	}

	if limit < 1 {
		limit = 10
	}

	users, err := s.repo.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}

// DeleteUser performs a soft delete of a user and revokes all tokens of the user
func (s *userService) DeleteUser(ctx context.Context, userID, reason string) error {
	if userID == "" {
		return fmt.Errorf("%w: user ID is required", domain.ErrInvalidInput)
	}

	// Soft delete user
	err := s.repo.SoftDelete(ctx, userID, reason)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Revoke all refresh tokens
	count, err := s.refreshTokenRepo.RevokeAllUserTokens(ctx, userID, "user_deleted")
	if err != nil {
		log.Printf("WARNING: Failed to revoke refresh tokens after user deletion for user %s: %v\n", userID, err)
	} else {
		log.Printf("INFO: Revoked %d refresh tokens for user %s after deletion\n", count, userID)
	}
	return nil
}

// HardDeleteUser permanently removes a user from database
// WARNING: This should ONLY be used for transaction rollback scenarios
// Regular deletions must use DeleteUser (soft delete) to preserve audit trail
func (s *userService) HardDeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("%w: user ID is required", domain.ErrInvalidInput)
	}

	// Hard delete user (no audit trail preserved)
	err := s.repo.HardDelete(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to hard delete user: %w", err)
	}

	log.Printf("WARNING: User %s permanently deleted (hard delete)", userID)
	return nil
}

// UpdateUserRole updates user's platform role
func (s *userService) UpdateUserRole(ctx context.Context, userID, newPlatformRole string) (*domain.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user ID is required", domain.ErrInvalidInput)
	}

	if !isValidPlatformRole(newPlatformRole) {
		return nil, fmt.Errorf("%w: invalid platform role (must be 'Client' or 'Merchant' or 'Admin')", domain.ErrInvalidInput)
	}

	// Update user role in repository
	user, err := s.repo.UpdateRole(ctx, userID, newPlatformRole)
	if err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	return user, nil
}

// GetActiveSessions retrieves all active sessions (refresh tokens) for a user
func (s *userService) GetActiveSessions(ctx context.Context, userID string) ([]*domain.RefreshToken, int, error) {
	// Validate user ID
	if userID == "" {
		return nil, 0, fmt.Errorf("user_id is required")
	}

	// Check if user exists
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, 0, fmt.Errorf("user not found")
	}

	// Get active tokens from repository
	tokens, err := s.refreshTokenRepo.GetActiveTokensByUserID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get active tokens: %w", err)
	}

	// Count active tokens (for metadata)
	count, err := s.refreshTokenRepo.CountActivateTokens(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activate token: %w", err)
	}

	return tokens, count, nil
}

// LogoutAllDevices revokes all refresh tokens for a user
func (s *userService) LogoutAllDevices(ctx context.Context, userID string) (int64, error) {
	// Validate user ID
	if userID == "" {
		return 0, fmt.Errorf("user_id is required")
	}

	// Check if user exists
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("user not found")
	}

	// Revoke all tokens
	revokedCount, err := s.refreshTokenRepo.RevokeAllUserTokens(ctx, userID, "logout_all_devices")
	if err != nil {
		return 0, fmt.Errorf("failed to revoke all tokens: %w", err)
	}

	return revokedCount, nil
}

// RevokeSession revokes a specific refresh token by its ID
func (s *userService) RevokeSession(ctx context.Context, userID, tokenID string) error {
	// Validate inputs
	if userID == "" || tokenID == "" {
		return fmt.Errorf("user_id and token_id are required")
	}

	// Revoke token by ID
	err := s.refreshTokenRepo.RevokeByID(ctx, tokenID, "manual_revoke")
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// Helpers
// isValidPhoneNumber validates phone number using shared validator
func isValidPhoneNumber(phone string) bool {
	return validator.ValidatePhoneNumber(phone) == nil
}

// GetUserStats returns statistics about user accounts
func (s *userService) GetUserStats(ctx context.Context) (map[string]int, error) {
	// 1. Get user stats from user repository
	userStats, err := s.repo.GetUserStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// 2. Get active sessions count
	activeSessions, err := s.refreshTokenRepo.CountAllActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active sessions: %w", err)
	}
	userStats["total_active_sessions"] = activeSessions

	return userStats, nil
}

func validatePasswordStrength(password string) error {
	// Use basic strength for backward compatibility
	// TODO: Upgrade to DefaultPasswordStrength() for better security
	return validator.ValidatePassword(password, validator.BasicPasswordStrength())
}

// IsTokenBlacklisted checks if a token JTI is in the blacklist
func (s *userService) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	if s.blacklistRepo == nil {
		// If blacklist is not configured, tokens are never blacklisted
		return false, nil
	}

	isBlacklisted, err := s.blacklistRepo.IsBlacklisted(ctx, jti)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return isBlacklisted, nil
}

// validateRegisterInput validates registration parameters using shared validator
func (s *userService) validateRegisterInput(phoneNumber, password, platformRole string) error {
	if err := validator.ValidatePhoneNumber(phoneNumber); err != nil {
		return err
	}

	if err := validatePasswordStrength(password); err != nil {
		return err
	}

	if err := validator.ValidatePlatformRole(platformRole); err != nil {
		return err
	}

	return nil
}

// hashPassword hashes a plain text password using bcrypt
func (s *userService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifyPassword compares hashed password with plain text password
func (s *userService) verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// generateJWT generates a JWT token for authenticated user
func (s *userService) generateJWT(userID, platformRole string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":       userID,
		"platform_role": platformRole,
		"exp":           time.Now().Add(time.Hour * time.Duration(s.jwtExpiryHours)).Unix(),
		"iat":           time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func isValidPlatformRole(role string) bool {
	return validator.ValidatePlatformRole(role) == nil
}
