package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thatlq1812/policy-system/consent/internal/domain"
)

type ConsentRepository interface {
	// Create single consent
	Create(ctx context.Context, params domain.CreateConsentParams) (*domain.UserConsent, error)

	// Bulk create (transaction)
	CreateBulk(ctx context.Context, consents []domain.CreateConsentParams) ([]*domain.UserConsent, error)

	// Check if user has consented to document with min version
	HasConsented(ctx context.Context, userID, documentID string, minVersion int64) (*domain.UserConsent, error)

	// Get all consents of user
	GetUserConsents(ctx context.Context, userID string, includeDeleted bool) ([]*domain.UserConsent, error)

	// Get consents by user + document
	GetByUserAndDocument(ctx context.Context, userID, documentID string) ([]*domain.UserConsent, error)

	// Soft delete consent
	SoftDelete(ctx context.Context, userID, documentID string, versionTimestamp int64) error

	// GetExisting checks if a consent already exists (idempotent check)
	GetExisting(ctx context.Context, userID, documentID string, versionTimestamp int64) (*domain.UserConsent, error)

	// Transaction support for Phase 2
	BeginTx(ctx context.Context) (pgx.Tx, error)
	CreateWithTx(ctx context.Context, tx pgx.Tx, params domain.CreateConsentParams) (*domain.UserConsent, error)

	// Phase 2: History tracking methods
	GetConsentHistory(ctx context.Context, userID, documentID string) ([]*domain.UserConsent, error)
	MarkOldConsentsAsNotLatest(ctx context.Context, tx pgx.Tx, userID, documentID string) error

	// Phase 4: Statistics methods
	GetConsentStats(ctx context.Context, platform string) (map[string]int, error)
}

type consentRepository struct {
	db *pgxpool.Pool
}

func NewConsentRepository(db *pgxpool.Pool) ConsentRepository {
	return &consentRepository{db: db}
}

func (r *consentRepository) Create(ctx context.Context, params domain.CreateConsentParams) (*domain.UserConsent, error) {
	query := `
        INSERT INTO user_consents (
            user_id, platform, document_id, document_name, 
            version_timestamp, agreed_file_url, consent_method,
            ip_address, user_agent, is_latest
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE)
        RETURNING id, user_id, platform, document_id, document_name,
                  version_timestamp, agreed_at, agreed_file_url, consent_method,
                  ip_address, user_agent, is_deleted, deleted_at, is_latest,
                  revoked_at, revoked_reason, revoked_by, created_at, updated_at
    `

	var consent domain.UserConsent
	err := r.db.QueryRow(ctx, query,
		params.UserID, params.Platform, params.DocumentID, params.DocumentName,
		params.VersionTimestamp, params.AgreedFileURL, params.ConsentMethod,
		params.IPAddress, params.UserAgent,
	).Scan(
		&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
		&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
		&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt, &consent.IsLatest,
		&consent.RevokedAt, &consent.RevokedReason, &consent.RevokedBy,
		&consent.CreatedAt, &consent.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create consent: %w", err)
	}

	return &consent, nil
}

