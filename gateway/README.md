# Gateway Service

HTTP/REST API Gateway that proxies requests to gRPC microservices with JWT authentication.

## Overview

**Port:** 8080  
**Protocol:** HTTP/REST (converts to gRPC)  
**Language:** Go 1.21+  
**Framework:** Gin

## Architecture

```
Client (HTTP/REST)
    ↓
Gateway Service :8080
    ├── JWT Verification
    ├── Rate Limiting
    ├── Error Handling
    └── gRPC Clients
        ├→ User Service :50052
        ├→ Document Service :50051
        └→ Consent Service :50053
```

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)
- All backend services running (user, document, consent)

### Run with Docker
```bash
# From project root
docker-compose up -d gateway

# Check logs
docker-compose logs -f gateway

# Test health
curl http://localhost:8080/health
```

### Run Locally
```bash
cd gateway

# Install dependencies
go mod download

# Set environment variables
export JWT_SECRET="your-secret-key-min-32-chars"
export USER_SERVICE_URL="localhost:50052"
export DOCUMENT_SERVICE_URL="localhost:50051"
export CONSENT_SERVICE_URL="localhost:50053"
export SERVER_PORT="8080"

# Run service
go run cmd/server/main.go
```

## API Endpoints

### Health Check

**GET /health**

Check if gateway is running.

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2025-12-09T10:30:00Z"
}
```

---

## Authentication APIs

All auth endpoints are under `/api/auth`

### 1. Register

**POST /api/auth/register**

Register new user account.

**Request:**
```json
{
  "phone_number": "0912345678",
  "password": "SecurePass123!",
  "name": "John Doe",
  "platform_role": "Client"
}
```

**Response:** `201 Created`
```json
{
  "user": {
    "id": "user-uuid",
    "phone_number": "0912345678",
    "name": "John Doe",
    "platform_role": "Client",
    "created_at": 1733740800,
    "updated_at": 1733740800
  },
  "access_token": "eyJhbGci...",
  "refresh_token": "uuid-token",
  "access_token_expires_at": 1733741700,
  "refresh_token_expires_at": 1736332800
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "0912345678",
    "password": "SecurePass123!",
    "name": "John Doe",
    "platform_role": "Client"
  }'
```

---

### 2. Login

**POST /api/auth/login**

Authenticate user and get tokens.

**Request:**
```json
{
  "phone_number": "0912345678",
  "password": "SecurePass123!"
}
```

**Response:** `200 OK`
```json
{
  "user": { /* same as register */ },
  "access_token": "eyJhbGci...",
  "refresh_token": "uuid-token",
  "access_token_expires_at": 1733741700,
  "refresh_token_expires_at": 1736332800
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "0912345678",
    "password": "SecurePass123!"
  }'
```

---

### 3. Refresh Token

**POST /api/auth/refresh**

Get new access token using refresh token.

**Request:**
```json
{
  "refresh_token": "uuid-refresh-token"
}
```

**Response:** `200 OK`
```json
{
  "access_token": "new-jwt-token",
  "refresh_token": "same-refresh-token",
  "access_token_expires_at": 1733745000
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your-refresh-token"
  }'
```

---

### 4. Logout

**POST /api/auth/logout**

Revoke refresh token.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request:**
```json
{
  "refresh_token": "uuid-refresh-token"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Successfully logged out"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Authorization: Bearer your-access-token" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your-refresh-token"
  }'
```

---

## User Profile APIs

All profile endpoints require authentication.

**Base Path:** `/api/users`  
**Auth Required:** Yes (JWT token)

### 5. Get My Profile

**GET /api/users/me**

Get current user's profile (user_id from JWT).

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:** `200 OK`
```json
{
  "id": "user-uuid",
  "phone_number": "0912345678",
  "name": "John Doe",
  "platform_role": "Client",
  "created_at": 1733740800,
  "updated_at": 1733740800
}
```

**Example:**
```bash
curl http://localhost:8080/api/users/me \
  -H "Authorization: Bearer your-access-token"
```

---

### 6. Update My Profile

**PUT /api/users/me**

Update current user's name and/or phone number.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request:**
```json
{
  "name": "New Name",
  "phone_number": "0987654321"
}
```

**Response:** `200 OK`
```json
{
  "user": { /* updated user object */ },
  "message": "Profile updated successfully"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/users/me \
  -H "Authorization: Bearer your-access-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Name"
  }'
