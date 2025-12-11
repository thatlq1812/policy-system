package response

import (
	"encoding/json"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	CodeConflict      = "409"
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

// ErrorFromGRPC maps gRPC error to HTTP response
func ErrorFromGRPC(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		Error(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	statusCode, code := mapGRPCCodeToHTTP(st.Code())
	message := st.Message()

	// Make error messages more user-friendly
	if strings.Contains(message, "already exists") {
		statusCode = http.StatusConflict
		code = CodeConflict
	}

	Error(w, statusCode, code, message)
}

// mapGRPCCodeToHTTP converts gRPC status code to HTTP status code
func mapGRPCCodeToHTTP(grpcCode codes.Code) (int, string) {
	switch grpcCode {
	case codes.OK:
		return http.StatusOK, CodeSuccess
	case codes.InvalidArgument:
		return http.StatusBadRequest, CodeBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized, CodeUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden, "403"
	case codes.NotFound:
		return http.StatusNotFound, CodeNotFound
	case codes.AlreadyExists:
		return http.StatusConflict, CodeConflict
	case codes.Aborted:
		return http.StatusConflict, CodeConflict
	case codes.Internal:
		return http.StatusInternalServerError, CodeInternalError
	case codes.Unavailable:
		return http.StatusServiceUnavailable, "503"
	default:
		return http.StatusInternalServerError, CodeInternalError
	}
}
