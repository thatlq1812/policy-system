package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thatlq1812/policy-system/document/internal/domain"
)

type DocumentRepository interface {
	Create(ctx context.Context, params domain.CreateDocumentParams) (*domain.PolicyDocument, error)
	GetLatest(ctx context.Context, platform, documentName string) (*domain.PolicyDocument, error)
	GetHistory(ctx context.Context, platform, documentName string) ([]*domain.PolicyDocument, error)
}

type postgresDocumentRepository struct {
	db *pgxpool.Pool
}

func NewPostgresDocumentRepository(db *pgxpool.Pool) DocumentRepository {
	return &postgresDocumentRepository{db: db}
}

func (r *postgresDocumentRepository) Create(ctx context.Context, params domain.CreateDocumentParams) (*domain.PolicyDocument, error) {
	// 1. Generate UUID for ID
	id := uuid.New().String()
	// 2. Write INSERT query
	query := `
		INSERT INTO policy_documents (
			id, document_name, platform, is_mandatory, effective_timestamp, content_html, file_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, document_name, platform, is_mandatory, effective_timestamp, content_html, file_url, created_at, created_by`
	// 3. Execute query with QueryRow
	var doc domain.PolicyDocument
	// 4. Scan result into PolicyDocument struct
	err := r.db.QueryRow(ctx, query,
		id,
		params.DocumentName,
		params.Platform,
		params.IsMandatory,
		params.EffectiveTimestamp,
		params.ContentHTML,
		params.FileURL,
		params.CreatedBy,
	).Scan(
		&doc.ID,
		&doc.DocumentName,
		&doc.Platform,
		&doc.IsMandatory,
		&doc.EffectiveTimestamp,
		&doc.ContentHTML,
		&doc.FileURL,
		&doc.CreatedAt,
		&doc.CreatedBy,
	)
	// 5. Handle errors properly
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}
	return &doc, nil
}

func (r *postgresDocumentRepository) GetLatest(ctx context.Context, platform, documentName string) (*domain.PolicyDocument, error) {
	// 1. Write SELECT query with ORDER BY effective_timestamp DESC LIMIT 1
	// 2. Add WHERE clause for platform and document_name
	query := `
		SELECT id, document_name, platform, is_mandatory, effective_timestamp, content_html, file_url, created_at, created_by
		FROM policy_documents
		WHERE platform = $1 AND document_name = $2
		ORDER BY effective_timestamp DESC
		LIMIT 1
	`

	// 3. Execute query
	var doc domain.PolicyDocument
	err := r.db.QueryRow(ctx, query, platform, documentName).Scan(
		&doc.ID,
		&doc.DocumentName,
		&doc.Platform,
		&doc.IsMandatory,
		&doc.EffectiveTimestamp,
		&doc.ContentHTML,
		&doc.FileURL,
		&doc.CreatedAt,
		&doc.CreatedBy,
	)
	// 4. Handle pgx.ErrNoRows case
	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest document: %w", err)
	}

	return &doc, nil
}

func (r *postgresDocumentRepository) GetHistory(ctx context.Context, platform, documentName string) ([]*domain.PolicyDocument, error) {
	// SQL Query: Lấy tất cả versions, sắp xếp theo timestamp giảm dần
	query := `
		SELECT id, document_name, platform, is_mandatory, effective_timestamp, content_html, file_url, created_at, created_by
		FROM policy_documents
		WHERE platform = $1 AND document_name = $2
		ORDER BY effective_timestamp DESC
	`

	// Execute query với Query(), không phải QueryRow() vì ta cần nhiều hàng
	rows, err := r.db.Query(ctx, query, platform, documentName)
	if err != nil {
		return nil, fmt.Errorf("failed to get document history: %w", err)
	}
	defer rows.Close() // CRITICAL: Always close rows

	// Scan results vào slice
	var documents []*domain.PolicyDocument
	for rows.Next() {
		var doc domain.PolicyDocument
		err := rows.Scan(
			&doc.ID,
			&doc.DocumentName,
			&doc.Platform,
			&doc.IsMandatory,
			&doc.EffectiveTimestamp,
			&doc.ContentHTML,
			&doc.FileURL,
			&doc.CreatedAt,
			&doc.CreatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		documents = append(documents, &doc)
	}

	// Check for errors from iterating
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return documents, nil
}
