package domain

import "time"

// Platform constants - EXACT values accepted
const (
	PlatformClient   = "Client"
	PlatformMerchant = "Merchant"
	PlatformAdmin    = "Admin"
)

type PolicyDocument struct {
	ID                 string    `db:"id"`
	DocumentName       string    `db:"document_name"`
	Platform           string    `db:"platform"`
	IsMandatory        bool      `db:"is_mandatory"`
	EffectiveTimestamp int64     `db:"effective_timestamp"`
	ContentHTML        string    `db:"content_html"`
	FileURL            string    `db:"file_url"`
	CreatedAt          time.Time `db:"created_at"`
	CreatedBy          string    `db:"created_by"`
}

type CreateDocumentParams struct {
	DocumentName       string
	Platform           string
	IsMandatory        bool
	EffectiveTimestamp int64
	ContentHTML        string
	FileURL            string
	CreatedBy          string
}

// Helper function for validation
func IsValidPlatform(platform string) bool {
	return platform == PlatformClient || platform == PlatformMerchant || platform == PlatformAdmin
}
