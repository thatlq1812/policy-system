package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"

	consentpb "github.com/thatlq1812/policy-system/shared/pkg/api/consent"
	docpb "github.com/thatlq1812/policy-system/shared/pkg/api/document"
	pb "github.com/thatlq1812/policy-system/shared/pkg/api/user"
)

// UserAPI xử lý các HTTP endpoints liên quan đến User
// Giải thích: Cần access đến cả 3 clients để làm orchestration
// - userClient: Tạo/xóa user
// - documentClient: Lấy policies mới nhất
// - consentClient: Record consent cho user
type UserAPI struct {
	userClient     *clients.UserClient
	documentClient *clients.DocumentClient
	consentClient  *clients.ConsentClient
}

// NewUserAPI tạo mới UserAPI handler với tất cả dependencies
// Giải thích: Constructor pattern - inject dependencies thay vì tạo bên trong
// Lợi ích: Dễ test (mock clients), dễ thay đổi implementation
func NewUserAPI(
	userClient *clients.UserClient,
	documentClient *clients.DocumentClient,
	consentClient *clients.ConsentClient,
) *UserAPI {
	return &UserAPI{
		userClient:     userClient,
		documentClient: documentClient,
		consentClient:  consentClient,
	}
}

// Register xử lý POST /api/v1/auth/register
func (api *UserAPI) Register(c *gin.Context) {
	var reqBody struct {
		PhoneNumber  string `json:"phone_number" binding:"required"`
		Password     string `json:"password" binding:"required,min=6"`
		Name         string `json:"name" binding:"required"`
		PlatformRole string `json:"platform_role" binding:"required,oneof=Client Merchant Admin"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	grpcReq := &pb.RegisterRequest{
		PhoneNumber:  reqBody.PhoneNumber,
		Password:     reqBody.Password,
		Name:         reqBody.Name,
		PlatformRole: reqBody.PlatformRole,
	}

	grpcResp, err := api.userClient.Register(c.Request.Context(), grpcReq)
	if err != nil {
		log.Printf("[Register] gRPC Error: %v | Request: phone=%s, role=%s", err, reqBody.PhoneNumber, reqBody.PlatformRole)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    "201",
		"message": "Registration successful",
		"data": gin.H{
			"user": gin.H{
				"id":            grpcResp.User.Id,
				"phone_number":  grpcResp.User.PhoneNumber,
				"name":          grpcResp.User.Name,
				"platform_role": grpcResp.User.PlatformRole,
				"created_at":    grpcResp.User.CreatedAt,
			},
			"access_token":             grpcResp.AccessToken,
			"refresh_token":            grpcResp.RefreshToken,
			"access_token_expires_at":  grpcResp.AccessTokenExpiresAt,
			"refresh_token_expires_at": grpcResp.RefreshTokenExpiresAt,
		},
	})
}

// Login xử lý POST /api/v1/auth/login
func (api *UserAPI) Login(c *gin.Context) {
	var reqBody struct {
		PhoneNumber string `json:"phone_number" binding:"required"`
		Password    string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	grpcReq := &pb.LoginRequest{
		PhoneNumber: reqBody.PhoneNumber,
		Password:    reqBody.Password,
	}

	grpcResp, err := api.userClient.Login(c.Request.Context(), grpcReq)
	if err != nil {
		log.Printf("[Login] gRPC Error: %v | Request: phone=%s", err, reqBody.PhoneNumber)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Login successful",
		"data": gin.H{
			"user": gin.H{
				"id":            grpcResp.User.Id,
				"phone_number":  grpcResp.User.PhoneNumber,
				"name":          grpcResp.User.Name,
				"platform_role": grpcResp.User.PlatformRole,
			},
			"access_token":             grpcResp.AccessToken,
			"refresh_token":            grpcResp.RefreshToken,
			"access_token_expires_at":  grpcResp.AccessTokenExpiresAt,
			"refresh_token_expires_at": grpcResp.RefreshTokenExpiresAt,
		},
	})
}

// LoginWithPendingCheck godoc
// @Summary      User Login with Pending Consent Check
// @Description  Authenticate user with phone number and password. Returns access token, refresh token, and checks for pending consents that require user acceptance.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body object{phone_number=string,password=string} true "Login credentials" example({"phone_number":"0901234567","password":"SecurePass@123"})
// @Success      200  {object}  object{code=string,message=string,data=object{user=object{id=string,phone_number=string,name=string,platform_role=string},access_token=string,refresh_token=string,access_token_expires_at=int64,refresh_token_expires_at=int64,requires_consent=boolean,pending_policies=array,consent_message=string}}
// @Failure      400  {object}  object{code=string,message=string} "Bad Request - Missing phone/password"
// @Failure      401  {object}  object{code=string,message=string} "Unauthorized - Invalid credentials"
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /auth/login [post]
func (api *UserAPI) LoginWithPendingCheck(c *gin.Context) {
	var reqBody struct {
		PhoneNumber string `json:"phone_number" binding:"required"`
		Password    string `json:"password" binding:"required"`
	}

	// Validate request
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	// ===== STEP 1: Authenticate User =====
	log.Printf("[LOGIN] Authenticating user phone=%s", reqBody.PhoneNumber)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userResp, err := api.userClient.Login(ctx, &pb.LoginRequest{
		PhoneNumber: reqBody.PhoneNumber,
		Password:    reqBody.Password,
	})

	if err != nil {
		// Step 1 failed - invalid credentials
		log.Printf("[LOGIN] Step 1 FAILED: Authentication failed for phone=%s", reqBody.PhoneNumber)
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    "401",
			"message": "Invalid phone number or password",
		})
		return
	}

	userID := userResp.User.Id
	log.Printf("[LOGIN] Step 1 SUCCESS: User %s authenticated", userID)

	// ===== STEP 2: Get Latest Policy (OPTIONAL - không block login) =====
	var pendingPolicies []map[string]interface{}

	policyCtx, policyCancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer policyCancel()

	policyResp, err := api.documentClient.GetLatestPolicy(policyCtx, &docpb.GetLatestPolicyRequest{
		Platform: userResp.User.PlatformRole,
	})

	if err != nil {
		// Log warning nhưng KHÔNG fail login
		log.Printf("[LOGIN WARNING] Step 2 FAILED: Cannot get latest policy: %v", err)
		log.Printf("[LOGIN] Continuing login without pending check (DocumentService unavailable)")
	} else if policyResp.Document != nil {
		log.Printf("[LOGIN] Step 2 SUCCESS: Got policy=%s, mandatory=%t",
			policyResp.Document.DocumentName,
			policyResp.Document.IsMandatory)

		// ===== STEP 3: Check Consent Status (OPTIONAL - không block login) =====
		consentCtx, consentCancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer consentCancel()

		checkResp, err := api.consentClient.CheckConsent(consentCtx, &consentpb.CheckConsentRequest{
			UserId:              userID,
			DocumentId:          policyResp.Document.Id,
			MinVersionTimestamp: policyResp.Document.EffectiveTimestamp,
		})

		if err != nil {
			// Log warning nhưng KHÔNG fail login
			log.Printf("[LOGIN WARNING] Step 3 FAILED: Cannot check consent: %v", err)
			log.Printf("[LOGIN] Continuing login without pending check (ConsentService unavailable)")
		} else if !checkResp.HasConsented {
			// User chưa consent policy này → Thêm vào pending list
			log.Printf("[LOGIN] Step 3 SUCCESS: User %s has pending consent for policy %s",
				userID, policyResp.Document.DocumentName)

			pendingPolicies = append(pendingPolicies, map[string]interface{}{
				"id":                  policyResp.Document.Id,
				"document_name":       policyResp.Document.DocumentName,
				"platform":            policyResp.Document.Platform,
				"is_mandatory":        policyResp.Document.IsMandatory,
				"effective_timestamp": policyResp.Document.EffectiveTimestamp,
				"content_summary":     truncateString(policyResp.Document.ContentHtml, 200),
				"file_url":            policyResp.Document.FileUrl,
			})
		} else {
			log.Printf("[LOGIN] Step 3 SUCCESS: User %s has already consented to latest policy", userID)
		}
	}

	// ===== STEP 4: Return Response =====
	requiresConsent := len(pendingPolicies) > 0

	// Prepare consent message for frontend
	consentMessage := ""
	if requiresConsent {
		if len(pendingPolicies) == 1 && pendingPolicies[0]["is_mandatory"].(bool) {
			consentMessage = "Please review and accept the updated policy to continue using our services."
		} else if requiresConsent {
			consentMessage = "We have updated our policies. Please review the changes."
		}
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Login successful",
		"data": gin.H{
			// User info và tokens
			"user": gin.H{
				"id":            userResp.User.Id,
				"phone_number":  userResp.User.PhoneNumber,
				"name":          userResp.User.Name,
				"platform_role": userResp.User.PlatformRole,
			},
			"access_token":             userResp.AccessToken,
			"refresh_token":            userResp.RefreshToken,
			"access_token_expires_at":  userResp.AccessTokenExpiresAt,
			"refresh_token_expires_at": userResp.RefreshTokenExpiresAt,

			// Pending consent info
			"requires_consent": requiresConsent,
			"pending_policies": pendingPolicies,
			"consent_message":  consentMessage,
		},
	})

	// Log final status
	if requiresConsent {
		log.Printf("[LOGIN] COMPLETE: User %s logged in with %d pending consent(s)",
			userID, len(pendingPolicies))
	} else {
		log.Printf("[LOGIN] COMPLETE: User %s logged in successfully (no pending consents)", userID)
	}
}

// RegisterWithConsent godoc
// @Summary      Register new user with auto-consent
// @Description  Register a new user account and automatically record consent for the latest policy document. Uses transaction rollback if consent fails. Note: Admin role cannot be registered via this endpoint for security reasons.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body object{phone_number=string,password=string,name=string,platform_role=string} true "Registration details. platform_role must be either 'Client' or 'Merchant' (Admin is not allowed)" example({"phone_number":"0901234567","password":"SecurePass@123","name":"John Doe","platform_role":"Client"})
// @Success      201  {object}  object{code=string,message=string,data=object{user=object{id=string,phone_number=string,name=string,platform_role=string,created_at=int64},access_token=string,refresh_token=string,access_token_expires_at=int64,refresh_token_expires_at=int64,policy_consented=object{document_id=string,version=string,consent_recorded=bool}}}
// @Failure      400  {object}  object{code=string,message=string} "Bad Request - Invalid phone/password format"
// @Failure      403  {object}  object{code=string,message=string} "Forbidden - Admin role cannot be registered publicly"
// @Failure      409  {object}  object{code=string,message=string} "Conflict - Phone number already exists"
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /auth/register [post]
func (api *UserAPI) RegisterWithConsent(c *gin.Context) {
	var reqBody struct {
		PhoneNumber  string `json:"phone_number" binding:"required"`
		Password     string `json:"password" binding:"required,min=6"`
		Name         string `json:"name" binding:"required"`
		PlatformRole string `json:"platform_role" binding:"required,oneof=Client Merchant"`
	}

	// Validate request body với Gin's built-in validator
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	// SECURITY: Block Admin registration from public API
	// Admin accounts must be created by existing Super Admin or via seed data
	if reqBody.PlatformRole == "Admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    "403",
			"message": "Admin accounts cannot be created through public registration. Please contact system administrator.",
		})
		return
	}

	// ===== STEP 1: Register User =====
	log.Printf("[ORCHESTRATION] Starting registration for phone=%s, role=%s",
		reqBody.PhoneNumber, reqBody.PlatformRole)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userResp, err := api.userClient.Register(ctx, &pb.RegisterRequest{
		PhoneNumber:  reqBody.PhoneNumber,
		Password:     reqBody.Password,
		Name:         reqBody.Name,
		PlatformRole: reqBody.PlatformRole,
	})

	if err != nil {
		// Step 1 failed - không cần rollback vì chưa tạo gì
		log.Printf("[ORCHESTRATION] Step 1 FAILED: User registration failed: %v", err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": "Failed to register user: " + msg,
		})
		return
	}

	userID := userResp.User.Id
	log.Printf("[ORCHESTRATION] Step 1 SUCCESS: User created with ID=%s", userID)

	// ===== STEP 2: Get Latest Policy =====
	policyCtx, policyCancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer policyCancel()

	policyResp, err := api.documentClient.GetLatestPolicy(policyCtx, &docpb.GetLatestPolicyRequest{
		Platform: reqBody.PlatformRole,
	})

	if err != nil {
		// Step 2 failed - PHẢI ROLLBACK Step 1
		log.Printf("[ORCHESTRATION] Step 2 FAILED: Cannot get latest policy: %v", err)
		log.Printf("[ORCHESTRATION] ROLLBACK: Deleting user %s", userID)

		api.rollbackUser(c.Request.Context(), userID, "policy fetch failed")

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "500",
			"message": "Failed to fetch policies. Registration has been rolled back.",
		})
		return
	}

	log.Printf("[ORCHESTRATION] Step 2 SUCCESS: Got policy=%s, mandatory=%t",
		policyResp.Document.DocumentName,
		policyResp.Document.IsMandatory)

	// ===== STEP 3: Record Consent (chỉ nếu mandatory) =====
	consentsRecorded := 0

	if policyResp.Document != nil && policyResp.Document.IsMandatory {
		consentCtx, consentCancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer consentCancel()

		_, err := api.consentClient.RecordConsent(consentCtx, &consentpb.RecordConsentRequest{
			UserId:   userID,
			Platform: reqBody.PlatformRole,
			Consents: []*consentpb.ConsentInput{{
				DocumentId:       policyResp.Document.Id,
				DocumentName:     policyResp.Document.DocumentName,
				VersionTimestamp: policyResp.Document.EffectiveTimestamp,
			}},
			ConsentMethod: "REGISTRATION",
			IpAddress:     c.ClientIP(), // Gin automatically handles X-Forwarded-For
			UserAgent:     c.GetHeader("User-Agent"),
		})

		if err != nil {
			// Step 3 failed - PHẢI ROLLBACK Step 1
			log.Printf("[ORCHESTRATION] Step 3 FAILED: Cannot record consent: %v", err)
			log.Printf("[ORCHESTRATION] ROLLBACK: Deleting user %s", userID)

			api.rollbackUser(c.Request.Context(), userID, "consent record failed")

			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    "500",
				"message": "Failed to record consent. Registration has been rolled back.",
			})
			return
		}

		consentsRecorded = 1
		log.Printf("[ORCHESTRATION] Step 3 SUCCESS: Consent recorded for user %s", userID)
	} else {
		log.Printf("[ORCHESTRATION] Step 3 SKIPPED: Policy is not mandatory")
	}

	// ===== SUCCESS: All steps completed =====
	log.Printf("[ORCHESTRATION] COMPLETE: User %s registered successfully with %d consents",
		userID, consentsRecorded)

	c.JSON(http.StatusCreated, gin.H{
		"code":    "201",
		"message": "Registration successful",
		"data": gin.H{
			"user": gin.H{
				"id":            userResp.User.Id,
				"phone_number":  userResp.User.PhoneNumber,
				"name":          userResp.User.Name,
				"platform_role": userResp.User.PlatformRole,
				"created_at":    userResp.User.CreatedAt,
			},
			"access_token":             userResp.AccessToken,
			"refresh_token":            userResp.RefreshToken,
			"access_token_expires_at":  userResp.AccessTokenExpiresAt,
			"refresh_token_expires_at": userResp.RefreshTokenExpiresAt,
			"consents_recorded":        consentsRecorded,
		},
	})
}

// rollbackUser xóa user khi orchestration fails
// Giải thích:
// - Được gọi khi Step 2 hoặc Step 3 fails
// - Đảm bảo không có "zombie users" (user tồn tại nhưng không có consent)
// - Nếu rollback cũng fail → Log critical error (cần manual cleanup)
func (api *UserAPI) rollbackUser(ctx context.Context, userID string, reason string) {
	deleteCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := api.userClient.DeleteUser(deleteCtx, &pb.DeleteUserRequest{
		UserId: userID,
		Reason: "Registration rollback - " + reason,
	})

	if err != nil {
		// CRITICAL ERROR: Rollback failed!
		// Giải thích: User vẫn tồn tại trong database nhưng không có consent
		// Action: Cần alert ops team để manual cleanup
		log.Printf("[ROLLBACK ERROR] CRITICAL: Failed to delete user %s: %v", userID, err)
		log.Printf("[ROLLBACK ERROR] Manual cleanup required for user_id=%s", userID)
		// TODO: Send alert to monitoring system (Slack, PagerDuty, etc.)
	} else {
		log.Printf("[ROLLBACK SUCCESS] User %s deleted successfully (reason: %s)", userID, reason)
	}
}

// ========== HELPER FUNCTIONS ==========

// errorResponse sends JSON error response với Gin
func errorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"code":    fmt.Sprintf("%d", statusCode),
		"message": message,
	})
}

// successResponse sends JSON success response với Gin
func successResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, gin.H{
		"code":    fmt.Sprintf("%d", statusCode),
		"message": message,
		"data":    data,
	})
}

// truncateString cắt string về maxLen characters và thêm "..." nếu quá dài
// Giải thích: Dùng để return content preview trong response, không return toàn bộ HTML
// Ví dụ: Policy content có thể dài hàng nghìn ký tự, chỉ cần 200 chars để preview
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ========== ADDITIONAL USER ENDPOINTS ==========

// RefreshToken godoc
// @Summary      Refresh Access Token
// @Description  Generate new access token using valid refresh token. Returns new access token and refresh token pair.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body object{refresh_token=string} true "Refresh token"
// @Success      200  {object}  object{code=string,message=string,data=object{access_token=string,refresh_token=string,access_token_expires_at=int64,refresh_token_expires_at=int64}}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /auth/refresh [post]
func (api *UserAPI) RefreshToken(c *gin.Context) {
	var reqBody struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Forward to User Service
	resp, err := api.userClient.RefreshToken(c.Request.Context(), &pb.RefreshTokenRequest{
		RefreshToken: reqBody.RefreshToken,
	})

	if err != nil {
		log.Printf("[REFRESH] Failed to refresh token: %v", err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	successResponse(c, http.StatusOK, "Token refreshed successfully", gin.H{
		"access_token":             resp.AccessToken,
		"refresh_token":            resp.RefreshToken,
		"access_token_expires_at":  resp.AccessTokenExpiresAt,
		"refresh_token_expires_at": resp.RefreshTokenExpiresAt,
	})
}

// Logout godoc
// @Summary      Logout user
// @Description  Revoke a specific refresh token and blacklist access token for immediate logout
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body object{refresh_token=string,access_token=string} true "Tokens to revoke. access_token is optional but recommended for immediate revocation"
// @Success      200  {object}  object{code=string,message=string}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Router       /auth/logout [post]
// Giải thích: Thu hồi refresh token và blacklist access token khi user logout
// Flow: Gateway forward request sang User Service → User Service invalidate refresh token + blacklist access token
func (api *UserAPI) Logout(c *gin.Context) {
	var reqBody struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
		AccessToken  string `json:"access_token"` // Optional but recommended
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Forward to User Service with both tokens
	resp, err := api.userClient.Logout(c.Request.Context(), &pb.LogoutRequest{
		RefreshToken: reqBody.RefreshToken,
		AccessToken:  reqBody.AccessToken,
	})

	if err != nil {
		log.Printf("[LOGOUT] Failed to logout: %v", err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	if !resp.Success {
		errorResponse(c, http.StatusBadRequest, resp.Message)
		return
	}

	successResponse(c, http.StatusOK, resp.Message, nil)
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Change password for the authenticated user. Requires old password verification.
// @Tags         User Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{old_password=string,new_password=string} true "Password change request. new_password must be at least 6 characters"
// @Success      200  {object}  object{code=string,message=string}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Router       /user/change-password [post]
func (api *UserAPI) ChangePassword(c *gin.Context) {
	// Get userID from JWT context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		errorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var reqBody struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Forward to User Service
	resp, err := api.userClient.ChangePassword(c.Request.Context(), &pb.ChangePasswordRequest{
		UserId:      userID.(string),
		OldPassword: reqBody.OldPassword,
		NewPassword: reqBody.NewPassword,
	})

	if err != nil {
		log.Printf("[CHANGE_PASSWORD] Failed for user %s: %v", userID, err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	if !resp.Success {
		errorResponse(c, http.StatusBadRequest, resp.Message)
		return
	}

	successResponse(c, http.StatusOK, resp.Message, nil)
}

// ========== ADMIN ENDPOINTS ==========

// ListUsers godoc
// @Summary      List all users (Admin only)
// @Description  Retrieve paginated list of users with optional platform role filtering. Requires Admin role.
// @Tags         Admin - User Management
// @Produce      json
// @Security     BearerAuth
// @Param        page         query     int     false  "Page number (default: 1)"
// @Param        page_size    query     int     false  "Items per page (default: 10, max: 100)"
// @Param        platform_role query    string  false  "Filter by platform role (Client, Merchant, Admin)"
// @Success      200  {object}  object{code=string,message=string,data=object{users=[]object,total_count=int32,page=int32,page_size=int32}}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      403  {object}  object{code=string,message=string}
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /admin/users [get]
func (api *UserAPI) ListUsers(c *gin.Context) {
	// Parse query parameters
	var req pb.ListUsersRequest

	// Default values
	req.Page = 1
	req.PageSize = 10

	if page := c.DefaultQuery("page", "1"); page != "" {
		var p int32
		if _, err := fmt.Sscanf(page, "%d", &p); err == nil && p > 0 {
			req.Page = p
		}
	}

	if pageSize := c.DefaultQuery("page_size", "10"); pageSize != "" {
		var ps int32
		if _, err := fmt.Sscanf(pageSize, "%d", &ps); err == nil && ps > 0 && ps <= 100 {
			req.PageSize = ps
		}
	}

	req.PlatformRole = c.Query("platform_role") // Optional filter

	// Forward to User Service
	resp, err := api.userClient.ListUsers(c.Request.Context(), &req)

	if err != nil {
		log.Printf("[ADMIN] Failed to list users: %v", err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	successResponse(c, http.StatusOK, "Users retrieved successfully", gin.H{
		"users":        resp.Users,
		"total_count":  resp.TotalCount,
		"current_page": resp.CurrentPage,
		"page_size":    resp.PageSize,
		"total_pages":  resp.TotalPages,
	})
}

// GetUserStats godoc
// @Summary      Get user statistics (Admin only)
// @Description  Retrieve comprehensive statistics about users including totals by role, active users, and session counts. Requires Admin role.
// @Tags         Admin - User Management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  object{code=string,message=string,data=object{total_users=int32,active_users=int32,deleted_users=int32,total_active_sessions=int32,users_by_role=map[string]int32}}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      403  {object}  object{code=string,message=string}
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /stats/users [get]
func (api *UserAPI) GetUserStats(c *gin.Context) {
	// Forward to User Service
	resp, err := api.userClient.GetUserStats(c.Request.Context(), &pb.GetUserStatsRequest{})

	if err != nil {
		log.Printf("[ADMIN] Failed to get user stats: %v", err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	successResponse(c, http.StatusOK, "Statistics retrieved successfully", resp)
}
