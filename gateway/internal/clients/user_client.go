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
	// Tạo gRPC connection với NewClient (non-deprecated)
	// insecure.NewCredentials() = không dùng TLS (dev mode)
	conn, err := grpc.NewClient(url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user client: %w", err)
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
func (c *UserClient) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.Register(ctx, req)
}

// Login gọi Login RPC
// Tự động add timeout vào context
func (c *UserClient) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.Login(ctx, req)
}

// Close đóng kết nối gRPC
func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
