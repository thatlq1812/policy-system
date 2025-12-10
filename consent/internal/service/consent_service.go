package service

import (
	"context"
	"fmt"

	"github.com/thatlq1812/policy-system/consent/internal/clients"
	"github.com/thatlq1812/policy-system/consent/internal/domain"
	"github.com/thatlq1812/policy-system/consent/internal/repository"
)

type ConsentService interface {
	// Record single or bulk consents
	RecordConsents(ctx context.Context, params RecordConsentsParams) ([]*domain.UserConsent, error)

	// Check if user has consented to document with specific version
	CheckConsent(ctx context.Context, userID, documentID string, minVersion int64) (*domain.UserConsent, error)

	// Get all consents of user
	GetUserConsents(ctx context.Context, userID string, includeDeleted bool) ([]*domain.UserConsent, error)

	// Check pending consents by comparing with latest policies
	CheckPendingConsents(ctx context.Context, userID string, latestPolicies []PolicyInfo) ([]PolicyInfo, error)

	// Revoke consent (soft delete)
	RevokeConsent(ctx context.Context, userID, documentID string, versionTimestamp int64) error

	// Phase 2: Get consent history for a user+document
	GetConsentHistory(ctx context.Context, userID, documentID string) ([]*domain.UserConsent, error)

	// Phase 4: Get consent statistics
	GetConsentStats(ctx context.Context, platform string) (map[string]int, error)
}

type consentService struct {
	repo      repository.ConsentRepository
	docClient *clients.DocumentClient // NEW: Document Service client
}

func NewConsentService(repo repository.ConsentRepository, docClient *clients.DocumentClient) ConsentService {
	return &consentService{
		repo:      repo,
		docClient: docClient,
	}
}

// RecordConsentsParams input for bulk consent
type RecordConsentsParams struct {
	UserID        string
	Platform      string
	Consents      []ConsentInput
	ConsentMethod string
	IPAddress     *string
	UserAgent     *string
}

type ConsentInput struct {
	DocumentID       string
	DocumentName     string
	VersionTimestamp int64
	AgreedFileURL    *string
}

// PolicyInfo for comparing with user consents
type PolicyInfo struct {
	DocumentID       string
	DocumentName     string
	VersionTimestamp int64
	Platform         string
}

func (s *consentService) RecordConsents(ctx context.Context, params RecordConsentsParams) ([]*domain.UserConsent, error) {
	// Validate platform
	if err := validatePlatform(params.Platform); err != nil {
		return nil, err
	}

	// Validate consent method
	if err := validateConsentMethod(params.ConsentMethod); err != nil {
		return nil, err
	}

	// Validate consents not empty
	if len(params.Consents) == 0 {
		return nil, fmt.Errorf("consents list cannot be empty")
	}

	var result []*domain.UserConsent

	// Convert to repository params
	var repoParams []domain.CreateConsentParams
	for _, c := range params.Consents {
		// Validate required fields
		if c.DocumentID == "" || c.DocumentName == "" || c.VersionTimestamp == 0 {
			return nil, fmt.Errorf("invalid consent input: document_id, document_name, and version_timestamp are required")
		}

		// PHASE 1: Verify document exists in Document Service
		if s.docClient != nil {
			doc, err := s.docClient.VerifyDocument(ctx, params.Platform, c.DocumentName)
			if err != nil {
				return nil, fmt.Errorf("document verification failed for %s: %w", c.DocumentName, err)
			}
			if doc == nil {
				return nil, fmt.Errorf("document not found: %s", c.DocumentName)
			}
			// Verify version matches
			if doc.EffectiveTimestamp != c.VersionTimestamp {
				return nil, fmt.Errorf("document version mismatch for %s: requested %d, current %d",
					c.DocumentName, c.VersionTimestamp, doc.EffectiveTimestamp)
			}
		}

		// PHASE 1: Check if consent already exists (idempotency)
		existing, err := s.repo.GetExisting(ctx, params.UserID, c.DocumentID, c.VersionTimestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing consent: %w", err)
		}

		if existing != nil {
			// Consent already exists, return it (idempotent)
			result = append(result, existing)
			continue
		}

		repoParams = append(repoParams, domain.CreateConsentParams{
			UserID:           params.UserID,
			Platform:         params.Platform,
			DocumentID:       c.DocumentID,
			DocumentName:     c.DocumentName,
			VersionTimestamp: c.VersionTimestamp,
			AgreedFileURL:    c.AgreedFileURL,
			ConsentMethod:    params.ConsentMethod,
			IPAddress:        params.IPAddress,
			UserAgent:        params.UserAgent,
		})
	}

	// Use bulk insert if multiple, single insert if one
	if len(repoParams) == 1 {
		consent, err := s.repo.Create(ctx, repoParams[0])
		if err != nil {
			return nil, fmt.Errorf("failed to record consent: %w", err)
		}
		return []*domain.UserConsent{consent}, nil
	}

	// Bulk insert with transaction
	consents, err := s.repo.CreateBulk(ctx, repoParams)
	if err != nil {
		return nil, fmt.Errorf("failed to record bulk consents: %w", err)
	}

	return consents, nil
}

