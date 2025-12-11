package handler

import (
	"context"
	"errors"

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
		return nil, mapErrorToGRPCStatus(err)
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

	// Use errors.Is() to check sentinel errors
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrVersionConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, domain.ErrNoActiveVersion):
		return status.Error(codes.NotFound, err.Error())
	default:
		// Default to internal error for unknown errors
		return status.Error(codes.Internal, "internal server error")
	}
}
