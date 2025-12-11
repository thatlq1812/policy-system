package handler

import (
	"context"
	"errors"
	"fmt"
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

	// Step 2: Call service layer with dual token support
	user, accessToken, refreshToken, accessExpiresAt, refreshExpiresAt, err := h.service.Login(
		ctx,
		req.PhoneNumber,
		req.Password,
	)
	if err != nil {
		return nil, mapErrorToGRPCStatus(err)
	}

	// Step 3: Return with dual tokens
	return &pb.LoginResponse{
		User:                  domainToProto(user),
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
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

// domainToProto converts domain User to protobuf User
func domainToProto(user *domain.User) *pb.User {
	return &pb.User{
		Id:           user.ID,
		PhoneNumber:  user.PhoneNumber,
		Name:         user.Name,
		PlatformRole: user.PlatformRole,
		CreatedAt:    user.CreatedAt.Unix(),
		UpdatedAt:    user.UpdatedAt.Unix(),
	}
}

// mapErrorToGRPCStatus maps service errors to gRPC status codes using Sentinel Errors
func mapErrorToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	// Check for sentinel errors using errors.Is()
	if errors.Is(err, domain.ErrNotFound) {
		return status.Error(codes.NotFound, "resource not found")
	}

	if errors.Is(err, domain.ErrAlreadyExists) {
		return status.Error(codes.AlreadyExists, err.Error())
	}

	if errors.Is(err, domain.ErrInvalidInput) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if errors.Is(err, domain.ErrUnauthorized) {
		return status.Error(codes.PermissionDenied, "unauthorized action")
	}

	if errors.Is(err, domain.ErrInvalidCredentials) {
		return status.Error(codes.Unauthenticated, "invalid credentials")
	}

	if errors.Is(err, domain.ErrTokenExpired) {
		return status.Error(codes.Unauthenticated, "token expired")
	}

	if errors.Is(err, domain.ErrTokenRevoked) {
		return status.Error(codes.Unauthenticated, "token revoked")
	}

	if errors.Is(err, domain.ErrTokenInvalid) {
		return status.Error(codes.Unauthenticated, "token invalid")
	}

	if errors.Is(err, domain.ErrInsufficientPermissions) {
		return status.Error(codes.PermissionDenied, "insufficient permissions")
	}

	// Default to internal error
	return status.Error(codes.Internal, "internal server error")
}

// RefreshToken generates new access token from valid refresh token
func (h *UserHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	// Validate input
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	// Call service layer with token rotation
	accessToken, newRefreshToken, accessExpiresAt, refreshExpiresAt, err := h.service.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "revoked") || strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "invalid") {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to refresh token: %v", err)
	}

	// Return new tokens with rotation
	return &pb.RefreshTokenResponse{
		AccessToken:           accessToken,
		RefreshToken:          newRefreshToken, // NEW: Rotated refresh token
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt, // NEW: New expiration time
	}, nil
}

// Logout revokes a refresh token and optionally blacklists access token
func (h *UserHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	// Validate input
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	// Call service layer with both refresh and access tokens
	err := h.service.Logout(ctx, req.RefreshToken, req.AccessToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to logout: %v", err)
	}

	message := "Successfully logged out"
	if req.AccessToken != "" {
		message = "Successfully logged out. Access token has been immediately revoked."
	}

	return &pb.LogoutResponse{
		Success: true,
		Message: message,
	}, nil
}

// GetUserProfile retrieves user profile by ID
func (h *UserHandler) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest) (*pb.GetUserProfileResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	user, err := h.service.GetUserProfile(ctx, req.UserId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user profile: %v", err)
	}

	return &pb.GetUserProfileResponse{
		User: domainToProto(user),
	}, nil
}

// UpdateProfile updates user's name and/or phone number
func (h *UserHandler) UpdateUserProfile(ctx context.Context, req *pb.UpdateUserProfileRequest) (*pb.UpdateUserProfileResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	var name, phoneNumber *string
	if req.Name != "" {
		name = &req.Name
	}

	if req.PhoneNumber != "" {
		phoneNumber = &req.PhoneNumber
	}

	user, err := h.service.UpdateUserProfile(ctx, req.UserId, name, phoneNumber)
	if err != nil {
		if strings.Contains(err.Error(), "validation") {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
		}
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Errorf(codes.AlreadyExists, "%s", err.Error())
		}
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "%s", err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to update user profile: %v", err)
	}

	return &pb.UpdateUserProfileResponse{
		User:    domainToProto(user),
		Message: "Profile updated successfully",
	}, nil
}

// ChangePassword changes user's password
func (h *UserHandler) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	if req.UserId == "" || req.OldPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "all fields are required")
	}

	err := h.service.ChangePassword(ctx, req.UserId, req.OldPassword, req.NewPassword)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			return nil, status.Errorf(codes.Unauthenticated, "old password is incorrect")
		}
		if strings.Contains(err.Error(), "validation") {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
		}
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to change password: %v", err)
	}
	return &pb.ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully. Please login again on all devices.",
	}, nil
}

// ListUsers return a paginated list of users with optional role filtering, admin only
func (h *UserHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	// TODO: Add admin role check from JWT in context metadata

	// Validate pagination parameters
	if req.Page < 1 {
		return nil, status.Error(codes.InvalidArgument, "page must be greater than 0")
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page_size must be between 1 and 100")
	}

	users, totalCount, totalPages, err := h.service.ListUsers(
		ctx,
		int(req.Page),
		int(req.PageSize),
		req.PlatformRole,
		req.IncludeDeleted,
	)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	pbUsers := make([]*pb.User, len(users))
	for i, user := range users {
		pbUsers[i] = domainToProto(user)
	}

	return &pb.ListUsersResponse{
		Users:       pbUsers,
		TotalCount:  int32(totalCount),
		CurrentPage: req.Page,
		PageSize:    req.PageSize,
		TotalPages:  int32(totalPages),
	}, nil
}

