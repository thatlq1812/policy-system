package clients

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/consent"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ConsentClient là wrapper cho gRPC consent service client
type ConsentClient struct {
	conn    *grpc.ClientConn
	client  pb.ConsentServiceClient
	timeout time.Duration
}

// NewConsentClient tạo kết nối tới Consent Service
// addr: địa chỉ service (vd: "localhost:50053")
// timeout: thời gian timeout cho mỗi gRPC call
func NewConsentClient(addr string, timeout time.Duration) (*ConsentClient, error) {
	// Context với timeout để tránh wait vô hạn khi connect
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tạo gRPC connection với DialContext + WithBlock
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Wait until connection is ready
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to consent service at %s: %w", addr, err)
	}

	log.Printf("Connected to Consent Service at %s", addr)

	return &ConsentClient{
		conn:    conn,
		client:  pb.NewConsentServiceClient(conn),
		timeout: timeout,
	}, nil
}

// RecordConsent gọi RecordConsent RPC
// Tự động add timeout vào context
func (c *ConsentClient) RecordConsent(ctx context.Context, req *pb.RecordConsentRequest) (*pb.RecordConsentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.RecordConsent(ctx, req)
}

// CheckConsent gọi CheckConsent RPC
// Tự động add timeout vào context
func (c *ConsentClient) CheckConsent(ctx context.Context, req *pb.CheckConsentRequest) (*pb.CheckConsentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.CheckConsent(ctx, req)
}

// GetUserConsents gọi GetUserConsents RPC
// Tự động add timeout vào context
func (c *ConsentClient) GetUserConsents(ctx context.Context, req *pb.GetUserConsentsRequest) (*pb.GetUserConsentsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetUserConsents(ctx, req)
}

// CheckPendingConsents gọi CheckPendingConsents RPC
// Tự động add timeout vào context
func (c *ConsentClient) CheckPendingConsents(ctx context.Context, req *pb.CheckPendingConsentsRequest) (*pb.CheckPendingConsentsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.CheckPendingConsents(ctx, req)
}

// RevokeConsent gọi RevokeConsent RPC
// Tự động add timeout vào context
func (c *ConsentClient) RevokeConsent(ctx context.Context, req *pb.RevokeConsentRequest) (*pb.RevokeConsentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.RevokeConsent(ctx, req)
}

// GetConsentStats gọi GetConsentStats RPC (Admin only)
// Giải thích: Lấy thống kê về consents
func (c *ConsentClient) GetConsentStats(ctx context.Context, req *pb.GetConsentStatsRequest) (*pb.GetConsentStatsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetConsentStats(ctx, req)
}

// Close đóng kết nối gRPC
func (c *ConsentClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
