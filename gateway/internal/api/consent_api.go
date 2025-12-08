package api

import (
	"encoding/json"
	"net/http"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"
	"github.com/thatlq1812/policy-system/gateway/internal/response"

	pb "github.com/thatlq1812/policy-system/shared/pkg/api/consent"
)

// ConsentAPI xử lý các HTTP endpoints liên quan đến Consent
type ConsentAPI struct {
	client *clients.ConsentClient
}

// NewConsentAPI tạo mới ConsentAPI handler
func NewConsentAPI(client *clients.ConsentClient) *ConsentAPI {
	return &ConsentAPI{client: client}
}

// RegisterRoutes đăng ký các routes cho consent
func (api *ConsentAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/consents", api.RecordConsent)
	mux.HandleFunc("POST /api/v1/consents/check", api.CheckConsent)
	mux.HandleFunc("GET /api/v1/consents/user", api.GetUserConsents)
	mux.HandleFunc("POST /api/v1/consents/pending", api.CheckPendingConsents)
	mux.HandleFunc("POST /api/v1/consents/revoke", api.RevokeConsent)
}

// RecordConsent xử lý POST /api/v1/consents
func (api *ConsentAPI) RecordConsent(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		UserID   string `json:"user_id"`
		Platform string `json:"platform"`
		Consents []struct {
			DocumentID       string `json:"document_id"`
			DocumentName     string `json:"document_name"`
			VersionTimestamp int64  `json:"version_timestamp"`
			AgreedFileURL    string `json:"agreed_file_url"`
		} `json:"consents"`
		ConsentMethod string `json:"consent_method"`
		IPAddress     string `json:"ip_address"`
		UserAgent     string `json:"user_agent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "Invalid request body")
		return
	}

	if reqBody.UserID == "" || reqBody.Platform == "" || len(reqBody.Consents) == 0 {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "user_id, platform, and consents are required")
		return
	}

	// Auto-fill IP and UserAgent if not provided
	if reqBody.IPAddress == "" {
		reqBody.IPAddress = r.RemoteAddr
	}
	if reqBody.UserAgent == "" {
		reqBody.UserAgent = r.Header.Get("User-Agent")
	}

	// Convert to proto format
	consentInputs := make([]*pb.ConsentInput, len(reqBody.Consents))
	for i, c := range reqBody.Consents {
		consentInputs[i] = &pb.ConsentInput{
			DocumentId:       c.DocumentID,
			DocumentName:     c.DocumentName,
			VersionTimestamp: c.VersionTimestamp,
			AgreedFileUrl:    c.AgreedFileURL,
		}
	}

	grpcReq := &pb.RecordConsentRequest{
		UserId:        reqBody.UserID,
		Platform:      reqBody.Platform,
		Consents:      consentInputs,
		ConsentMethod: reqBody.ConsentMethod,
		IpAddress:     reqBody.IPAddress,
		UserAgent:     reqBody.UserAgent,
	}

	grpcResp, err := api.client.RecordConsent(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	// Convert consents to response format
	consents := make([]map[string]interface{}, len(grpcResp.Consents))
	for i, c := range grpcResp.Consents {
		consents[i] = map[string]interface{}{
			"id":                c.Id,
			"user_id":           c.UserId,
			"platform":          c.Platform,
			"document_id":       c.DocumentId,
			"document_name":     c.DocumentName,
			"version_timestamp": c.VersionTimestamp,
			"agreed_at":         c.AgreedAt,
			"agreed_file_url":   c.AgreedFileUrl,
			"consent_method":    c.ConsentMethod,
			"ip_address":        c.IpAddress,
		}
	}

	data := map[string]interface{}{
		"consents":       consents,
		"total_recorded": grpcResp.TotalRecorded,
	}

	response.Success(w, data)
}

// CheckConsent xử lý POST /api/v1/consents/check
func (api *ConsentAPI) CheckConsent(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		UserID              string `json:"user_id"`
		DocumentID          string `json:"document_id"`
		MinVersionTimestamp int64  `json:"min_version_timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "Invalid request body")
		return
	}

	if reqBody.UserID == "" || reqBody.DocumentID == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "user_id and document_id are required")
		return
	}

	grpcReq := &pb.CheckConsentRequest{
		UserId:              reqBody.UserID,
		DocumentId:          reqBody.DocumentID,
		MinVersionTimestamp: reqBody.MinVersionTimestamp,
	}

	grpcResp, err := api.client.CheckConsent(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	data := map[string]interface{}{
		"has_consented": grpcResp.HasConsented,
	}

	if grpcResp.LatestConsent != nil {
		data["latest_consent"] = map[string]interface{}{
			"id":                grpcResp.LatestConsent.Id,
			"user_id":           grpcResp.LatestConsent.UserId,
			"document_id":       grpcResp.LatestConsent.DocumentId,
			"version_timestamp": grpcResp.LatestConsent.VersionTimestamp,
			"agreed_at":         grpcResp.LatestConsent.AgreedAt,
		}
	}

	response.Success(w, data)
}

