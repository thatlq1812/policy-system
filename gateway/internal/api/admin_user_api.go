package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/thatlq1812/policy-system/gateway/internal/middleware"
	pb "github.com/thatlq1812/policy-system/shared/pkg/api/user"
)

// CreateAdminUser godoc
// @Summary      Create new admin user (Super Admin only)
// @Description  Create a new admin account. This endpoint can only be accessed by existing admin users. Admin accounts cannot be created through public registration for security reasons.
// @Tags         Admin - User Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{phone_number=string,password=string,name=string} true "Admin user details"
// @Success      201  {object}  object{code=string,message=string,data=object{user=object{id=string,phone_number=string,name=string,platform_role=string,created_at=int64}}}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      403  {object}  object{code=string,message=string}
// @Failure      409  {object}  object{code=string,message=string}
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /admin/create-admin [post]
func (api *UserAPI) CreateAdminUser(c *gin.Context) {
	// SECURITY CHECK: Only Admin can create new Admin
	userRole, exists := c.Get("platform_role")
	if !exists || userRole != "Admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    "403",
			"message": "Only administrators can create admin accounts",
		})
		return
	}

	var reqBody struct {
		PhoneNumber string `json:"phone_number" binding:"required"`
		Password    string `json:"password" binding:"required,min=6"`
		Name        string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[ADMIN] Creating new admin account: phone=%s, name=%s", reqBody.PhoneNumber, reqBody.Name)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Call User Service to create admin
	grpcReq := &pb.RegisterRequest{
		PhoneNumber:  reqBody.PhoneNumber,
		Password:     reqBody.Password,
		Name:         reqBody.Name,
		PlatformRole: "Admin", // Force Admin role
	}

	grpcResp, err := api.userClient.Register(ctx, grpcReq)
	if err != nil {
		log.Printf("[ADMIN] Failed to create admin: %v", err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	log.Printf("[ADMIN] Successfully created admin user: id=%s", grpcResp.User.Id)

	c.JSON(http.StatusCreated, gin.H{
		"code":    "201",
		"message": "Admin user created successfully",
		"data": gin.H{
			"user": gin.H{
				"id":            grpcResp.User.Id,
				"phone_number":  grpcResp.User.PhoneNumber,
				"name":          grpcResp.User.Name,
				"platform_role": grpcResp.User.PlatformRole,
				"created_at":    grpcResp.User.CreatedAt,
			},
			// Note: Do NOT return tokens for admin creation
			// New admin should login separately with their credentials
		},
	})
}
