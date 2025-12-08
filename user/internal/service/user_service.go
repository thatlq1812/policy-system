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
}

// userService implements UserService
type userService struct {
	repo           repository.UserRepository
	jwtSecret      string
	jwtExpiryHours int
}

// NewUserService creates a new service instance
func NewUserService(repo repository.UserRepository, jwtSecret string, jwtExpiryHours int) UserService {
	return &userService{
		repo:           repo,
		jwtSecret:      jwtSecret,
		jwtExpiryHours: jwtExpiryHours,
	}
}

// Register creates a new user with hashed password
func (s *userService) Register(ctx context.Context, phoneNumber, password, name, platformRole string) (*domain.User, string, error) {
	// 1. Validate input
	if err := s.validateRegisterInput(phoneNumber, password, platformRole); err != nil {
		return nil, "", fmt.Errorf("validation failed: %w", err)
	}

	// 2. Check if user already exists
	existingUser, err := s.repo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, "", fmt.Errorf("user with phone number %s already exists", phoneNumber)
	}

	// 3. Hash password
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Create user in database
	user, err := s.repo.Create(ctx, domain.CreateUserParams{
		PhoneNumber:  phoneNumber,
		PasswordHash: passwordHash,
		Name:         name,
		PlatformRole: platformRole,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// 5. Generate JWT token
	token, err := s.generateJWT(user.ID, user.PlatformRole)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}

// Login authenticates user and returns JWT token
func (s *userService) Login(ctx context.Context, phoneNumber, password string) (*domain.User, string, error) {
	// 1. Validate input
	if phoneNumber == "" || password == "" {
		return nil, "", fmt.Errorf("phone number and password are required")
	}

	// 2. Get user by phone number
	user, err := s.repo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	// 3. Verify password
	if err := s.verifyPassword(user.PasswordHash, password); err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	// 4. Generate JWT token
	token, err := s.generateJWT(user.ID, user.PlatformRole)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
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
