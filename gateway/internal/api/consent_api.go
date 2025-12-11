package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"

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

// RecordConsent godoc
// @Summary      Record user consent
// @Description  Record user consent for one or more policy documents with audit trail
// @Tags         Consent Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{user_id=string,platform=string,consents=[]object,consent_method=string,ip_address=string,user_agent=string} true "Consent recording request"
// @Success      201  {object}  object{code=string,message=string,data=object{consents=[]object,recorded_count=int32}}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /consents [post]
func (api *ConsentAPI) RecordConsent(c *gin.Context) {
	var reqBody struct {
		UserID   string `json:"user_id" binding:"required"`
		Platform string `json:"platform" binding:"required,oneof=Client Merchant Admin"`
		Consents []struct {
			DocumentID       string `json:"document_id" binding:"required"`
			DocumentName     string `json:"document_name" binding:"required"`
			VersionTimestamp int64  `json:"version_timestamp" binding:"required"`
			AgreedFileURL    string `json:"agreed_file_url"`
		} `json:"consents" binding:"required,min=1"`
		ConsentMethod string `json:"consent_method"`
		IPAddress     string `json:"ip_address"`
		UserAgent     string `json:"user_agent"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	// Auto-fill IP and UserAgent if not provided
	if reqBody.IPAddress == "" {
		reqBody.IPAddress = c.ClientIP()
	}
	if reqBody.UserAgent == "" {
		reqBody.UserAgent = c.GetHeader("User-Agent")
	}

	// Convert to proto format
	consentInputs := make([]*pb.ConsentInput, len(reqBody.Consents))
	for i, consent := range reqBody.Consents {
		consentInputs[i] = &pb.ConsentInput{
			DocumentId:       consent.DocumentID,
			DocumentName:     consent.DocumentName,
			VersionTimestamp: consent.VersionTimestamp,
			AgreedFileUrl:    consent.AgreedFileURL,
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

	grpcResp, err := api.client.RecordConsent(c.Request.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	// Convert consents to response format
	consents := make([]gin.H, len(grpcResp.Consents))
	for i, consent := range grpcResp.Consents {
		consents[i] = gin.H{
			"id":                consent.Id,
			"user_id":           consent.UserId,
			"platform":          consent.Platform,
			"document_id":       consent.DocumentId,
			"document_name":     consent.DocumentName,
			"version_timestamp": consent.VersionTimestamp,
			"agreed_at":         consent.AgreedAt,
			"agreed_file_url":   consent.AgreedFileUrl,
			"consent_method":    consent.ConsentMethod,
			"ip_address":        consent.IpAddress,
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    "201",
		"message": "Consent recorded successfully",
		"data": gin.H{
			"consents":       consents,
			"total_recorded": grpcResp.TotalRecorded,
		},
	})
}

// CheckConsent godoc
// @Summary      Check user consent status
// @Description  Check if a user has consented to a specific document and optionally verify minimum version
// @Tags         Consent Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{user_id=string,document_id=string,min_version_timestamp=int64} true "Consent check request"
// @Success      200  {object}  object{code=string,message=string,data=object{has_consented=bool,latest_consent=object}}
// @Failure      400  {object}  object{code=string,message=string}
// @Router       /consents/check [post]
func (api *ConsentAPI) CheckConsent(c *gin.Context) {
	var reqBody struct {
		UserID              string `json:"user_id" binding:"required"`
		DocumentID          string `json:"document_id" binding:"required"`
		MinVersionTimestamp int64  `json:"min_version_timestamp"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	grpcReq := &pb.CheckConsentRequest{
		UserId:              reqBody.UserID,
		DocumentId:          reqBody.DocumentID,
		MinVersionTimestamp: reqBody.MinVersionTimestamp,
	}

	grpcResp, err := api.client.CheckConsent(c.Request.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	data := gin.H{
		"has_consented": grpcResp.HasConsented,
	}

	if grpcResp.LatestConsent != nil {
		data["latest_consent"] = gin.H{
			"id":                grpcResp.LatestConsent.Id,
			"user_id":           grpcResp.LatestConsent.UserId,
			"document_id":       grpcResp.LatestConsent.DocumentId,
			"version_timestamp": grpcResp.LatestConsent.VersionTimestamp,
			"agreed_at":         grpcResp.LatestConsent.AgreedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Success",
		"data":    data,
	})
}

// GetUserConsents godoc
// @Summary      Get user's consent history
// @Description  Retrieve all consent records for a specific user with optional filtering
// @Tags         Consent Management
// @Produce      json
// @Security     BearerAuth
// @Param        user_id         query  string  true   "User ID"
// @Param        include_deleted query  bool    false  "Include revoked consents (default: false)"
// @Success      200  {object}  object{code=string,message=string,data=object{consents=[]object,total_count=int32}}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Router       /consents/user [get]
func (api *ConsentAPI) GetUserConsents(c *gin.Context) {
	userID := c.Query("user_id")
	includeDeleted := c.Query("include_deleted") == "true"

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": "user_id query parameter is required",
		})
		return
	}

	grpcReq := &pb.GetUserConsentsRequest{
		UserId:         userID,
		IncludeDeleted: includeDeleted,
	}

	grpcResp, err := api.client.GetUserConsents(c.Request.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	consents := make([]gin.H, len(grpcResp.Consents))
	for i, consent := range grpcResp.Consents {
		consents[i] = gin.H{
			"id":                consent.Id,
			"user_id":           consent.UserId,
			"platform":          consent.Platform,
			"document_id":       consent.DocumentId,
			"document_name":     consent.DocumentName,
			"version_timestamp": consent.VersionTimestamp,
			"agreed_at":         consent.AgreedAt,
			"is_deleted":        consent.IsDeleted,
			"deleted_at":        consent.DeletedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Success",
		"data": gin.H{
			"consents": consents,
			"total":    grpcResp.Total,
		},
	})
}

