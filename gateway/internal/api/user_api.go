package api

import (
	"encoding/json"
	"net/http"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"
	"github.com/thatlq1812/policy-system/gateway/internal/response"

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

// RegisterRoutes đăng ký các routes cho user
func (api *UserAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", api.Register)
	mux.HandleFunc("POST /api/v1/auth/login", api.Login)
}

// Register xử lý POST /api/v1/auth/register
func (api *UserAPI) Register(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		PhoneNumber  string `json:"phone_number"`
		Password     string `json:"password"`
		Name         string `json:"name"`
		PlatformRole string `json:"platform_role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "Invalid request body")
		return
	}

	if reqBody.PhoneNumber == "" || reqBody.Password == "" || reqBody.Name == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "phone_number, password, and name are required")
		return
	}

	grpcReq := &pb.RegisterRequest{
		PhoneNumber:  reqBody.PhoneNumber,
		Password:     reqBody.Password,
		Name:         reqBody.Name,
		PlatformRole: reqBody.PlatformRole,
	}

	grpcResp, err := api.client.Register(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id":            grpcResp.User.Id,
			"phone_number":  grpcResp.User.PhoneNumber,
			"name":          grpcResp.User.Name,
			"platform_role": grpcResp.User.PlatformRole,
			"created_at":    grpcResp.User.CreatedAt,
		},
		"token": grpcResp.Token,
	}

	response.Success(w, data)
}

// Login xử lý POST /api/v1/auth/login
func (api *UserAPI) Login(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		PhoneNumber string `json:"phone_number"`
		Password    string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "Invalid request body")
		return
	}

	if reqBody.PhoneNumber == "" || reqBody.Password == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "phone_number and password are required")
		return
	}

	grpcReq := &pb.LoginRequest{
		PhoneNumber: reqBody.PhoneNumber,
		Password:    reqBody.Password,
	}

	grpcResp, err := api.client.Login(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id":            grpcResp.User.Id,
			"phone_number":  grpcResp.User.PhoneNumber,
			"name":          grpcResp.User.Name,
			"platform_role": grpcResp.User.PlatformRole,
		},
		"token": grpcResp.Token,
	}

	response.Success(w, data)
}
