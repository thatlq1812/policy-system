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
            ip_address, user_agent
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, user_id, platform, document_id, document_name,
                  version_timestamp, agreed_at, agreed_file_url, consent_method,
                  ip_address, user_agent, is_deleted, deleted_at, created_at, updated_at
    `

	var consent domain.UserConsent
	err := r.db.QueryRow(ctx, query,
		params.UserID, params.Platform, params.DocumentID, params.DocumentName,
		params.VersionTimestamp, params.AgreedFileURL, params.ConsentMethod,
		params.IPAddress, params.UserAgent,
	).Scan(
		&consent.ID, &consent.UserID, &consent.Platform, &consent.DocumentID, &consent.DocumentName,
		&consent.VersionTimestamp, &consent.AgreedAt, &consent.AgreedFileURL, &consent.ConsentMethod,
		&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt,
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
            ip_address, user_agent
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, user_id, platform, document_id, document_name,
                  version_timestamp, agreed_at, agreed_file_url, consent_method,
                  ip_address, user_agent, is_deleted, deleted_at, created_at, updated_at
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
			&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt,
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
               ip_address, user_agent, is_deleted, deleted_at, created_at, updated_at
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
		&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt,
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
               ip_address, user_agent, is_deleted, deleted_at, created_at, updated_at
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
			&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt,
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
               ip_address, user_agent, is_deleted, deleted_at, created_at, updated_at
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
			&consent.IPAddress, &consent.UserAgent, &consent.IsDeleted, &consent.DeletedAt,
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
