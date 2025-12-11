package clients

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/document"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DocumentClient là wrapper cho gRPC document service client
type DocumentClient struct {
	conn    *grpc.ClientConn
	client  pb.DocumentServiceClient
	timeout time.Duration
}

// NewDocumentClient tạo kết nối tới Document Service
// addr: địa chỉ service (vd: "localhost:50051")
// timeout: thời gian timeout cho mỗi gRPC call
func NewDocumentClient(addr string, timeout time.Duration) (*DocumentClient, error) {
	// Tạo gRPC connection với NewClient (non-deprecated)
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to document service at %s: %w", addr, err)
	}

	log.Printf("Connected to Document Service at %s", addr)

	return &DocumentClient{
		conn:    conn,
		client:  pb.NewDocumentServiceClient(conn),
		timeout: timeout,
	}, nil
}

// CreatePolicy gọi CreatePolicy RPC
// Tự động add timeout vào context
func (c *DocumentClient) CreatePolicy(ctx context.Context, req *pb.CreateDocumentRequest) (*pb.CreateDocumentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.CreatePolicy(ctx, req)
}

// GetLatestPolicy gọi GetLatestPolicyByPlatform RPC
// Tự động add timeout vào context
func (c *DocumentClient) GetLatestPolicy(ctx context.Context, req *pb.GetLatestPolicyRequest) (*pb.GetLatestPolicyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.client.GetLatestPolicyByPlatform(ctx, req)
}

// Close đóng kết nối gRPC
func (c *DocumentClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
