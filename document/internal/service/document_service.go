package service

import (
	"context"
	"fmt"

	"github.com/thatlq1812/policy-system/document/internal/domain"
	"github.com/thatlq1812/policy-system/document/internal/repository"
)

// DocumentService defines business operatons for policy documents
type DocumentService interface {
	CreatePolicy(ctx context.Context, params domain.CreateDocumentParams) (*domain.PolicyDocument, error)
	GetLatestPolicy(ctx context.Context, platform, documentName string) (*domain.PolicyDocument, error)
}

// documentService implements DocumentService
type documentService struct {
	repo repository.DocumentRepository
}

// NewDocumentService creates a new service instance
func NewDocumentService(repo repository.DocumentRepository) DocumentService {
	return &documentService{repo: repo}
}

// CreatePolicy creates a new policy document with validation
func (s *documentService) CreatePolicy(ctx context.Context, params domain.CreateDocumentParams) (*domain.PolicyDocument, error) {
	// Validate input
	if err := s.validateCreateParams(params); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Call repository
	doc, err := s.repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("service: failed to create policy: %w", err)
	}

	return doc, nil
}

// GetLatestPolicy retrieves the most recent policy version
func (s *documentService) GetLatestPolicy(ctx context.Context, platform, documentName string) (*domain.PolicyDocument, error) {
	// Validate input
	if platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if documentName == "" {
		return nil, fmt.Errorf("document_name is required")
	}

	// Call repository
	doc, err := s.repo.GetLatest(ctx, platform, documentName)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get latest policy: %w", err)
	}

	return doc, nil
}

// validateCreateParams validates document creation parameters
func (s *documentService) validateCreateParams(params domain.CreateDocumentParams) error {
	if params.DocumentName == "" {
		return fmt.Errorf("document_name is required")
	}

	if params.Platform != "Client" && params.Platform != "Merchant" {
		return fmt.Errorf("platform must be either 'Client' or 'Merchant'")
	}

	if params.EffectiveTimestamp <= 0 {
		return fmt.Errorf("effective_timestamp must be positive")
	}

	// At least one content source must be provided
	if params.ContentHTML == "" && params.FileURL == "" {
		return fmt.Errorf("either content_html or file_url must be provided")
	}

	if params.CreatedBy == "" {
		return fmt.Errorf("created_by is required")
	}

	return nil
}
