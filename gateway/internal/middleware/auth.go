package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims định nghĩa cấu trúc data trong JWT token
type Claims struct {
	UserID       string `json:"user_id"`
	PhoneNumber  string `json:"phone_number"`
	PlatformRole string `json:"platform_role"`
	jwt.RegisteredClaims
}

// AuthMiddleware tạo middleware để validate JWT token (Gin version)
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bước 1: Lấy Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "401",
				"message": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// Bước 2: Check format "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "401",
				"message": "Invalid authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "401",
				"message": "Missing token",
			})
			c.Abort()
			return
		}

		// Bước 3: Parse và validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Check signing method (phải là HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Return secret key để verify signature
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "401",
				"message": fmt.Sprintf("Invalid token: %v", err),
			})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "401",
				"message": "Token is not valid",
			})
			c.Abort()
			return
		}

		// Bước 4: Extract claims
		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "401",
				"message": "Invalid token claims",
			})
			c.Abort()
			return
		}

		// Bước 5: Lưu claims vào Gin context
		c.Set("user_id", claims.UserID)
		c.Set("phone_number", claims.PhoneNumber)
		c.Set("platform_role", claims.PlatformRole)

		// Bước 6: Gọi handler tiếp theo
		c.Next()
	}
}

// AdminOnly middleware kiểm tra user có phải Admin không
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("platform_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "401",
				"message": "User info not found. Did you apply AuthMiddleware first?",
			})
			c.Abort()
			return
		}

		if role != "Admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    "403",
				"message": "Admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions để lấy data từ Gin context

// GetUserID lấy user_id từ context (sau khi auth)
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetPhoneNumber lấy phone_number từ context
func GetPhoneNumber(c *gin.Context) (string, bool) {
	phone, exists := c.Get("phone_number")
	if !exists {
		return "", false
	}
	p, ok := phone.(string)
	return p, ok
}

// GetPlatformRole lấy platform_role từ context
func GetPlatformRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("platform_role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}
