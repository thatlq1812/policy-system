package domain

import "time"

// User represents a user entity in the system
type User struct {
	ID           string    `db:"id"`
	PhoneNumber  string    `db:"phone_number"`
	PasswordHash string    `db:"password_hash"`
	Name         string    `db:"name"`
	PlatformRole string    `db:"platform_role"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	IsDeleted    bool      `db:"is_deleted"`
}

// CreateUserParams holds parameters for creating a new user
type CreateUserParams struct {
	PhoneNumber  string
	PasswordHash string
	Name         string
	PlatformRole string
}

// UpdateUserParams holds parameters for updating user profile
type UpdateUserParams struct {
	ID   string
	Name string
	// Add other updatable fields here
}
