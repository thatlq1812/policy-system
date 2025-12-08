package middleware

import (
	"net/http"

	"github.com/thatlq1812/policy-system/gateway/internal/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GrpcErrorToHTTP(err error) (int, string, string) {
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, response.CodeInternalError, err.Error()
	}

	switch st.Code() {
	case codes.OK:
		return http.StatusOK, response.CodeSuccess, "success"
	case codes.InvalidArgument:
		return http.StatusBadRequest, response.CodeBadRequest, st.Message()
	case codes.Unauthenticated:
		return http.StatusUnauthorized, response.CodeUnauthorized, st.Message()
	case codes.NotFound:
		return http.StatusNotFound, response.CodeNotFound, st.Message()
	case codes.AlreadyExists:
		return http.StatusConflict, "409", st.Message()
	case codes.PermissionDenied:
		return http.StatusForbidden, "403", st.Message()
	default:
		return http.StatusInternalServerError, response.CodeInternalError, st.Message()
	}
}