// SearchUsers searches users by phone or name (admin only)
func (h *UserHandler) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	// TODO: Add admin role check

	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	users, err := h.service.SearchUsers(ctx, req.Query, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search users: %v", err)
	}

	pbUsers := make([]*pb.User, len(users))
	for i, user := range users {
		pbUsers[i] = domainToProto(user)
	}

	return &pb.SearchUsersResponse{
		Users:      pbUsers,
		TotalCount: int32(len(users)),
	}, nil
}

// DeleteUser soft deletes a user (admin only)
func (h *UserHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	// TODO: Add admin role check

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	err := h.service.DeleteUser(ctx, req.UserId, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "user not found or already deleted")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	return &pb.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

// UpdateUserRole changes user's platform role (admin only)
func (h *UserHandler) UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleRequest) (*pb.UpdateUserRoleResponse, error) {
	// TODO: Add admin role check
	if req.UserId == "" || req.NewPlatformRole == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID and new platform role are required")
	}

	user, err := h.service.UpdateUserRole(ctx, req.UserId, req.NewPlatformRole)
	if err != nil {
		if strings.Contains(err.Error(), "validation") {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
		}

		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to update user role: %v", err)
	}

	return &pb.UpdateUserRoleResponse{
		User:    domainToProto(user),
		Message: "User role updated successfully",
	}, nil
}

// GetActiveSessions returns all active sessions for a user
func (h *UserHandler) GetActiveSessions(ctx context.Context, req *pb.GetActiveSessionsRequest) (*pb.GetActiveSessionsResponse, error) {
	// 1. Validate request
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 2. Call service
	tokens, count, err := h.service.GetActiveSessions(ctx, req.UserId)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get active sessions: %v", err))
	}

	// 3. Convert to proto
	var sessionInfos []*pb.RefreshTokenInfo
	for _, token := range tokens {
		deviceInfo := ""
		if token.DeviceInfo != nil {
			deviceInfo = *token.DeviceInfo
		}
		ipAddress := ""
		if token.IPAddress != nil {
			ipAddress = *token.IPAddress
		}
		sessionInfos = append(sessionInfos, &pb.RefreshTokenInfo{
			Id:         token.ID,
			DeviceInfo: deviceInfo,
			IpAddress:  ipAddress,
			CreatedAt:  token.CreatedAt.Unix(),
			ExpiresAt:  token.ExpiresAt.Unix(),
		})
	}

	return &pb.GetActiveSessionsResponse{
		Sessions:   sessionInfos,
		TotalCount: int32(count),
	}, nil
}

// LogoutAllDevices revokes all refresh tokens for a user
func (h *UserHandler) LogoutAllDevices(ctx context.Context, req *pb.LogoutAllDevicesRequest) (*pb.LogoutAllDevicesResponse, error) {
	// 1. Validate request
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 2. Call service
	revokedCount, err := h.service.LogoutAllDevices(ctx, req.UserId)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to logout all devices: %v", err))
	}

	return &pb.LogoutAllDevicesResponse{
		Success:      true,
		RevokedCount: int32(revokedCount),
		Message:      fmt.Sprintf("Successfully logged out from %d device(s)", revokedCount),
	}, nil
}

// RevokeSession revokes a specific session by token ID
func (h *UserHandler) RevokeSession(ctx context.Context, req *pb.RevokeSessionRequest) (*pb.RevokeSessionResponse, error) {
	// 1. Validate request
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.TokenId == "" {
		return nil, status.Error(codes.InvalidArgument, "token_id is required")
	}

	// 2. Call service
	err := h.service.RevokeSession(ctx, req.UserId, req.TokenId)
	if err != nil {
		if err.Error() == "token not found or already revoked" {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to revoke session: %v", err))
	}

	return &pb.RevokeSessionResponse{
		Success: true,
		Message: "Session revoked successfully",
	}, nil
}

// GetUserStats returns statistics about user accounts
func (h *UserHandler) GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest) (*pb.GetUserStatsResponse, error) {
	// Call service
	stats, err := h.service.GetUserStats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get user stats: %v", err))
	}

	return &pb.GetUserStatsResponse{
		TotalUsers:          int32(stats["total_users"]),
		TotalDeletedUsers:   int32(stats["total_deleted_users"]),
		TotalActiveSessions: int32(stats["total_active_sessions"]),
		UsersByRoleClient:   int32(stats["role_Client"]),
		UsersByRoleMerchant: int32(stats["role_Merchant"]),
		UsersByRoleAdmin:    int32(stats["role_Admin"]),
	}, nil
}

// IsTokenBlacklisted checks if a token JTI is in the blacklist
func (h *UserHandler) IsTokenBlacklisted(ctx context.Context, req *pb.IsTokenBlacklistedRequest) (*pb.IsTokenBlacklistedResponse, error) {
	if req.Jti == "" {
		return nil, status.Error(codes.InvalidArgument, "jti is required")
	}

	isBlacklisted, err := h.service.IsTokenBlacklisted(ctx, req.Jti)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check token blacklist: %v", err))
	}

	return &pb.IsTokenBlacklistedResponse{
		IsBlacklisted: isBlacklisted,
	}, nil
}
