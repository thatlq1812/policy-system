package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/thatlq1812/policy-system/gateway/configs"
	"github.com/thatlq1812/policy-system/gateway/internal/api"
	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"

	_ "github.com/thatlq1812/policy-system/gateway/docs" // swagger docs
)

// @title           Policy & Consent Management API Gateway
// @version         1.0
// @description     HTTP/REST API Gateway for Policy & Consent Management System. Provides authentication, user management, document management, and consent tracking.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// 1. Load configuration từ environment variables
	cfg := configs.Load()
	log.Printf("Starting API Gateway on port %d", cfg.Server.Port)

	// 2. Initialize gRPC clients với timeout
	userClient, err := clients.NewUserClient(cfg.Services.UserServiceAddr, cfg.GRPCcallTimeout)
	if err != nil {
		log.Fatalf("Failed to create user client: %v", err)
	}
	defer userClient.Close()

	documentClient, err := clients.NewDocumentClient(cfg.Services.DocumentServiceAddr, cfg.GRPCcallTimeout)
	if err != nil {
		log.Fatalf("Failed to create document client: %v", err)
	}
	defer documentClient.Close()

	consentClient, err := clients.NewConsentClient(cfg.Services.ConsentServiceAddr, cfg.GRPCcallTimeout)
	if err != nil {
		log.Fatalf("Failed to create consent client: %v", err)
	}
	defer consentClient.Close()

	// 3. Initialize API handlers
	// Giải thích: UserAPI cần access cả 3 clients để làm orchestration
	// - userClient: Register/Delete user
	// - documentClient: Get policies
	// - consentClient: Record consent
	userAPI := api.NewUserAPI(userClient, documentClient, consentClient)
	documentAPI := api.NewDocumentAPI(documentClient)
	consentAPI := api.NewConsentAPI(consentClient)
	adminAPI := api.NewAdminAPI(consentClient) // Admin endpoints

	// 4. Setup Gin router
	// Set Gin mode based on environment
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 5. Apply global middleware
	// CORS must be first
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: cfg.AllowedCredentials,
		MaxAge:           12 * time.Hour,
	}))

	// Custom logger middleware
	router.Use(middleware.GinLogger())

	// Recovery middleware
	router.Use(gin.Recovery())

	// 6. Register routes
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "api-gateway",
		})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes (no authentication required)
	public := router.Group("/api/v1")
	{
		// Authentication
		// Giải thích: Dùng enhanced versions với orchestration
		// - RegisterWithConsent: Register + Auto-consent orchestration (có rollback)
		// - LoginWithPendingCheck: Login + Check pending consents (graceful degradation)
		public.POST("/auth/register", userAPI.RegisterWithConsent)
		public.POST("/auth/login", userAPI.LoginWithPendingCheck)
		public.POST("/auth/refresh", userAPI.RefreshToken) // Token refresh
		public.POST("/auth/logout", userAPI.Logout)        // Logout

		// Documents (public access)
		public.POST("/policies", documentAPI.CreatePolicy)
		public.GET("/policies/latest", documentAPI.GetLatestPolicy)
	}

	// Protected routes (require JWT authentication)
	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		// User endpoints
		protected.POST("/user/change-password", userAPI.ChangePassword)

		// Consent endpoints
		protected.POST("/consents", consentAPI.RecordConsent)
		protected.POST("/consents/check", consentAPI.CheckConsent)
		protected.GET("/consents/user", consentAPI.GetUserConsents)
		protected.POST("/consents/pending", consentAPI.CheckPendingConsents)
		protected.POST("/consents/revoke", consentAPI.RevokeConsent)
	}

	// Admin routes (require JWT + Admin role)
	// TODO: Add role-based authorization middleware
	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	// admin.Use(middleware.RequireRole("Admin")) // TODO: Implement role check
	{
		// User management
		admin.GET("/users", userAPI.ListUsers)
		admin.GET("/stats/users", userAPI.GetUserStats)
		admin.POST("/create-admin", userAPI.CreateAdminUser) // Admin-only: Create new admin accounts

		// Consent statistics
		admin.GET("/stats/consents", adminAPI.GetConsentStats)
	}

	// 7. Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.GetAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. Graceful shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	go func() {
		log.Printf("API Gateway listening on %s", cfg.Server.GetAddr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-quit
	log.Println("Shutting down API Gateway...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
}