```

---

### 7. Change Password

**PUT /api/users/me/password**

Change current user's password.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request:**
```json
{
  "old_password": "OldPass123!",
  "new_password": "NewPass456!"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Password changed successfully. Please login again on all devices."
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/users/me/password \
  -H "Authorization: Bearer your-access-token" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "OldPass123!",
    "new_password": "NewPass456!"
  }'
```

---

## Document APIs

**Base Path:** `/api/documents`  
**Auth Required:** Yes (some endpoints)

### 8. List Documents

**GET /api/documents**

Get all published documents (no auth required for public view).

**Query Parameters:**
- `status` (optional): Filter by status (Draft, Published, Archived)

**Response:** `200 OK`
```json
{
  "documents": [
    {
      "id": "doc-uuid",
      "title": "Privacy Policy",
      "content": "Full content...",
      "version": 3,
      "status": "Published",
      "effective_date": "2025-01-01",
      "created_at": 1733740800,
      "updated_at": 1733740800
    }
  ],
  "total_count": 5
}
```

**Example:**
```bash
# Get all published documents
curl http://localhost:8080/api/documents

# Filter by status
curl http://localhost:8080/api/documents?status=Published
```

---

### 9. Get Document by Title

**GET /api/documents/title/:title**

Get latest published version of a document.

**Parameters:**
- `title`: Document title (URL encoded)

**Response:** `200 OK`
```json
{
  "id": "doc-uuid",
  "title": "Privacy Policy",
  "content": "Full content...",
  "version": 3,
  "status": "Published",
  "effective_date": "2025-01-01",
  "created_at": 1733740800,
  "updated_at": 1733740800
}
```

**Example:**
```bash
curl http://localhost:8080/api/documents/title/Privacy%20Policy
```

---

### 10. Get Document by ID

**GET /api/documents/:id**

Get specific document version by ID.

**Response:** `200 OK`
```json
{
  "id": "doc-uuid",
  "title": "Privacy Policy",
  "content": "Full content...",
  "version": 3,
  "status": "Published",
  "effective_date": "2025-01-01",
  "created_at": 1733740800,
  "updated_at": 1733740800
}
```

**Example:**
```bash
curl http://localhost:8080/api/documents/doc-uuid-here
```

---

### 11. Create Document (Admin)

**POST /api/documents**

Create new policy document (Admin only).

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Request:**
```json
{
  "title": "Cookie Policy",
  "content": "We use cookies...",
  "effective_date": "2025-06-01"
}
```

**Response:** `201 Created`
```json
{
  "id": "new-doc-uuid",
  "title": "Cookie Policy",
  "content": "We use cookies...",
  "version": 1,
  "status": "Draft",
  "effective_date": "2025-06-01",
  "owner": "admin-uuid",
  "created_at": 1733740800,
  "updated_at": 1733740800
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/documents \
  -H "Authorization: Bearer admin-access-token" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Cookie Policy",
    "content": "We use cookies...",
    "effective_date": "2025-06-01"
  }'
```

---

### 12. Update Document (Admin)

**PUT /api/documents/:id**

Update document (creates new version if published).

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Request:**
```json
{
  "title": "Privacy Policy",
  "content": "Updated content...",
  "effective_date": "2025-04-01"
}
```

**Response:** `200 OK`
```json
{
  "id": "new-version-uuid",
  "title": "Privacy Policy",
  "version": 4,
  "status": "Draft",
  // ... other fields
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/documents/doc-uuid \
  -H "Authorization: Bearer admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Updated content"
  }'
