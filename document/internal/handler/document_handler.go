package handler

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/thatlq1812/policy-system/document/internal/domain"
	"github.com/thatlq1812/policy-system/document/internal/service"
	pb "github.com/thatlq1812/policy-system/shared/pkg/api/document"
)

type DocumentHandler struct {
	pb.UnimplementedDocumentServiceServer
	service service.DocumentService
}

func NewDocumentHandler(service service.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: service}
}

func (h *DocumentHandler) CreatePolicy(ctx context.Context, req *pb.CreateDocumentRequest) (*pb.CreateDocumentResponse, error) {
	// Step 1: Convert protobuf to domain
	params := domain.CreateDocumentParams{
		DocumentName:       req.DocumentName,
		Platform:           req.Platform,
		IsMandatory:        req.IsMandatory,
		EffectiveTimestamp: req.EffectiveTimestamp, // Pass từ client, service sẽ handle nếu = 0
		ContentHTML:        req.ContentHtml,
		FileURL:            req.FileUrl,
		CreatedBy:          req.CreatedBy, // UpdaedBy map sang CreatedBy vì create record mới
	}

	// Step 2: Call service layer
	doc, err := h.service.CreatePolicy(ctx, params)
	if err != nil {
		// Step 3: Map errors to gRPC status codes
		return nil, mapErrorToGRPCStatus(err) // Dùng helper function đã có để map lỗi
	}

	// Step 4: Convert domain to protobuf
	return &pb.CreateDocumentResponse{
		Document: domainToPb(doc),
	}, nil
}

func (h *DocumentHandler) GetLatestPolicyByPlatform(ctx context.Context, req *pb.GetLatestPolicyRequest) (*pb.GetLatestPolicyResponse, error) {
	// Step 1: Call service
	doc, err := h.service.GetLatestPolicy(ctx, req.Platform, req.DocumentName)
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}

	// Step 2: Handle not found case
	if doc == nil {
		return nil, status.Error(codes.NotFound, "document not found")
	}

	// Step 3: Convert and return
	return &pb.GetLatestPolicyResponse{
		Document: domainToPb(doc),
	}, nil
}

func (h *DocumentHandler) UpdatePolicy(ctx context.Context, req *pb.UpdatePolicyRequest) (*pb.UpdatePolicyResponse, error) {
	// Step 1: Convert protobuf to domain
	params := domain.CreateDocumentParams{
		DocumentName:       req.DocumentName,
		Platform:           req.Platform,
		IsMandatory:        req.IsMandatory,
		EffectiveTimestamp: req.EffectiveTimestamp, // Pass từ client, service sẽ handle nếu = 0
		ContentHTML:        req.ContentHtml,
		FileURL:            req.FileUrl,
		CreatedBy:          req.UpdatedBy, // UpdatedBy map sang CreatedBy vì create record mới
	}

	// Step 2: Call service layer
	doc, err := h.service.UpdatePolicy(ctx, params)
	if err != nil {
		// Step 3: Map errors to gRPC status codes
		return nil, mapErrorToGRPCStatus(err) // Dùng helper function đã có để map lỗi
	}

	// Step 4: Convert domain to protobuf
	return &pb.UpdatePolicyResponse{
		Document: domainToPb(doc), // Use helper function để convert có sẵn
		Message:  "Policy document updated successfully. New version created.",
	}, nil
}

// Helper: Convert domain model to protobuf message
func domainToPb(doc *domain.PolicyDocument) *pb.PolicyDocument {
	return &pb.PolicyDocument{
		Id:                 doc.ID,
		DocumentName:       doc.DocumentName,
		Platform:           doc.Platform,
		IsMandatory:        doc.IsMandatory,
		EffectiveTimestamp: doc.EffectiveTimestamp,
		ContentHtml:        doc.ContentHTML,
		FileUrl:            doc.FileURL,
		CreatedAt:          doc.CreatedAt.Unix(),
		CreatedBy:          doc.CreatedBy,
	}
}

// GetPolicyHistory retrieves the version history of a policy document
func (h *DocumentHandler) GetPolicyHistory(ctx context.Context, req *pb.GetPolicyHistoryRequest) (*pb.GetPolicyHistoryResponse, error) {
	documents, err := h.service.GetPolicyHistory(ctx, req.Platform, req.DocumentName)
	if err != nil {
		if strings.Contains(err.Error(), "validation") || strings.Contains(err.Error(), "required") {
			return nil, status.Errorf(codes.InvalidArgument, "invalid input: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to get history: %v", err)
	}

	// Convert domain models to proto messages
	pbDocument := make([]*pb.PolicyDocument, len(documents))
	for i, doc := range documents {
		pbDocument[i] = &pb.PolicyDocument{
			Id:                 doc.ID,
			DocumentName:       doc.DocumentName,
			Platform:           doc.Platform,
			IsMandatory:        doc.IsMandatory,
			EffectiveTimestamp: doc.EffectiveTimestamp,
			ContentHtml:        doc.ContentHTML,
			FileUrl:            doc.FileURL,
			CreatedAt:          doc.CreatedAt.Unix(),
			CreatedBy:          doc.CreatedBy,
		}
	}

	return &pb.GetPolicyHistoryResponse{
		Documents:     pbDocument,
		TotalVersions: int32(len(pbDocument)),
	}, nil
}

// Helper: Map service errors to gRPC status codes
func mapErrorToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	// Check for validation errors
	errMsg := err.Error()
	if contains(errMsg, "validation failed") ||
		contains(errMsg, "is required") ||
		contains(errMsg, "must be") {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Check for not found errors
	if contains(errMsg, "not found") {
		return status.Error(codes.NotFound, err.Error())
	}

	// Default to internal error
	return status.Error(codes.Internal, "internal server error")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
