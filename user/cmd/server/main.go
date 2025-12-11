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

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/user"
	configs "github.com/thatlq1812/policy-system/user/internal/configs"
	"github.com/thatlq1812/policy-system/user/internal/handler"
	"github.com/thatlq1812/policy-system/user/internal/repository"
	"github.com/thatlq1812/policy-system/user/internal/service"
)

func main() {
	// 1. Load configuration
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Connect to database
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbpool.Close()

	// Verify database connection
	if err := dbpool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// 3. Initialize layers (bottom-up: Repository → Service → Handler)
	userRepo := repository.NewPostgresUserRepository(dbpool)
	refreshTokenRepo := repository.NewPostgresRefreshTokenRepository(dbpool)
	blacklistRepo := repository.NewPostgresTokenBlacklistRepository(dbpool) // NEW
	svc := service.NewUserService(userRepo, refreshTokenRepo, blacklistRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
	hdl := handler.NewUserHandler(svc)

	// 4. Setup gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.ServerPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.ServerPort, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, hdl)

	// Enable gRPC reflection for grpcurl testing
	reflection.Register(grpcServer)

	// 5. Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nShutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	log.Printf("User service listening on port %s", cfg.ServerPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
