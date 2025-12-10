package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/thatlq1812/policy-system/consent/internal/clients"
	configs "github.com/thatlq1812/policy-system/consent/internal/configs"
	"github.com/thatlq1812/policy-system/consent/internal/handler"
	"github.com/thatlq1812/policy-system/consent/internal/repository"
	"github.com/thatlq1812/policy-system/consent/internal/service"
	pb "github.com/thatlq1812/policy-system/shared/pkg/api/consent"
)

func main() {
	// 1. Load configuration
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Connect to database
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Verify connection
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// 3. Initialize Document Service client
	docClient, err := clients.NewDocumentClient(cfg.DocumentServiceURL)
	if err != nil {
		log.Fatalf("Failed to connect to document service: %v", err)
	}
	defer docClient.Close()
	log.Printf("Connected to document service at %s", cfg.DocumentServiceURL)

	// 4. Initialize layers
	consentRepo := repository.NewConsentRepository(dbPool)
	consentService := service.NewConsentService(consentRepo, docClient)
	consentHandler := handler.NewConsentHandler(consentService)

	// 5. Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterConsentServiceServer(grpcServer, consentHandler)

	// Enable reflection for testing with grpcurl
	reflection.Register(grpcServer)

	// 6. Start listening
	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	log.Printf("Consent service listening on port %s", cfg.GRPCPort)

	// 7. Graceful shutdown
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down consent service...")
	grpcServer.GracefulStop()
	log.Println("Consent service stopped")
}
