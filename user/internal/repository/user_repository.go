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
	Update(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID, newPasswordHash string) error

	// Admin operations
	ListUsers(ctx context.Context, params domain.ListUsersParams) ([]*domain.User, int, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]*domain.User, error)
	SoftDelete(ctx context.Context, userID, reason string) error
	HardDelete(ctx context.Context, userID string) error // NEW: For rollback scenarios
	UpdateRole(ctx context.Context, userID, platformRole string) (*domain.User, error)

	GetUserStats(ctx context.Context) (map[string]int, error)
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

// Update updates user profile (name and/or phone number)
func (r *postgresUserRepository) Update(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error) {
	// Build dynamic query based on provided fields
	query := "UPDATE users SET updated_at = CURRENT_TIMESTAMP"
	args := []interface{}{}
	argPos := 1

	if params.Name != nil {
		query += fmt.Sprintf(", name = $%d", argPos)
		args = append(args, *params.Name)
		argPos++
	}

	if params.PhoneNumber != nil {
		query += fmt.Sprintf(", phone_number = $%d", argPos)
		args = append(args, *params.PhoneNumber)
		argPos++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND is_deleted = FALSE RETURNING id, phone_number, password_hash, name, platform_role, created_at, updated_at, is_deleted", argPos)
	args = append(args, params.ID)

	var user domain.User
	err := r.db.QueryRow(ctx, query, args...).Scan(
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
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

// UpdatePassword updates the user's password
func (r *postgresUserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND is_deleted = FALSE
	`

	result, err := r.db.Exec(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// ListUsers returns paginated list of users with optional filtering
func (r *postgresUserRepository) ListUsers(ctx context.Context, params domain.ListUsersParams) ([]*domain.User, int, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if !params.IncludeDeleted {
		whereClause += " AND is_deleted = FALSE"
	}

	if params.PlatformRole != "" {
		whereClause += fmt.Sprintf(" AND platform_role = $%d", argPos)
		args = append(args, params.PlatformRole)
		argPos++
	}

	// Count total matching users
	countQuery := "SELECT COUNT(*) FROM users " + whereClause
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated data
	offset := (params.Page - 1) * params.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, phone_number, password_hash, name, platform_role, 
		       created_at, updated_at, is_deleted
		FROM users %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []*domain.User{}
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.Name,
			&user.PlatformRole, &user.CreatedAt, &user.UpdatedAt, &user.IsDeleted,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, totalCount, nil
}

// SearchUsers searches users by phone number or name using ILIKE
func (r *postgresUserRepository) SearchUsers(ctx context.Context, query string, limit int) ([]*domain.User, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	sqlQuery := `
		SELECT id, phone_number, password_hash, name, platform_role, 
		       created_at, updated_at, is_deleted
		FROM users
		WHERE is_deleted = FALSE 
		  AND (phone_number ILIKE $1 OR name ILIKE $1)
		ORDER BY created_at DESC
		LIMIT $2
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	users := []*domain.User{}
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.Name,
			&user.PlatformRole, &user.CreatedAt, &user.UpdatedAt, &user.IsDeleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

// SoftDelete marks user as deleted (is_deleted = TRUE)
func (r *postgresUserRepository) SoftDelete(ctx context.Context, userID, reason string) error {
	// Note: 'reason' parameter is for audit logging but not stored in users table
	// You may want to add a deletion_reason column or log it elsewhere
	query := `
		UPDATE users 
		SET is_deleted = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_deleted = FALSE
	`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	return nil
}

// HardDelete permanently removes user from database
// WARNING: This should only be used for rollback scenarios, not normal deletion
// Normal deletions should use SoftDelete to preserve audit trail
func (r *postgresUserRepository) HardDelete(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to hard delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateRole updates user's platform_role
func (r *postgresUserRepository) UpdateRole(ctx context.Context, userID, platformRole string) (*domain.User, error) {
	query := `
		UPDATE users 
		SET platform_role = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND is_deleted = FALSE
		RETURNING id, phone_number, password_hash, name, platform_role, 
		          created_at, updated_at, is_deleted
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, platformRole, userID).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.Name,
		&user.PlatformRole, &user.CreatedAt, &user.UpdatedAt, &user.IsDeleted,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return &user, nil
}

// GetUserStats returns statistics about users
func (r *postgresUserRepository) GetUserStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	// Total active users (not deleted)
	var totalUsers int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_deleted = FALSE").Scan(&totalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}
	stats["total_users"] = totalUsers

	// Total deleted users
	var totalDeleted int
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_deleted = TRUE").Scan(&totalDeleted)
	if err != nil {
		return nil, fmt.Errorf("failed to count deleted users: %w", err)
	}
	stats["total_deleted_users"] = totalDeleted

	// Users by role
	query := `
        SELECT platform_role, COUNT(*) 
        FROM users 
        WHERE is_deleted = FALSE 
        GROUP BY platform_role
    `
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count users by role: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return nil, fmt.Errorf("failed to scan role count: %w", err)
		}
		stats["role_"+role] = count
	}

	return stats, nil
}
