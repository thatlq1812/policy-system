package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/user"
)

// UserAPI xử lý các HTTP endpoints liên quan đến User
type UserAPI struct {
	client *clients.UserClient
}

// NewUserAPI tạo mới UserAPI handler
func NewUserAPI(client *clients.UserClient) *UserAPI {
	return &UserAPI{client: client}
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

	grpcResp, err := api.client.Register(c.Request.Context(), grpcReq)
	if err != nil {
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

	grpcResp, err := api.client.Login(c.Request.Context(), grpcReq)
	if err != nil {
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
