package domain

import "errors"

// Sentinel errors for Consent Service
var (
	// ErrNotFound indicates the requested consent was not found
	ErrNotFound = errors.New("consent not found")

	// ErrAlreadyExists indicates the consent already exists
	ErrAlreadyExists = errors.New("consent already exists")

	// ErrInvalidInput indicates validation failure on input parameters
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrInvalidConsent indicates the consent is invalid (e.g., revoking non-existent consent)
	ErrInvalidConsent = errors.New("invalid consent")

	// ErrDocumentNotFound indicates the referenced document does not exist
	ErrDocumentNotFound = errors.New("document not found")
)
