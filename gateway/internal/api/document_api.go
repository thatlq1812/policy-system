package api

import (
	"encoding/json"
	"net/http"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"
	"github.com/thatlq1812/policy-system/gateway/internal/response"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/document"
)

// DocumentAPI xử lý các HTTP endpoints liên quan đến Document
type DocumentAPI struct {
	client *clients.DocumentClient
}

// NewDocumentAPI tạo mới DocumentAPI handler
func NewDocumentAPI(client *clients.DocumentClient) *DocumentAPI {
	return &DocumentAPI{client: client}
}

// RegisterRoutes đăng ký các routes cho document
func (api *DocumentAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/policies", api.CreatePolicy)
	mux.HandleFunc("GET /api/v1/policies/latest", api.GetLatestPolicy)
}

// CreatePolicy xử lý POST /api/v1/policies
func (api *DocumentAPI) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		DocumentName       string `json:"document_name"`
		Platform           string `json:"platform"`
		IsMandatory        bool   `json:"is_mandatory"`
		EffectiveTimestamp int64  `json:"effective_timestamp"`
		ContentHTML        string `json:"content_html"`
		FileURL            string `json:"file_url"`
		CreatedBy          string `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "Invalid request body")
		return
	}

	if reqBody.DocumentName == "" || reqBody.Platform == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "document_name and platform are required")
		return
	}

	grpcReq := &pb.CreateDocumentRequest{
		DocumentName:       reqBody.DocumentName,
		Platform:           reqBody.Platform,
		IsMandatory:        reqBody.IsMandatory,
		EffectiveTimestamp: reqBody.EffectiveTimestamp,
		ContentHtml:        reqBody.ContentHTML,
		FileUrl:            reqBody.FileURL,
		CreatedBy:          reqBody.CreatedBy,
	}

	grpcResp, err := api.client.CreatePolicy(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	data := map[string]interface{}{
		"id":                  grpcResp.Document.Id,
		"document_name":       grpcResp.Document.DocumentName,
		"platform":            grpcResp.Document.Platform,
		"is_mandatory":        grpcResp.Document.IsMandatory,
		"effective_timestamp": grpcResp.Document.EffectiveTimestamp,
		"content_html":        grpcResp.Document.ContentHtml,
		"file_url":            grpcResp.Document.FileUrl,
		"created_at":          grpcResp.Document.CreatedAt,
		"created_by":          grpcResp.Document.CreatedBy,
	}

	response.Success(w, data)
}

// GetLatestPolicy xử lý GET /api/v1/policies/latest?platform=xxx&document_name=xxx
func (api *DocumentAPI) GetLatestPolicy(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	documentName := r.URL.Query().Get("document_name")

	if platform == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "platform is required")
		return
	}

	grpcReq := &pb.GetLatestPolicyRequest{
		Platform:     platform,
		DocumentName: documentName,
	}

	grpcResp, err := api.client.GetLatestPolicy(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	if grpcResp.Document == nil {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "Policy not found")
		return
	}

	data := map[string]interface{}{
		"id":                  grpcResp.Document.Id,
		"document_name":       grpcResp.Document.DocumentName,
		"platform":            grpcResp.Document.Platform,
		"is_mandatory":        grpcResp.Document.IsMandatory,
		"effective_timestamp": grpcResp.Document.EffectiveTimestamp,
		"content_html":        grpcResp.Document.ContentHtml,
		"file_url":            grpcResp.Document.FileUrl,
		"created_at":          grpcResp.Document.CreatedAt,
		"created_by":          grpcResp.Document.CreatedBy,
	}

	response.Success(w, data)
}
