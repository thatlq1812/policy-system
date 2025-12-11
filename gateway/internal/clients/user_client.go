package clients

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserClient là wrapper cho gRPC user service client
type UserClient struct {
	conn    *grpc.ClientConn
	client  pb.UserServiceClient
	timeout time.Duration
}

// NewUserClient tạo kết nối tới User Service
// url: địa chỉ service (vd: "localhost:50052")
// timeout: thời gian timeout cho mỗi gRPC call
func NewUserClient(url string, timeout time.Duration) (*UserClient, error) {
	// Context với timeout để tránh wait vô hạn khi connect
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tạo gRPC connection với Dial + WithBlock để đảm bảo connection ready
	// insecure.NewCredentials() = không dùng TLS (dev mode)
	conn, err := grpc.DialContext(ctx, url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Wait until connection is ready
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	log.Printf("Connected to User Service at %s", url)

	return &UserClient{
		conn:    conn,
		client:  pb.NewUserServiceClient(conn),
		timeout: timeout,
	}, nil
}

// Register gọi Register RPC
// Tự động add timeout vào context để tránh request bị treo
func (c *UserClient) Register(ctx context.Context, req *pb.RegisterRequest, opts ...grpc.CallOption) (*pb.RegisterResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.Register(ctx, req, opts...)
}

// Login gọi Login RPC
// Tự động add timeout vào context
func (c *UserClient) Login(ctx context.Context, req *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.Login(ctx, req, opts...)
}

// DeleteUser gọi DeleteUser RPC (for rollback scenarios)
// Giải thích: Dùng để xóa user khi registration orchestration fails
// Ví dụ: User tạo thành công nhưng consent failed → Rollback bằng DeleteUser
func (c *UserClient) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest, opts ...grpc.CallOption) (*pb.DeleteUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.DeleteUser(ctx, req, opts...)
}

// RefreshToken gọi RefreshToken RPC
// Giải thích: Dùng refresh token để lấy access token mới
func (c *UserClient) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest, opts ...grpc.CallOption) (*pb.RefreshTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.RefreshToken(ctx, req, opts...)
}

// Logout gọi Logout RPC
// Giải thích: Thu hồi refresh token khi user logout
func (c *UserClient) Logout(ctx context.Context, req *pb.LogoutRequest, opts ...grpc.CallOption) (*pb.LogoutResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.Logout(ctx, req, opts...)
}

// ChangePassword gọi ChangePassword RPC
// Giải thích: Thay đổi password của user
func (c *UserClient) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest, opts ...grpc.CallOption) (*pb.ChangePasswordResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.ChangePassword(ctx, req, opts...)
}

// ListUsers gọi ListUsers RPC (Admin only)
// Giải thích: Lấy danh sách users với pagination
func (c *UserClient) ListUsers(ctx context.Context, req *pb.ListUsersRequest, opts ...grpc.CallOption) (*pb.ListUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.ListUsers(ctx, req, opts...)
}

// GetUserStats gọi GetUserStats RPC (Admin only)
// Giải thích: Lấy thống kê về users
func (c *UserClient) GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest, opts ...grpc.CallOption) (*pb.GetUserStatsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetUserStats(ctx, req, opts...)
}

// GetUserProfile gọi GetUserProfile RPC
func (c *UserClient) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest, opts ...grpc.CallOption) (*pb.GetUserProfileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetUserProfile(ctx, req, opts...)
}

// UpdateUserProfile gọi UpdateUserProfile RPC
func (c *UserClient) UpdateUserProfile(ctx context.Context, req *pb.UpdateUserProfileRequest, opts ...grpc.CallOption) (*pb.UpdateUserProfileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.UpdateUserProfile(ctx, req, opts...)
}

// SearchUsers gọi SearchUsers RPC (Admin only)
func (c *UserClient) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest, opts ...grpc.CallOption) (*pb.SearchUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.SearchUsers(ctx, req, opts...)
}

// HardDeleteUser gọi HardDeleteUser RPC (for rollback only)
func (c *UserClient) HardDeleteUser(ctx context.Context, req *pb.HardDeleteUserRequest, opts ...grpc.CallOption) (*pb.HardDeleteUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.HardDeleteUser(ctx, req, opts...)
}

// UpdateUserRole gọi UpdateUserRole RPC (Admin only)
func (c *UserClient) UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleRequest, opts ...grpc.CallOption) (*pb.UpdateUserRoleResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.UpdateUserRole(ctx, req, opts...)
}

// GetActiveSessions gọi GetActiveSessions RPC
func (c *UserClient) GetActiveSessions(ctx context.Context, req *pb.GetActiveSessionsRequest, opts ...grpc.CallOption) (*pb.GetActiveSessionsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetActiveSessions(ctx, req, opts...)
}

// LogoutAllDevices gọi LogoutAllDevices RPC
func (c *UserClient) LogoutAllDevices(ctx context.Context, req *pb.LogoutAllDevicesRequest, opts ...grpc.CallOption) (*pb.LogoutAllDevicesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.LogoutAllDevices(ctx, req, opts...)
}

// RevokeSession gọi RevokeSession RPC
func (c *UserClient) RevokeSession(ctx context.Context, req *pb.RevokeSessionRequest, opts ...grpc.CallOption) (*pb.RevokeSessionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.RevokeSession(ctx, req, opts...)
}

// IsTokenBlacklisted checks if access token is blacklisted
func (c *UserClient) IsTokenBlacklisted(ctx context.Context, req *pb.IsTokenBlacklistedRequest, opts ...grpc.CallOption) (*pb.IsTokenBlacklistedResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.IsTokenBlacklisted(ctx, req, opts...)
}

// Close đóng kết nối gRPC
func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
