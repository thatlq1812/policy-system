package clients

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/document"
)

type DocumentClient struct {
	conn   *grpc.ClientConn
	client pb.DocumentServiceClient
}

func NewDocumentClient(address string) (*DocumentClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to document service: %w", err)
	}

	return &DocumentClient{
		conn:   conn,
		client: pb.NewDocumentServiceClient(conn),
	}, nil
}

func (c *DocumentClient) Close() error {
	return c.conn.Close()
}

// VerifyDocument checks if document exists and gets its info
func (c *DocumentClient) VerifyDocument(ctx context.Context, platform, documentName string) (*pb.PolicyDocument, error) {
	resp, err := c.client.GetLatestPolicyByPlatform(ctx, &pb.GetLatestPolicyRequest{
		Platform:     platform,
		DocumentName: documentName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to verify document: %w", err)
	}

	return resp.Document, nil
}
