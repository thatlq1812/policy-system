package domain

import "time"

// UserConsent represents a consent record
type UserConsent struct {
	ID               string     `db:"id"`
	UserID           string     `db:"user_id"`
	Platform         string     `db:"platform"`
	DocumentID       string     `db:"document_id"`
	DocumentName     string     `db:"document_name"`
	VersionTimestamp int64      `db:"version_timestamp"`
	AgreedAt         time.Time  `db:"agreed_at"`
	AgreedFileURL    *string    `db:"agreed_file_url"` // Pointer for NULL
	ConsentMethod    string     `db:"consent_method"`
	IPAddress        *string    `db:"ip_address"` // Pointer for NULL
	UserAgent        *string    `db:"user_agent"` // Pointer for NULL
	IsDeleted        bool       `db:"is_deleted"`
	DeletedAt        *time.Time `db:"deleted_at"` // Pointer for NULL
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

// CreateConsentParams for inserting new consent
type CreateConsentParams struct {
	UserID           string
	Platform         string
	DocumentID       string
	DocumentName     string
	VersionTimestamp int64
	AgreedFileURL    *string // Optional
	ConsentMethod    string
	IPAddress        *string // Optional
	UserAgent        *string // Optional
}

// ConsentMethod constants
const (
	ConsentMethodRegistration = "REGISTRATION"
	ConsentMethodUI           = "UI"
	ConsentMethodAPI          = "API"
)

// Platform constants (giống User Service)
const (
	PlatformClient   = "Client"
	PlatformMerchant = "Merchant"
)