```

---

### 13. Publish Document (Admin)

**POST /api/documents/:id/publish**

Change document status from Draft to Published.

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Response:** `200 OK`
```json
{
  "id": "doc-uuid",
  "status": "Published",
  // ... other fields
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/documents/doc-uuid/publish \
  -H "Authorization: Bearer admin-token"
```

---

## Consent APIs

**Base Path:** `/api/consents`  
**Auth Required:** Yes

### 14. Create Consent

**POST /api/consents**

Record user's acceptance of a policy document.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request:**
```json
{
  "document_id": "privacy-policy-uuid",
  "status": "Accepted"
}
```

**Response:** `201 Created`
```json
{
  "id": "consent-uuid",
  "user_id": "user-uuid",
  "document_id": "privacy-policy-uuid",
  "version": 3,
  "status": "Accepted",
  "consented_at": 1733740800,
  "created_at": 1733740800
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/consents \
  -H "Authorization: Bearer your-access-token" \
  -H "Content-Type: application/json" \
  -d '{
    "document_id": "doc-uuid",
    "status": "Accepted"
  }'
```

---

### 15. Get My Consents

**GET /api/consents/me**

Get all consents for current user.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:** `200 OK`
```json
{
  "consents": [
    {
      "id": "consent-uuid",
      "user_id": "user-uuid",
      "document_id": "doc-uuid",
      "version": 3,
      "status": "Accepted",
      "consented_at": 1733740800
    }
  ],
  "total_count": 2
}
```

**Example:**
```bash
curl http://localhost:8080/api/consents/me \
  -H "Authorization: Bearer your-access-token"
```

---

### 16. Check Consent Status

**GET /api/consents/check**

Check if user has valid consent for a document.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `document_title`: Title of the document

**Response:** `200 OK`
```json
{
  "has_consent": true,
  "consent": {
    "id": "consent-uuid",
    "version": 3,
    "status": "Accepted",
    "consented_at": 1733740800
  },
  "is_latest_version": true
}
```

**Example:**
```bash
curl "http://localhost:8080/api/consents/check?document_title=Privacy%20Policy" \
  -H "Authorization: Bearer your-access-token"
```

---

### 17. Revoke Consent

**PUT /api/consents/:id/revoke**

Revoke a consent (GDPR compliance).

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request:**
```json
{
  "reason": "User requested data deletion"
}
```

**Response:** `200 OK`
```json
{
  "id": "consent-uuid",
  "status": "Revoked",
  "revoked_at": 1733745000,
  "revoked_reason": "User requested data deletion"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/consents/consent-uuid/revoke \
  -H "Authorization: Bearer your-access-token" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "User requested data deletion"
  }'
```

---

## Authentication Flow

### JWT Token Structure

**Access Token Claims:**
```json
{
  "user_id": "uuid",
  "platform_role": "Client",
  "type": "access",
  "exp": 1733741700,
  "iat": 1733740800
}
```

**Token Verification:**
```go
// Gateway middleware
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract token from Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Missing authorization header"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        // Parse and verify JWT
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(jwtSecret), nil
        })

        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        // Extract claims
        claims := token.Claims.(jwt.MapClaims)
        c.Set("user_id", claims["user_id"].(string))
        c.Set("platform_role", claims["platform_role"].(string))

        c.Next()
    }
}
```

### Role-Based Access Control

**Admin-Only Endpoints:**
```go
func RequireAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("platform_role")
        if role != "Admin" {
            c.JSON(403, gin.H{"error": "Admin access required"})
            c.Abort()
            return
        }
        c.Next()
    }
}

// Apply to routes
adminRoutes := router.Group("/api/admin")
adminRoutes.Use(AuthMiddleware(), RequireAdmin())
{
    adminRoutes.POST("/documents", createDocumentHandler)
    adminRoutes.PUT("/documents/:id", updateDocumentHandler)
}
```

## Error Handling

### Standard Error Response

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": "Additional context"
}
```

### HTTP Status Codes

- `200 OK`: Success
- `201 Created`: Resource created
- `400 Bad Request`: Invalid input
- `401 Unauthorized`: Missing or invalid token
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Duplicate or conflict
- `500 Internal Server Error`: Server error

### gRPC to HTTP Status Mapping

```go
grpc.Code                → HTTP Status
-------------------------------------------------
OK                       → 200 OK
InvalidArgument          → 400 Bad Request
Unauthenticated          → 401 Unauthorized
PermissionDenied         → 403 Forbidden
NotFound                 → 404 Not Found
AlreadyExists            → 409 Conflict
Internal                 → 500 Internal Server Error
```

## Middleware Chain

