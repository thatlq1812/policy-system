package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/thatlq1812/policy-system/gateway/internal/response"
)

// AuthMiddleware kiểm tra JWT token trong Authorization header
// TODO: Implement JWT validation logic khi có JWT library
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lấy token từ header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "Missing authorization header")
			return
		}

		// Format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "Invalid authorization header format")
			return
		}

		token := parts[1]
		if token == "" {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "Missing token")
			return
		}

		// TODO: Validate JWT token here
		// - Parse token
		// - Verify signature
		// - Check expiration
		// - Extract user_id

		// Tạm thời skip validation (development mode)
		// Trong production, nên validate token và extract user_id
		// Sau đó lưu user_id vào context để handler sử dụng:
		// ctx := context.WithValue(r.Context(), "user_id", userID)
		// r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// OptionalAuth middleware cho endpoints không bắt buộc login
func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				// TODO: Validate token và add user_id vào context nếu valid
				_ = token
			}
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserIDFromContext lấy user_id từ context (sau khi auth)
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value("user_id").(int64)
	return userID, ok
}
