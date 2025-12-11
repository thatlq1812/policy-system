package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"

	consentpb "github.com/thatlq1812/policy-system/shared/pkg/api/consent"
)

// AdminAPI xử lý các HTTP endpoints dành cho Admin
// Giải thích: Tách riêng admin endpoints để dễ apply admin middleware
type AdminAPI struct {
	consentClient *clients.ConsentClient
}

// NewAdminAPI tạo mới AdminAPI handler
func NewAdminAPI(consentClient *clients.ConsentClient) *AdminAPI {
	return &AdminAPI{
		consentClient: consentClient,
	}
}

// GetConsentStats godoc
// @Summary      Get consent statistics (Admin only)
// @Description  Retrieve comprehensive consent statistics including totals, breakdowns by document, platform, and method. Requires Admin role.
// @Tags         Admin - Consent Management
// @Produce      json
// @Security     BearerAuth
// @Param        platform  query  string  false  "Filter by platform (Client, Merchant, Admin)"
// @Success      200  {object}  object{code=string,message=string,data=object{total_consents=int64,active_consents=int64,revoked_consents=int64,consents_by_document=map[string]int64,consents_by_platform=map[string]int64,consents_by_method=map[string]int64}}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      403  {object}  object{code=string,message=string}
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /admin/stats/consents [get]
func (api *AdminAPI) GetConsentStats(c *gin.Context) {
	// Optional filter by platform
	platform := c.Query("platform") // e.g., "Client" or "Merchant"

	log.Printf("[ADMIN] Fetching consent stats (platform=%s)", platform)

	// Forward to Consent Service
	resp, err := api.consentClient.GetConsentStats(c.Request.Context(), &consentpb.GetConsentStatsRequest{
		Platform: platform,
	})

	if err != nil {
		log.Printf("[ADMIN] Failed to get consent stats: %v", err)
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	// Format response
	successResponse(c, http.StatusOK, "Consent statistics retrieved successfully", gin.H{
		"total_consents":       resp.TotalConsents,
		"active_consents":      resp.ActiveConsents,
		"revoked_consents":     resp.RevokedConsents,
		"consents_by_document": resp.ConsentsByDocument,
		"consents_by_platform": resp.ConsentsByPlatform,
		"consents_by_method":   resp.ConsentsByMethod,
	})
}
