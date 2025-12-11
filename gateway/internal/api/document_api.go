package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thatlq1812/policy-system/gateway/internal/clients"
	"github.com/thatlq1812/policy-system/gateway/internal/middleware"

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

// CreatePolicy godoc
// @Summary      Create new policy document
// @Description  Create a new policy document with version tracking and platform targeting
// @Tags         Policy Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{document_name=string,platform=string,is_mandatory=bool,effective_timestamp=int64,content_html=string,file_url=string,created_by=string} true "Policy document details. platform must be one of: Client, Merchant, Admin"
// @Success      201  {object}  object{code=string,message=string,data=object{id=string,document_name=string,platform=string,is_mandatory=bool,effective_timestamp=int64,content_html=string,file_url=string,created_at=int64,created_by=string}}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      500  {object}  object{code=string,message=string}
// @Router       /policies [post]
func (api *DocumentAPI) CreatePolicy(c *gin.Context) {
	var reqBody struct {
		DocumentName       string `json:"document_name" binding:"required"`
		Platform           string `json:"platform" binding:"required,oneof=Client Merchant Admin"`
		IsMandatory        bool   `json:"is_mandatory"`
		EffectiveTimestamp int64  `json:"effective_timestamp"`
		ContentHTML        string `json:"content_html"`
		FileURL            string `json:"file_url"`
		CreatedBy          string `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": err.Error(),
		})
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

	grpcResp, err := api.client.CreatePolicy(c.Request.Context(), grpcReq)
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
		"message": "Policy created successfully",
		"data": gin.H{
			"id":                  grpcResp.Document.Id,
			"document_name":       grpcResp.Document.DocumentName,
			"platform":            grpcResp.Document.Platform,
			"is_mandatory":        grpcResp.Document.IsMandatory,
			"effective_timestamp": grpcResp.Document.EffectiveTimestamp,
			"content_html":        grpcResp.Document.ContentHtml,
			"file_url":            grpcResp.Document.FileUrl,
			"created_at":          grpcResp.Document.CreatedAt,
			"created_by":          grpcResp.Document.CreatedBy,
		},
	})
}

// GetLatestPolicy godoc
// @Summary      Get latest policy document
// @Description  Retrieve the latest policy document for a specific platform and optional document name
// @Tags         Policy Management
// @Produce      json
// @Param        platform      query  string  true   "Platform (Client, Merchant, Admin)"
// @Param        document_name query  string  false  "Filter by document name"
// @Success      200  {object}  object{code=string,message=string,data=object{document=object{id=string,document_name=string,platform=string,is_mandatory=bool,effective_timestamp=int64,content_html=string,file_url=string,created_at=int64}}}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /policies/latest [get]
func (api *DocumentAPI) GetLatestPolicy(c *gin.Context) {
	platform := c.Query("platform")
	documentName := c.Query("document_name")

	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "400",
			"message": "platform query parameter is required",
		})
		return
	}

	grpcReq := &pb.GetLatestPolicyRequest{
		Platform:     platform,
		DocumentName: documentName,
	}

	grpcResp, err := api.client.GetLatestPolicy(c.Request.Context(), grpcReq)
	if err != nil {
		statusCode, code, msg := middleware.GrpcErrorToHTTP(err)
		c.JSON(statusCode, gin.H{
			"code":    code,
			"message": msg,
		})
		return
	}

	if grpcResp.Document == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "404",
			"message": "Policy not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "200",
		"message": "Success",
		"data": gin.H{
			"id":                  grpcResp.Document.Id,
			"document_name":       grpcResp.Document.DocumentName,
			"platform":            grpcResp.Document.Platform,
			"is_mandatory":        grpcResp.Document.IsMandatory,
			"effective_timestamp": grpcResp.Document.EffectiveTimestamp,
			"content_html":        grpcResp.Document.ContentHtml,
			"file_url":            grpcResp.Document.FileUrl,
			"created_at":          grpcResp.Document.CreatedAt,
			"created_by":          grpcResp.Document.CreatedBy,
		},
	})
}