// CheckPendingConsents godoc
// @Summary      Check pending consents
// @Description  Check if user has any mandatory policies that haven't been consented to yet
// @Tags         Consent Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{user_id=string,platform=string,latest_policies=[]object} true "Pending consent check request"
// @Success      200  {object}  object{code=string,message=string,data=object{has_pending=bool,pending_documents=[]object,total_pending=int32}}
// @Failure      400  {object}  object{code=string,message=string}
// @Router       /consents/pending [post]
func (api *ConsentAPI) CheckPendingConsents(c *gin.Context) {
	var reqBody struct {
		UserID         string `json:"user_id" binding:"required"`
		Platform       string `json:"platform" binding:"required,oneof=Client Merchant Admin"`
		LatestPolicies []struct {
			DocumentID       string `json:"document_id" binding:"required"`
			DocumentName     string `json:"document_name" binding:"required"`
			VersionTimestamp int64  `json:"version_timestamp" binding:"required"`
			Platform         string `json:"platform" binding:"required,oneof=Client Merchant Admin"`
		} `json:"latest_policies" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
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

	grpcResp, err := api.client.CheckPendingConsents(c.Request.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	pendingPolicies := make([]gin.H, len(grpcResp.PendingPolicies))
	for i, p := range grpcResp.PendingPolicies {
		pendingPolicies[i] = gin.H{
			"document_id":       p.DocumentId,
			"document_name":     p.DocumentName,
			"version_timestamp": p.VersionTimestamp,
			"platform":          p.Platform,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Success",
		"data": gin.H{
			"pending_policies": pendingPolicies,
			"requires_consent": grpcResp.RequiresConsent,
		},
	})
}

// RevokeConsent godoc
// @Summary      Revoke user consent
// @Description  Revoke a previously given consent (GDPR compliance)
// @Tags         Consent Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{user_id=string,document_id=string,version_timestamp=int64} true "Consent revocation request"
// @Success      200  {object}  object{code=string,message=string}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /consents/revoke [post]
func (api *ConsentAPI) RevokeConsent(c *gin.Context) {
	var reqBody struct {
		UserID           string `json:"user_id" binding:"required"`
		DocumentID       string `json:"document_id" binding:"required"`
		VersionTimestamp int64  `json:"version_timestamp"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
		return
	}

	grpcReq := &pb.RevokeConsentRequest{
		UserId:           reqBody.UserID,
		DocumentId:       reqBody.DocumentID,
		VersionTimestamp: reqBody.VersionTimestamp,
	}

	grpcResp, err := api.client.RevokeConsent(c.Request.Context(), grpcReq)
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
		"message": grpcResp.Message,
		"data": gin.H{
			"success": grpcResp.Success,
		},
	})
}