// GetUserConsents xử lý GET /api/v1/consents/user?user_id=xxx&include_deleted=false
func (api *ConsentAPI) GetUserConsents(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	includeDeleted := r.URL.Query().Get("include_deleted") == "true"

	if userID == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "user_id is required")
		return
	}

	grpcReq := &pb.GetUserConsentsRequest{
		UserId:         userID,
		IncludeDeleted: includeDeleted,
	}

	grpcResp, err := api.client.GetUserConsents(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	consents := make([]map[string]interface{}, len(grpcResp.Consents))
	for i, c := range grpcResp.Consents {
		consents[i] = map[string]interface{}{
			"id":                c.Id,
			"user_id":           c.UserId,
			"platform":          c.Platform,
			"document_id":       c.DocumentId,
			"document_name":     c.DocumentName,
			"version_timestamp": c.VersionTimestamp,
			"agreed_at":         c.AgreedAt,
			"is_deleted":        c.IsDeleted,
			"deleted_at":        c.DeletedAt,
		}
	}

	data := map[string]interface{}{
		"consents": consents,
		"total":    grpcResp.Total,
	}

	response.Success(w, data)
}

// CheckPendingConsents xử lý POST /api/v1/consents/pending
func (api *ConsentAPI) CheckPendingConsents(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		UserID         string `json:"user_id"`
		Platform       string `json:"platform"`
		LatestPolicies []struct {
			DocumentID       string `json:"document_id"`
			DocumentName     string `json:"document_name"`
			VersionTimestamp int64  `json:"version_timestamp"`
			Platform         string `json:"platform"`
		} `json:"latest_policies"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "Invalid request body")
		return
	}

	if reqBody.UserID == "" || reqBody.Platform == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "user_id and platform are required")
		return
	}

	policies := make([]*pb.PendingPolicy, len(reqBody.LatestPolicies))
	for i, p := range reqBody.LatestPolicies {
		policies[i] = &pb.PendingPolicy{
			DocumentId:       p.DocumentID,
			DocumentName:     p.DocumentName,
			VersionTimestamp: p.VersionTimestamp,
			Platform:         p.Platform,
		}
	}

	grpcReq := &pb.CheckPendingConsentsRequest{
		UserId:         reqBody.UserID,
		Platform:       reqBody.Platform,
		LatestPolicies: policies,
	}

	grpcResp, err := api.client.CheckPendingConsents(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	pendingPolicies := make([]map[string]interface{}, len(grpcResp.PendingPolicies))
	for i, p := range grpcResp.PendingPolicies {
		pendingPolicies[i] = map[string]interface{}{
			"document_id":       p.DocumentId,
			"document_name":     p.DocumentName,
			"version_timestamp": p.VersionTimestamp,
			"platform":          p.Platform,
		}
	}

	data := map[string]interface{}{
		"pending_policies": pendingPolicies,
		"requires_consent": grpcResp.RequiresConsent,
	}

	response.Success(w, data)
}

// RevokeConsent xử lý POST /api/v1/consents/revoke
func (api *ConsentAPI) RevokeConsent(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		UserID           string `json:"user_id"`
		DocumentID       string `json:"document_id"`
		VersionTimestamp int64  `json:"version_timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "Invalid request body")
		return
	}

	if reqBody.UserID == "" || reqBody.DocumentID == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "user_id and document_id are required")
		return
	}

	grpcReq := &pb.RevokeConsentRequest{
		UserId:           reqBody.UserID,
		DocumentId:       reqBody.DocumentID,
		VersionTimestamp: reqBody.VersionTimestamp,
	}

	grpcResp, err := api.client.RevokeConsent(r.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		response.Error(w, statusCode, code, msg)
		return
	}

	data := map[string]interface{}{
		"success": grpcResp.Success,
		"message": grpcResp.Message,
	}

	response.Success(w, data)
}
