package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/thatlq1812/policy-system/consent/internal/domain"
	"github.com/thatlq1812/policy-system/consent/internal/service"
	pb "github.com/thatlq1812/policy-system/shared/pkg/api/consent"
)

type ConsentHandler struct {
	pb.UnimplementedConsentServiceServer
	service service.ConsentService
}

func NewConsentHandler(service service.ConsentService) *ConsentHandler {
	return &ConsentHandler{service: service}
}

// RecordConsent - Lưu đồng ý mới (single hoặc bulk)
func (h *ConsentHandler) RecordConsent(ctx context.Context, req *pb.RecordConsentRequest) (*pb.RecordConsentResponse, error) {
	// Validate request
	if req.UserId == "" || req.Platform == "" || req.ConsentMethod == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, platform, and consent_method are required")
	}

	if len(req.Consents) == 0 {
		return nil, status.Error(codes.InvalidArgument, "consents list cannot be empty")
	}

	// Convert protobuf to service params
	var consents []service.ConsentInput
	for _, c := range req.Consents {
		var agreedFileURL *string
		if c.AgreedFileUrl != "" {
			agreedFileURL = &c.AgreedFileUrl
		}

		consents = append(consents, service.ConsentInput{
			DocumentID:       c.DocumentId,
			DocumentName:     c.DocumentName,
			VersionTimestamp: c.VersionTimestamp,
			AgreedFileURL:    agreedFileURL,
		})
	}

	var ipAddress *string
	if req.IpAddress != "" {
		ipAddress = &req.IpAddress
	}

	var userAgent *string
	if req.UserAgent != "" {
		userAgent = &req.UserAgent
	}

	params := service.RecordConsentsParams{
		UserID:        req.UserId,
		Platform:      req.Platform,
		Consents:      consents,
		ConsentMethod: req.ConsentMethod,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
	}

	// Call service
	results, err := h.service.RecordConsents(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	// Convert to protobuf response
	var pbConsents []*pb.Consent
	for _, c := range results {
		pbConsents = append(pbConsents, domainToProto(c))
	}

	return &pb.RecordConsentResponse{
		Consents:      pbConsents,
		TotalRecorded: int32(len(pbConsents)),
	}, nil
}

// CheckConsent - Check user đã đồng ý document/version chưa
func (h *ConsentHandler) CheckConsent(ctx context.Context, req *pb.CheckConsentRequest) (*pb.CheckConsentResponse, error) {
	if req.UserId == "" || req.DocumentId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and document_id are required")
	}

	consent, err := h.service.CheckConsent(ctx, req.UserId, req.DocumentId, req.MinVersionTimestamp)
	if err != nil {
		return nil, mapError(err)
	}

	if consent == nil {
		return &pb.CheckConsentResponse{
			HasConsented:  false,
			LatestConsent: nil,
		}, nil
	}

	return &pb.CheckConsentResponse{
		HasConsented:  true,
		LatestConsent: domainToProto(consent),
	}, nil
}

// GetUserConsents - Lấy tất cả consents của user
func (h *ConsentHandler) GetUserConsents(ctx context.Context, req *pb.GetUserConsentsRequest) (*pb.GetUserConsentsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	consents, err := h.service.GetUserConsents(ctx, req.UserId, req.IncludeDeleted)
	if err != nil {
		return nil, mapError(err)
	}

	var pbConsents []*pb.Consent
	for _, c := range consents {
		pbConsents = append(pbConsents, domainToProto(c))
	}

	return &pb.GetUserConsentsResponse{
		Consents: pbConsents,
		Total:    int32(len(pbConsents)),
	}, nil
}

// CheckPendingConsents - Check policies nào user chưa consent
func (h *ConsentHandler) CheckPendingConsents(ctx context.Context, req *pb.CheckPendingConsentsRequest) (*pb.CheckPendingConsentsResponse, error) {
	if req.UserId == "" || req.Platform == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and platform are required")
	}

	// Convert protobuf policies to service type
	var latestPolicies []service.PolicyInfo
	for _, p := range req.LatestPolicies {
		latestPolicies = append(latestPolicies, service.PolicyInfo{
			DocumentID:       p.DocumentId,
			DocumentName:     p.DocumentName,
			VersionTimestamp: p.VersionTimestamp,
			Platform:         p.Platform,
		})
	}

	// Call service
	pending, err := h.service.CheckPendingConsents(ctx, req.UserId, latestPolicies)
	if err != nil {
		return nil, mapError(err)
	}

	// Convert to protobuf response
	var pbPending []*pb.PendingPolicy
	for _, p := range pending {
		pbPending = append(pbPending, &pb.PendingPolicy{
			DocumentId:       p.DocumentID,
			DocumentName:     p.DocumentName,
			VersionTimestamp: p.VersionTimestamp,
			Platform:         p.Platform,
		})
	}

	return &pb.CheckPendingConsentsResponse{
		PendingPolicies: pbPending,
		RequiresConsent: len(pbPending) > 0,
	}, nil
}

// RevokeConsent - Soft delete consent
func (h *ConsentHandler) RevokeConsent(ctx context.Context, req *pb.RevokeConsentRequest) (*pb.RevokeConsentResponse, error) {
	if req.UserId == "" || req.DocumentId == "" || req.VersionTimestamp == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id, document_id, and version_timestamp are required")
	}

	err := h.service.RevokeConsent(ctx, req.UserId, req.DocumentId, req.VersionTimestamp)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.RevokeConsentResponse{
		Success: true,
		Message: "Consent revoked successfully",
	}, nil
}

// Helper: Convert domain to protobuf
func domainToProto(c *domain.UserConsent) *pb.Consent {
	consent := &pb.Consent{
		Id:               c.ID,
		UserId:           c.UserID,
		Platform:         c.Platform,
		DocumentId:       c.DocumentID,
		DocumentName:     c.DocumentName,
		VersionTimestamp: c.VersionTimestamp,
		AgreedAt:         c.AgreedAt.Unix(),
		ConsentMethod:    c.ConsentMethod,
		IsDeleted:        c.IsDeleted,
		CreatedAt:        c.CreatedAt.Unix(),
		UpdatedAt:        c.UpdatedAt.Unix(),
	}

	// Handle nullable fields
	if c.AgreedFileURL != nil {
		consent.AgreedFileUrl = *c.AgreedFileURL
	}

	if c.IPAddress != nil {
		consent.IpAddress = *c.IPAddress
	}

	if c.UserAgent != nil {
		consent.UserAgent = *c.UserAgent
	}

	if c.DeletedAt != nil {
		consent.DeletedAt = c.DeletedAt.Unix()
	}

	return consent
}

// Helper: Map service errors to gRPC status codes
func mapError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Validation errors
	if contains(errMsg, "invalid") || contains(errMsg, "required") || contains(errMsg, "cannot be empty") {
		return status.Error(codes.InvalidArgument, errMsg)
	}

	// Not found errors
	if contains(errMsg, "not found") || contains(errMsg, "already deleted") {
		return status.Error(codes.NotFound, errMsg)
	}

	// Duplicate errors (unique constraint violation)
	if contains(errMsg, "duplicate") || contains(errMsg, "already exists") {
		return status.Error(codes.AlreadyExists, errMsg)
	}

	// Default: Internal error
	return status.Error(codes.Internal, "internal server error")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
