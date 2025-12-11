package domain

import "errors"

// Sentinel errors for Document Service
var (
	// ErrNotFound indicates the requested document was not found
	ErrNotFound = errors.New("document not found")

	// ErrAlreadyExists indicates the document already exists
	ErrAlreadyExists = errors.New("document already exists")

	// ErrInvalidInput indicates validation failure on input parameters
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrVersionConflict indicates a version conflict (stale version)
	ErrVersionConflict = errors.New("version conflict")

	// ErrNoActiveVersion indicates no active version exists for the document
	ErrNoActiveVersion = errors.New("no active version")
)
