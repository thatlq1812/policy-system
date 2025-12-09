package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/thatlq1812/policy-system/user/internal/domain"
	"github.com/thatlq1812/policy-system/user/internal/repository"
)

// UserService defines business operations for users
type UserService interface {
	Register(ctx context.Context, phoneNumber, password, name, platformRole string) (*domain.User, string, error)

	Login(ctx context.Context, phoneNumber, password string) (*domain.User, string, error)

	// RefreshToken generates new access token from valid refresh token
	RefreshToken(ctx context.Context, refreshToken string) (string, int64, error)

	// Logout revokes a refresh token
	Logout(ctx context.Context, refreshToken string) error
}

// userService implements UserService
type userService struct {
	repo             repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository // NEW
	jwtSecret        string
	jwtExpiryHours   int // Deprecated, use constants in token_helpers.go
}

// NewUserService creates a new service instance
func NewUserService(
	repo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository, // NEW
	jwtSecret string,
	jwtExpiryHours int,
) UserService {
	return &userService{
		repo:             repo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        jwtSecret,
		jwtExpiryHours:   jwtExpiryHours,
	}
}

// Register creates a new user with dual token authentication
func (s *userService) Register(ctx context.Context, phoneNumber, password, name, platformRole string) (*domain.User, string, string, int64, int64, error) {
	// 1. Validate input
	if err := s.validateRegisterInput(phoneNumber, password, platformRole); err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Check if user already exists
	existingUser, err := s.repo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, "", "", 0, 0, fmt.Errorf("user with phone number %s already exists", phoneNumber)
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
		return nil, "", "", 0, 0, fmt.Errorf("phone number and password are required")
	}

	// 2. Get user by phone number
	user, err := s.repo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, "", "", 0, 0, fmt.Errorf("invalid credentials")
	}

	// 3. Verify password
	if err := s.verifyPassword(user.PasswordHash, password); err != nil {
		return nil, "", "", 0, 0, fmt.Errorf("invalid credentials")
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

// RefreshToken generates new access token from valid refresh token
func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (string, int64, error) {
	// 1. Hash the provided refresh token
	tokenHash := s.hashToken(refreshToken)

	// 2. Get refresh token from database
	storedToken, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if storedToken == nil {
		return "", 0, fmt.Errorf("invalid refresh token")
	}

	// 3. Verify token is valid (not revoked, not expired)
	if !storedToken.IsValid() {
		if storedToken.IsRevoked() {
			return "", 0, fmt.Errorf("refresh token has been revoked")
		}
		return "", 0, fmt.Errorf("refresh token has expired")
	}

	// 4. Get user to retrieve platform_role for new access token
	user, err := s.repo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", 0, fmt.Errorf("user not found")
	}

	// 5. Generate new access token
	accessToken, accessExpiresAt, err := s.generateAccessToken(user.ID, user.PlatformRole)
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate access token: %w", err)
	}

	// TODO: Optional - Token rotation (generate new refresh token and revoke old one)
	// This improves security but adds DB write on every refresh

	return accessToken, accessExpiresAt, nil
}

// Logout revokes a refresh token
func (s *userService) Logout(ctx context.Context, refreshToken string) error {
	// 1. Hash the refresh token
	tokenHash := s.hashToken(refreshToken)

	// 2. Revoke the token in database
	err := s.refreshTokenRepo.Revoke(ctx, tokenHash, "user_logout")
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// validateRegisterInput validates registration parameters
func (s *userService) validateRegisterInput(phoneNumber, password, platformRole string) error {
	if phoneNumber == "" {
		return fmt.Errorf("phone number is required")
	}

	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	if platformRole != "Client" && platformRole != "Merchant" {
		return fmt.Errorf("platform_role must be either 'Client' or 'Merchant'")
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
