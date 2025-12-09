package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thatlq1812/policy-system/user/internal/domain"
)

// UserRepository defines database operations for users
type UserRepository interface {
	Create(ctx context.Context, params domain.CreateUserParams) (*domain.User, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error)
	GetByID(ctx context.Context, userID string) (*domain.User, error)
}

// postgresUserRepository implements UserRepository using PostgreSQL
type postgresUserRepository struct {
	db *pgxpool.Pool
}

// NewPostgresUserRepository creates a new repository instance
func NewPostgresUserRepository(db *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{db: db}
}

// Create inserts a new user into database
func (r *postgresUserRepository) Create(ctx context.Context, params domain.CreateUserParams) (*domain.User, error) {
	id := uuid.New().String()

	query := `
        INSERT INTO users (id, phone_number, password_hash, name, platform_role)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, phone_number, password_hash, name, platform_role, created_at, updated_at, is_deleted`

	var user domain.User
	err := r.db.QueryRow(ctx, query,
		id,
		params.PhoneNumber,
		params.PasswordHash,
		params.Name,
		params.PlatformRole,
	).Scan(
		&user.ID,
		&user.PhoneNumber,
		&user.PasswordHash,
		&user.Name,
		&user.PlatformRole,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsDeleted,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetByPhoneNumber retrieves an active user by phone number
func (r *postgresUserRepository) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
	query := `
        SELECT id, phone_number, password_hash, name, platform_role, 
               created_at, updated_at, is_deleted
        FROM users
        WHERE phone_number = $1 AND is_deleted = FALSE`

	var user domain.User
	err := r.db.QueryRow(ctx, query, phoneNumber).Scan(
		&user.ID,
		&user.PhoneNumber,
		&user.PasswordHash,
		&user.Name,
		&user.PlatformRole,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsDeleted,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // User not found is not an error
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByID retrieves a user by ID
func (r *postgresUserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, phone_number, password_hash, name, platform_role, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_deleted = FALSE
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.PhoneNumber,
		&user.PasswordHash,
		&user.Name,
		&user.PlatformRole,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}
