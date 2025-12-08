package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ListResponse struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Data    ListData `json:"data"`
}

type ListData struct {
	Items   interface{} `json:"items"`
	Total   int64       `json:"total"`
	Page    int32       `json:"page"`
	Size    int32       `json:"size"`
	HasMore bool        `json:"has_more"`
}

// Success codes
const (
	CodeSuccess = "000"
)

// Error codes
const (
	CodeBadRequest    = "400"
	CodeUnauthorized  = "401"
	CodeNotFound      = "404"
	CodeInternalError = "500"
)

func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

func SuccessList(w http.ResponseWriter, items interface{}, total int64, page, size int32) {
	hasMore := int64(page*size) < total
	JSON(w, http.StatusOK, ListResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data: ListData{
			Items:   items,
			Total:   total,
			Page:    page,
			Size:    size,
			HasMore: hasMore,
		},
	})
}

func Error(w http.ResponseWriter, statusCode int, code, message string) {
	JSON(w, statusCode, Response{
		Code:    code,
		Message: message,
	})
}

func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
