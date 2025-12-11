package domain

import "errors"

// Sentinel errors for User Service
// These errors should be checked using errors.Is() instead of string matching
var (
	// ErrNotFound indicates the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates the resource already exists (e.g., duplicate phone number)
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates validation failure on input parameters
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrUnauthorized indicates authentication/authorization failure
	ErrUnauthorized = errors.New("unauthorized action")

	// ErrInvalidCredentials indicates wrong username/password
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenExpired indicates the token has expired
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenRevoked indicates the token has been revoked
	ErrTokenRevoked = errors.New("token revoked")

	// ErrTokenInvalid indicates the token is malformed or invalid
	ErrTokenInvalid = errors.New("token invalid")

	// ErrInsufficientPermissions indicates the user lacks required permissions
	ErrInsufficientPermissions = errors.New("insufficient permissions")
)
