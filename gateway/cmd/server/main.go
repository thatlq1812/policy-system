package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thatlq1812/policy-system/gateway/configs"
	"github.com/thatlq1812/policy-system/gateway/internal/api"
	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"
)

func main() {
	// 1. Load configuration từ environment variables
	cfg := configs.Load()
	log.Printf("Starting API Gateway on port %d", cfg.Server.Port)

	// 2. Initialize gRPC clients
	userClient, err := clients.NewUserClient(cfg.Services.UserServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create user client: %v", err)
	}
	defer userClient.Close()

	documentClient, err := clients.NewDocumentClient(cfg.Services.DocumentServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create document client: %v", err)
	}
	defer documentClient.Close()

	consentClient, err := clients.NewConsentClient(cfg.Services.ConsentServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create consent client: %v", err)
	}
	defer consentClient.Close()

	// 3. Initialize API handlers
	userAPI := api.NewUserAPI(userClient)
	documentAPI := api.NewDocumentAPI(documentClient)
	consentAPI := api.NewConsentAPI(consentClient)

	// 4. Setup HTTP router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"api-gateway"}`))
	})

	// Register API routes
	userAPI.RegisterRoutes(mux)
	documentAPI.RegisterRoutes(mux)
	consentAPI.RegisterRoutes(mux)

	// 5. Apply middleware
	// Thứ tự middleware: Logger (ngoài cùng) -> Auth -> Handler
	handler := middleware.Logger(mux)

	// 6. Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 7. Graceful shutdown setup
	// Channel để nhận signal từ OS
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Chạy server trong goroutine
	go func() {
		log.Printf("API Gateway listening on :%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 8. Chờ signal shutdown
	<-quit
	log.Println("Shutting down API Gateway...")

	// Graceful shutdown với timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
}
