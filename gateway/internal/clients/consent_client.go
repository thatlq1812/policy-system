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
	conn   *grpc.ClientConn
	client pb.ConsentServiceClient
}

// NewConsentClient tạo kết nối tới Consent Service
func NewConsentClient(addr string) (*ConsentClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to consent service at %s: %w", addr, err)
	}

	log.Printf("Connected to Consent Service at %s", addr)

	return &ConsentClient{
		conn:   conn,
		client: pb.NewConsentServiceClient(conn),
	}, nil
}

// RecordConsent gọi RecordConsent RPC
func (c *ConsentClient) RecordConsent(ctx context.Context, req *pb.RecordConsentRequest) (*pb.RecordConsentResponse, error) {
	return c.client.RecordConsent(ctx, req)
}

// CheckConsent gọi CheckConsent RPC
func (c *ConsentClient) CheckConsent(ctx context.Context, req *pb.CheckConsentRequest) (*pb.CheckConsentResponse, error) {
	return c.client.CheckConsent(ctx, req)
}

// GetUserConsents gọi GetUserConsents RPC
func (c *ConsentClient) GetUserConsents(ctx context.Context, req *pb.GetUserConsentsRequest) (*pb.GetUserConsentsResponse, error) {
	return c.client.GetUserConsents(ctx, req)
}

// CheckPendingConsents gọi CheckPendingConsents RPC
func (c *ConsentClient) CheckPendingConsents(ctx context.Context, req *pb.CheckPendingConsentsRequest) (*pb.CheckPendingConsentsResponse, error) {
	return c.client.CheckPendingConsents(ctx, req)
}

// RevokeConsent gọi RevokeConsent RPC
func (c *ConsentClient) RevokeConsent(ctx context.Context, req *pb.RevokeConsentRequest) (*pb.RevokeConsentResponse, error) {
	return c.client.RevokeConsent(ctx, req)
}

// Close đóng kết nối gRPC
func (c *ConsentClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