```
Request
  ↓
[CORS Middleware] ← Allow cross-origin requests
  ↓
[Logger Middleware] ← Log request/response
  ↓
[Recovery Middleware] ← Recover from panics
  ↓
[Rate Limit Middleware] ← Limit requests
  ↓
[Auth Middleware] ← Verify JWT (if required)
  ↓
[Role Middleware] ← Check permissions (if required)
  ↓
Handler
  ↓
Response
```

## Configuration

### Environment Variables

```bash
# Server
SERVER_PORT="8080"

# JWT
JWT_SECRET="your-secret-key-minimum-32-characters-long"

# Backend Services (docker network)
USER_SERVICE_URL="user_service:50052"
DOCUMENT_SERVICE_URL="document_service:50051"
CONSENT_SERVICE_URL="consent_service:50053"

# CORS (optional)
ALLOWED_ORIGINS="http://localhost:3000,https://yourdomain.com"

# Rate Limiting (optional)
RATE_LIMIT_REQUESTS_PER_SECOND="10"
RATE_LIMIT_BURST="20"
```

## Testing

### Health Check
```bash
curl http://localhost:8080/health
```

### Complete User Flow
```bash
# 1. Register
REGISTER=$(curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "0999888777",
    "password": "Test123456",
    "name": "Test User",
    "platform_role": "Client"
  }')

echo $REGISTER | jq

ACCESS_TOKEN=$(echo $REGISTER | jq -r '.access_token')
REFRESH_TOKEN=$(echo $REGISTER | jq -r '.refresh_token')

# 2. Get profile
curl http://localhost:8080/api/users/me \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq

# 3. Get documents
curl http://localhost:8080/api/documents | jq

# 4. Accept privacy policy
DOC_ID=$(curl -s http://localhost:8080/api/documents/title/Privacy%20Policy | jq -r '.id')

curl -X POST http://localhost:8080/api/consents \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"document_id\": \"$DOC_ID\", \"status\": \"Accepted\"}" | jq

# 5. Check consent
curl "http://localhost:8080/api/consents/check?document_title=Privacy%20Policy" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq
```

## Performance Optimization

### Connection Pooling

Gateway maintains persistent gRPC connections to backend services:

```go
// Initialize once at startup
userConn, _ := grpc.Dial(userServiceURL, grpc.WithInsecure())
documentConn, _ := grpc.Dial(documentServiceURL, grpc.WithInsecure())
consentConn, _ := grpc.Dial(consentServiceURL, grpc.WithInsecure())

// Reuse connections
userClient := userpb.NewUserServiceClient(userConn)
documentClient := documentpb.NewDocumentServiceClient(documentConn)
consentClient := consentpb.NewConsentServiceClient(consentConn)
```

### Caching Strategy (Future)

- Cache public documents (5 min TTL)
- Cache consent check results (1 min TTL)
- Use Redis for distributed caching

## Troubleshooting

### Gateway can't connect to services

```bash
# Check backend services are running
docker-compose ps

# Check network connectivity
docker exec gateway ping document_service
docker exec gateway ping user_service
docker exec gateway ping consent_service

# Verify service URLs in gateway config
docker-compose logs gateway | grep "service_url"
```

### JWT verification fails

```bash
# Check JWT_SECRET matches User Service
# Both must use same secret key

# Decode token to check claims
# Use https://jwt.io/ or:
echo $ACCESS_TOKEN | cut -d'.' -f2 | base64 -d | jq
```

### CORS errors

```bash
# Update ALLOWED_ORIGINS in gateway config
# Or add CORS middleware with wildcard (dev only):
router.Use(cors.Default())
```

## Security Considerations

1. **HTTPS Only:** Always use HTTPS in production
2. **JWT Secret:** Use strong, random secret (32+ chars)
3. **Rate Limiting:** Protect against DDoS
4. **Input Validation:** Validate all request bodies
5. **Error Messages:** Don't expose sensitive info in errors
6. **CORS:** Restrict allowed origins in production

## Related Services

- **User Service:** Authentication backend (port 50052)
- **Document Service:** Policy documents backend (port 50051)
- **Consent Service:** Consent tracking backend (port 50053)

## Support

For issues or questions:
1. Check logs: `docker-compose logs -f gateway`
2. Test backend services: `grpcurl -plaintext localhost:5005X list`
3. Verify JWT: https://jwt.io/
4. Check network: `docker network inspect policy-system_default`