func (r *consentRepository) CreateBulk(ctx context.Context, consents []domain.CreateConsentParams) ([]*domain.UserConsent, error) {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO user_consents (
            user_id, platform, document_id, document_name, 
            version_timestamp, agreed_file_url, consent_method,
            ip_address, user_agent, is_latest
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE)
        RETURNING id, user_id, platform, document_id, document_name,
                  version_timestamp, agreed_at, agreed_file_url, consent_method,
                  ip_address, user_agent, is_deleted, deleted_at, is_latest,
                  revoked_at, revoked_reason, revoked_by, created_at, updated_at
    `

	var result []*domain.UserConsent

	for _, params := range consents {
		var consent domain.UserConsent
		err := tx.QueryRow(ctx, query,
			params.UserID, params.Platform, params.DocumentID, params.DocumentName,
			params.VersionTimestamp, params.AgreedFileURL, params.ConsentMethod,
			params.IPAddress, params.UserAgent,
		).Scan(
			&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
			&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
			&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt, &consent.IsLatest,
			&consent.RevokedAt, &consent.RevokedReason, &consent.RevokedBy,
			&consent.CreatedAt, &consent.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to insert consent: %w", err)
		}

		result = append(result, &consent)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

func (r *consentRepository) HasConsented(ctx context.Context, userID, documentID string, minVersion int64) (*domain.UserConsent, error) {
	query := `
        SELECT id, user_id, platform, document_id, document_name,
               version_timestamp, agreed_at, agreed_file_url, consent_method,
               ip_address, user_agent, is_deleted, deleted_at, is_latest,
               revoked_at, revoked_reason, revoked_by, created_at, updated_at
        FROM user_consents
        WHERE user_id = $1 
          AND document_id = $2 
          AND version_timestamp >= $3
          AND is_deleted = FALSE
        ORDER BY version_timestamp DESC
        LIMIT 1
    `

	var consent domain.UserConsent
	err := r.db.QueryRow(ctx, query, userID, documentID, minVersion).Scan(
		&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
		&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
		&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt, &consent.IsLatest,
		&consent.RevokedAt, &consent.RevokedReason, &consent.RevokedBy,
		&consent.CreatedAt, &consent.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check consent: %w", err)
	}

	return &consent, nil
}

func (r *consentRepository) GetUserConsents(ctx context.Context, userID string, includeDeleted bool) ([]*domain.UserConsent, error) {
	query := `
        SELECT id, user_id, platform, document_id, document_name,
               version_timestamp, agreed_at, agreed_file_url, consent_method,
               ip_address, user_agent, is_deleted, deleted_at, is_latest,
               revoked_at, revoked_reason, revoked_by, created_at, updated_at
        FROM user_consents
        WHERE user_id = $1
    `

	if !includeDeleted {
		query += " AND is_deleted = FALSE"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user consents: %w", err)
	}
	defer rows.Close()

	var consents []*domain.UserConsent
	for rows.Next() {
		var consent domain.UserConsent
		err := rows.Scan(
			&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
			&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
			&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt, &consent.IsLatest,
			&consent.RevokedAt, &consent.RevokedReason, &consent.RevokedBy,
			&consent.CreatedAt, &consent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent: %w", err)
		}
		consents = append(consents, &consent)
	}

	return consents, nil
}

func (r *consentRepository) GetByUserAndDocument(ctx context.Context, userID, documentID string) ([]*domain.UserConsent, error) {
	query := `
        SELECT id, user_id, platform, document_id, document_name,
               version_timestamp, agreed_at, agreed_file_url, consent_method,
               ip_address, user_agent, is_deleted, deleted_at, is_latest,
               revoked_at, revoked_reason, revoked_by, created_at, updated_at
        FROM user_consents
        WHERE user_id = $1 AND document_id = $2 AND is_deleted = FALSE
        ORDER BY version_timestamp DESC
    `

	rows, err := r.db.Query(ctx, query, userID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consents: %w", err)
	}
	defer rows.Close()

	var consents []*domain.UserConsent
	for rows.Next() {
		var consent domain.UserConsent
		err := rows.Scan(
			&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
			&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
			&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt, &consent.IsLatest,
			&consent.RevokedAt, &consent.RevokedReason, &consent.RevokedBy,
			&consent.CreatedAt, &consent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent: %w", err)
		}
		consents = append(consents, &consent)
	}

	return consents, nil
}

func (r *consentRepository) SoftDelete(ctx context.Context, userID, documentID string, versionTimestamp int64) error {
	query := `
        UPDATE user_consents
        SET is_deleted = TRUE, deleted_at = $4
        WHERE user_id = $1 AND document_id = $2 AND version_timestamp = $3 AND is_deleted = FALSE
    `

	result, err := r.db.Exec(ctx, query, userID, documentID, versionTimestamp, time.Now())
	if err != nil {
		return fmt.Errorf("failed to soft delete consent: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("consent not found or already deleted")
	}

	return nil
}

// GetExisting checks if a consent already exists
func (r *consentRepository) GetExisting(ctx context.Context, userID, documentID string, versionTimestamp int64) (*domain.UserConsent, error) {
	query := `
		SELECT id, user_id, platform, document_id, document_name,
		       version_timestamp, agreed_at, agreed_file_url, consent_method,
		       ip_address, user_agent, is_deleted, deleted_at, is_latest,
		       revoked_at, revoked_reason, revoked_by, created_at, updated_at
		FROM user_consents
		WHERE user_id = $1 AND document_id = $2 AND version_timestamp = $3 AND is_deleted = FALSE
		LIMIT 1
	`

	var consent domain.UserConsent
	err := r.db.QueryRow(ctx, query, userID, documentID, versionTimestamp).Scan(
		&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
		&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
		&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt, &consent.IsLatest,
		&consent.RevokedAt, &consent.RevokedReason, &consent.RevokedBy,
		&consent.CreatedAt, &consent.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get existing consent: %w", err)
	}
	return &consent, nil
}

// BeginTx starts a new transaction
func (r *consentRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// CreateWithTx creates a consent within a transaction
func (r *consentRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, params domain.CreateConsentParams) (*domain.UserConsent, error) {
	query := `
        INSERT INTO user_consents (
            user_id, platform, document_id, document_name, 
            version_timestamp, agreed_file_url, consent_method,
            ip_address, user_agent, is_latest
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE)
        RETURNING id, user_id, platform, document_id, document_name,
                  version_timestamp, agreed_at, agreed_file_url, consent_method,
                  ip_address, user_agent, is_deleted, deleted_at, is_latest,
                  revoked_at, revoked_reason, revoked_by, created_at, updated_at
    `

	var consent domain.UserConsent
	err := tx.QueryRow(ctx, query,
		params.UserID, params.Platform, params.DocumentID, params.DocumentName,
		params.VersionTimestamp, params.AgreedFileURL, params.ConsentMethod,
		params.IPAddress, params.UserAgent,
	).Scan(
		&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
		&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
		&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt, &consent.IsLatest,
		&consent.RevokedAt, &consent.RevokedReason, &consent.RevokedBy,
		&consent.CreatedAt, &consent.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create consent with tx: %w", err)
	}

	return &consent, nil
}

// GetConsentHistory retrieves all consent records for a user+document (including old versions)
func (r *consentRepository) GetConsentHistory(ctx context.Context, userID, documentID string) ([]*domain.UserConsent, error) {
	query := `
		SELECT id, user_id, platform, document_id, document_name,
		       version_timestamp, agreed_at, agreed_file_url, consent_method,
		       ip_address, user_agent, is_deleted, deleted_at, is_latest,
		       revoked_at, revoked_reason, revoked_by, created_at, updated_at
		FROM user_consents
		WHERE user_id = $1 AND document_id = $2
		ORDER BY version_timestamp DESC, agreed_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent history: %w", err)
	}
	defer rows.Close()

	var consents []*domain.UserConsent
	for rows.Next() {
		var c domain.UserConsent
		err := rows.Scan(
			&c.ID, &c.UserID, &c.Platform, &c.DocumentID, &c.DocumentName,
			&c.VersionTimestamp, &c.AgreedAt, &c.AgreedFileURL, &c.ConsentMethod,
			&c.IPAddress, &c.UserAgent, &c.IsDeleted, &c.DeletedAt, &c.IsLatest,
			&c.RevokedAt, &c.RevokedReason, &c.RevokedBy,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent history: %w", err)
		}
		consents = append(consents, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating consent history: %w", err)
	}

	return consents, nil
}

