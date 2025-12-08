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
	conn   *grpc.ClientConn
	client pb.DocumentServiceClient
}

// NewDocumentClient tạo kết nối tới Document Service
func NewDocumentClient(addr string) (*DocumentClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to document service at %s: %w", addr, err)
	}

	log.Printf("Connected to Document Service at %s", addr)

	return &DocumentClient{
		conn:   conn,
		client: pb.NewDocumentServiceClient(conn),
	}, nil
}

// CreatePolicy gọi CreatePolicy RPC
func (c *DocumentClient) CreatePolicy(ctx context.Context, req *pb.CreateDocumentRequest) (*pb.CreateDocumentResponse, error) {
	return c.client.CreatePolicy(ctx, req)
}

// GetLatestPolicy gọi GetLatestPolicyByPlatform RPC
func (c *DocumentClient) GetLatestPolicy(ctx context.Context, req *pb.GetLatestPolicyRequest) (*pb.GetLatestPolicyResponse, error) {
	return c.client.GetLatestPolicyByPlatform(ctx, req)
}

// Close đóng kết nối gRPC
func (c *DocumentClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
