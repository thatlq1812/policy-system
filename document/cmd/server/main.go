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

	"github.com/thatlq1812/policy-system/document/internal/config"
	"github.com/thatlq1812/policy-system/document/internal/handler"
	"github.com/thatlq1812/policy-system/document/internal/repository"
	"github.com/thatlq1812/policy-system/document/internal/service"
	pb "github.com/thatlq1812/policy-system/shared/pkg/api/document"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Connect to database
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()

	// Verify database connection
	if err := dbpool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// 3. Initialize layers (bottom-up)
	repo := repository.NewPostgresDocumentRepository(dbpool)
	svc := service.NewDocumentService(repo)
	hdl := handler.NewDocumentHandler(svc)

	// 4. Setup gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.ServerPort)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDocumentServiceServer(grpcServer, hdl)

	//
	reflection.Register(grpcServer)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Document service listening on port %s", cfg.ServerPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
