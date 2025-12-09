package handler

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/user"
	"github.com/thatlq1812/policy-system/user/internal/domain"
	"github.com/thatlq1812/policy-system/user/internal/service"
)

// UserHandler implements gRPC UserService interface
type UserHandler struct {
	pb.UnimplementedUserServiceServer
	service service.UserService
}

// NewUserHandler creates a new handler instance
func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Register creates a new user account with dual token authentication
func (h *UserHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Call service layer
	user, accessToken, refreshToken, accessExpiresAt, refreshExpiresAt, err := h.service.Register(
		ctx,
		req.PhoneNumber,
		req.Password,
		req.Name,
		req.PlatformRole,
	)
	if err != nil {
		// Map errors to gRPC status codes
		if strings.Contains(err.Error(), "validation") {
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Errorf(codes.AlreadyExists, "user already exists: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	// Convert domain to proto
	return &pb.RegisterResponse{
		User:                  domainToProto(user),
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
	}, nil
}

// Login handles user authentication
func (h *UserHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Step 1: Validate request
	if req.PhoneNumber == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "phone number and password are required")
	}

	// Step 2: Call service layer
	user, token, err := h.service.Login(ctx, req.PhoneNumber, req.Password)
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}

	// Step 3: Convert and return
	return &pb.LoginResponse{
		User:  domainUserToPb(user),
		Token: token,
	}, nil
}

// validateRegisterRequest validates registration request fields
func (h *UserHandler) validateRegisterRequest(req *pb.RegisterRequest) error {
	if req.PhoneNumber == "" {
		return status.Error(codes.InvalidArgument, "phone number is required")
	}
	if req.Password == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}
	if req.PlatformRole == "" {
		return status.Error(codes.InvalidArgument, "platform role is required")
	}
	return nil
}

// domainUserToPb converts domain User to protobuf User
func domainUserToPb(user *domain.User) *pb.User {
	return &pb.User{
		Id:           user.ID,
		PhoneNumber:  user.PhoneNumber,
		Name:         user.Name,
		PlatformRole: user.PlatformRole,
		CreatedAt:    user.CreatedAt.Unix(),
		UpdatedAt:    user.UpdatedAt.Unix(),
	}
}

// mapErrorToGRPCStatus maps service errors to gRPC status codes
func mapErrorToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for validation errors
	if contains(errMsg, "validation failed") ||
		contains(errMsg, "is required") ||
		contains(errMsg, "must be") {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Check for already exists errors
	if contains(errMsg, "already exists") {
		return status.Error(codes.AlreadyExists, err.Error())
	}

	// Check for authentication errors
	if contains(errMsg, "invalid credentials") {
		return status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Default to internal error
	return status.Error(codes.Internal, "internal server error")
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