func (s *consentService) CheckConsent(ctx context.Context, userID, documentID string, minVersion int64) (*domain.UserConsent, error) {
	if userID == "" || documentID == "" {
		return nil, fmt.Errorf("user_id and document_id are required")
	}

	consent, err := s.repo.HasConsented(ctx, userID, documentID, minVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to check consent: %w", err)
	}

	return consent, nil
}

func (s *consentService) GetUserConsents(ctx context.Context, userID string, includeDeleted bool) ([]*domain.UserConsent, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	consents, err := s.repo.GetUserConsents(ctx, userID, includeDeleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get user consents: %w", err)
	}

	return consents, nil
}

func (s *consentService) CheckPendingConsents(ctx context.Context, userID string, latestPolicies []PolicyInfo) ([]PolicyInfo, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// Get all user's active consents
	userConsents, err := s.repo.GetUserConsents(ctx, userID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user consents: %w", err)
	}

	// Build map of user's consents: documentID -> max version
	consentMap := make(map[string]int64)
	for _, consent := range userConsents {
		if existing, exists := consentMap[consent.DocumentID]; !exists || consent.VersionTimestamp > existing {
			consentMap[consent.DocumentID] = consent.VersionTimestamp
		}
	}

	// Find pending policies (not consented or consented to older version)
	var pending []PolicyInfo
	for _, policy := range latestPolicies {
		userVersion, hasConsented := consentMap[policy.DocumentID]
		if !hasConsented || userVersion < policy.VersionTimestamp {
			pending = append(pending, policy)
		}
	}

	return pending, nil
}

func (s *consentService) RevokeConsent(ctx context.Context, userID, documentID string, versionTimestamp int64) error {
	if userID == "" || documentID == "" || versionTimestamp == 0 {
		return fmt.Errorf("user_id, document_id, and version_timestamp are required")
	}

	err := s.repo.SoftDelete(ctx, userID, documentID, versionTimestamp)
	if err != nil {
		return fmt.Errorf("failed to revoke consent: %w", err)
	}

	return nil
}

// GetConsentHistory retrieves all historical consents for a user+document combination
func (s *consentService) GetConsentHistory(ctx context.Context, userID, documentID string) ([]*domain.UserConsent, error) {
	if userID == "" || documentID == "" {
		return nil, fmt.Errorf("user_id and document_id are required")
	}

	return s.repo.GetConsentHistory(ctx, userID, documentID)
}

// GetConsentStats retrieves aggregated statistics about consents
func (s *consentService) GetConsentStats(ctx context.Context, platform string) (map[string]int, error) {
	// Validate platform if provided
	if platform != "" {
		if err := validatePlatform(platform); err != nil {
			return nil, err
		}
	}

	return s.repo.GetConsentStats(ctx, platform)
}

// Validation helpers
func validatePlatform(platform string) error {
	if platform != domain.PlatformClient && platform != domain.PlatformMerchant {
		return fmt.Errorf("invalid platform: must be '%s' or '%s'", domain.PlatformClient, domain.PlatformMerchant)
	}
	return nil
}

func validateConsentMethod(method string) error {
	validMethods := []string{
		domain.ConsentMethodRegistration,
		domain.ConsentMethodUI,
		domain.ConsentMethodAPI,
	}

	for _, valid := range validMethods {
		if method == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid consent_method: must be one of %v", validMethods)
}