// MarkOldConsentsAsNotLatest marks all previous consents as not latest (for version upgrades)
func (r *consentRepository) MarkOldConsentsAsNotLatest(ctx context.Context, tx pgx.Tx, userID, documentID string) error {
	query := `
		UPDATE user_consents
		SET is_latest = FALSE,
		    updated_at = NOW()
		WHERE user_id = $1 
		  AND document_id = $2 
		  AND is_latest = TRUE
		  AND is_deleted = FALSE
	`

	_, err := tx.Exec(ctx, query, userID, documentID)
	if err != nil {
		return fmt.Errorf("failed to mark old consents as not latest: %w", err)
	}

	return nil
}

// GetConsentStats retrieves aggregated statistics about consents
func (r *consentRepository) GetConsentStats(ctx context.Context, platform string) (map[string]int, error) {
	stats := make(map[string]int)

	// Build platform filter
	platformFilter := ""
	var args []interface{}
	if platform != "" {
		platformFilter = " WHERE platform = $1"
		args = append(args, platform)
	}

	// Total consents
	query := "SELECT COUNT(*) FROM user_consents" + platformFilter
	var totalConsents int
	err := r.db.QueryRow(ctx, query, args...).Scan(&totalConsents)
	if err != nil {
		return nil, fmt.Errorf("failed to count total consents: %w", err)
	}
	stats["total_consents"] = totalConsents

	// Active consents (is_latest = TRUE, is_deleted = FALSE)
	activeFilter := platformFilter
	if activeFilter == "" {
		activeFilter = " WHERE is_latest = TRUE AND is_deleted = FALSE"
	} else {
		activeFilter += " AND is_latest = TRUE AND is_deleted = FALSE"
	}
	query = "SELECT COUNT(*) FROM user_consents" + activeFilter
	var activeConsents int
	err = r.db.QueryRow(ctx, query, args...).Scan(&activeConsents)
	if err != nil {
		return nil, fmt.Errorf("failed to count active consents: %w", err)
	}
	stats["active_consents"] = activeConsents

	// Revoked consents (revoked_at IS NOT NULL)
	revokedFilter := platformFilter
	if revokedFilter == "" {
		revokedFilter = " WHERE revoked_at IS NOT NULL"
	} else {
		revokedFilter += " AND revoked_at IS NOT NULL"
	}
	query = "SELECT COUNT(*) FROM user_consents" + revokedFilter
	var revokedConsents int
	err = r.db.QueryRow(ctx, query, args...).Scan(&revokedConsents)
	if err != nil {
		return nil, fmt.Errorf("failed to count revoked consents: %w", err)
	}
	stats["revoked_consents"] = revokedConsents

	// Consents by document
	query = "SELECT document_name, COUNT(*) FROM user_consents" + platformFilter + " GROUP BY document_name"
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query consents by document: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var docName string
		var count int
		if err := rows.Scan(&docName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan document stats: %w", err)
		}
		stats["doc_"+docName] = count
	}

	// Consents by platform (if no platform filter)
	if platform == "" {
		query = "SELECT platform, COUNT(*) FROM user_consents GROUP BY platform"
		rows, err = r.db.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to query consents by platform: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var plat string
			var count int
			if err := rows.Scan(&plat, &count); err != nil {
				return nil, fmt.Errorf("failed to scan platform stats: %w", err)
			}
			stats["platform_"+plat] = count
		}
	}

	// Consents by method
	query = "SELECT consent_method, COUNT(*) FROM user_consents" + platformFilter + " GROUP BY consent_method"
	rows, err = r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query consents by method: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var method string
		var count int
		if err := rows.Scan(&method, &count); err != nil {
			return nil, fmt.Errorf("failed to scan method stats: %w", err)
		}
		stats["method_"+method] = count
	}

	return stats, nil
}
