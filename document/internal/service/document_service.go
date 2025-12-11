package service

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/thatlq1812/policy-system/document/internal/domain"
	"github.com/thatlq1812/policy-system/document/internal/repository"
)

// DocumentService defines business operatons for policy documents
type DocumentService interface {
	CreatePolicy(ctx context.Context, params domain.CreateDocumentParams) (*domain.PolicyDocument, error)
	GetLatestPolicy(ctx context.Context, platform, documentName string) (*domain.PolicyDocument, error)
	// Update new method for UpdatePolicy
	UpdatePolicy(ctx context.Context, params domain.CreateDocumentParams) (*domain.PolicyDocument, error)

	// Update new method for GetPolicyHistory
	GetPolicyHistory(ctx context.Context, platform, documentName string) ([]*domain.PolicyDocument, error)
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
	// document_name is optional - if empty, will get any latest policy for platform

	// Call repository
	doc, err := s.repo.GetLatest(ctx, platform, documentName)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get latest policy: %w", err)
	}

	return doc, nil
}

// UpdatePolicy updates an existing policy document (creates new version)
func (s *documentService) UpdatePolicy(ctx context.Context, params domain.CreateDocumentParams) (*domain.PolicyDocument, error) {
	// Step 1: Basic validation (không check timestamp vì có thể = 0)
	if params.DocumentName == "" {
		return nil, fmt.Errorf("validation failed: document_name is required")
	}
	if params.Platform != "Client" && params.Platform != "Merchant" && params.Platform != "Admin" {
		return nil, fmt.Errorf("validation failed: platform must be one of: 'Client', 'Merchant', or 'Admin'")
	}
	if params.ContentHTML == "" && params.FileURL == "" {
		return nil, fmt.Errorf("validation failed: either content_html or file_url must be provided")
	}
	if params.CreatedBy == "" {
		return nil, fmt.Errorf("validation failed: created_by is required")
	}

	// Step 2: Check document cũ có tồn tại không
	existingDoc, err := s.repo.GetLatest(ctx, params.Platform, params.DocumentName)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing document: %w", err)
	}

	if existingDoc == nil {
		return nil, fmt.Errorf("document not found: cannot update non-existent document")
	}

	// Step 3: Generate effective_timestamp mới NẾU client không gửi (= 0)
	// KEY POINT: Kiểm tra nếu params.EffectiveTimestamp == 0 thì mới generate
	if params.EffectiveTimestamp == 0 {
		params.EffectiveTimestamp = time.Now().Unix()
	}

	// Step 4: Tạo record MỚI bằng cách gọi repo.Create()
	// Đây là KEY POINT: Update = Insert record mới
	newDoc, err := s.repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create new version: %w", err)
	}

	// Step 5: Return document mới
	return newDoc, nil
}

// GetPolicyHistory retrieves all versions of a policy document
func (s *documentService) GetPolicyHistory(ctx context.Context, platform, documentName string) ([]*domain.PolicyDocument, error) {
	// Validate input
	if platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if documentName == "" {
		return nil, fmt.Errorf("document_name is required")
	}

	// Validate platform enum
	if platform != "Client" && platform != "Merchant" && platform != "Admin" {
		return nil, fmt.Errorf("platform must be one of: 'Client', 'Merchant', or 'Admin'")
	}

	// Call repository
	documents, err := s.repo.GetHistory(ctx, platform, documentName)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get policy history: %w", err)
	}

	return documents, nil
}

// validateCreateParams validates document creation parameters
func (s *documentService) validateCreateParams(params domain.CreateDocumentParams) error {
	if params.DocumentName == "" {
		return fmt.Errorf("document_name is required")
	}

	if !domain.IsValidPlatform(params.Platform) {
		return fmt.Errorf("platform must be either '%s' or '%s' got '%s'", domain.PlatformClient, domain.PlatformMerchant, params.Platform)
	}

	if params.EffectiveTimestamp <= 0 {
		return fmt.Errorf("effective_timestamp must be positive")
	}

	// At least one content source must be provided
	if params.ContentHTML == "" && params.FileURL == "" {
		return fmt.Errorf("either content_html or file_url must be provided")
	}

	// Validate URL
	if params.FileURL != "" {
		if err := validateFileURL(params.FileURL); err != nil {
			return fmt.Errorf("invalid file_url: %w", err)
		}
	}

	if params.CreatedBy == "" {
		return fmt.Errorf("created_by is required")
	}

	return nil
}

// validateFileURL checks if the provided URL is valid (basic check)
func validateFileURL(fileURL string) error {
	// Basic validation: must start with http:// or https://
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Check scheme (must be http or https)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must start with http:// or https://, got: %s", parsedURL.Scheme)
	}

	// Check file extension (whitelist)
	allowedExtensions := []string{".pdf", ".docx", ".html", ".txt", ".md", ".jpg", ".png", ".jpeg"}
	ext := strings.ToLower(filepath.Ext(parsedURL.Path))
	isAllowed := false
	for _, allowed := range allowedExtensions {
		if ext == allowed {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("file extension must be one of %v, got %s", allowedExtensions, ext)
	}
	return nil
}
