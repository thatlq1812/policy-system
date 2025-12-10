# User Service - Complete Authentication & Management

Authentication and user management service using gRPC with dual token system (Access + Refresh tokens).

**Version:** 1.0.0 | **Status:** Production Ready | **Port:** 50052 (gRPC) | **Methods:** 15/15

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Database Schema](#database-schema)
4. [API Reference](#api-reference)
5. [Testing Guide](#testing-guide)
6. [Security](#security)
7. [Troubleshooting](#troubleshooting)

---

## Overview

### Features

**Authentication (Phase 1-2)**
- User Registration and Login
- Dual Token Authentication (Access + Refresh)
- Token Refresh (generate new access token)
- Logout (token revocation)
- Multi-device Support (multiple refresh tokens per user)
- Secure Password Storage (bcrypt)
- Secure Token Storage (SHA256 hashing)

**User Profile Management (Phase 3-4)**
- Get user profile by ID
- Update user profile (name, phone number)
- Change password with validation

**Admin Operations (Phase 5-6)**
- List users with pagination and filtering
- Search users by name or phone number
- Soft delete users with reason tracking
- Update user platform roles

**Session Management (Phase 7)**
- View all active sessions for a user
- Logout from all devices at once
- Revoke specific sessions by token ID

**Statistics (Phase 8)**
- Get comprehensive user statistics
- Count users by role
- Track active sessions across all users

### Security Highlights
- **Access Token:** JWT, 15 minutes, stateless
- **Refresh Token:** UUID, 30 days, stateful (stored in DB)
- **Password:** bcrypt hashed (cost factor 10)
- **Tokens:** SHA256 hashed in database
- **Revocation:** Full logout support with audit trail

### Architecture
```
Client/Gateway
    | (gRPC)
User Service
    |-- Service Layer (Business Logic)
    |-- Repository Layer (Data Access)
    +-- PostgreSQL Database
        |-- users table
        +-- refresh_tokens table
```

---

## Quick Start

### Run with Docker (Recommended)
```bash
# From project root d:/w2
docker-compose up -d user_service

# Check logs
docker-compose logs -f user_service

# Run migrations
docker-compose up user_migrate

# Verify service
grpcurl -plaintext localhost:50052 list user.UserService
```

### Run Locally (Development)
```bash
cd user

# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/user_db?sslmode=disable"
export JWT_SECRET="your-secret-key-min-32-chars-recommended-64"
export JWT_EXPIRY_HOURS="720"
export SERVER_PORT="50052"

# Run service
go run cmd/server/main.go

# Or build binary
go build -o bin/user cmd/server/main.go
./bin/user
```

### Environment Variables
| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | - | Yes |
| `JWT_SECRET` | Secret key for JWT signing | - | Yes |
| `JWT_EXPIRY_HOURS` | Deprecated (now using constants) | 720 | No |
| `SERVER_PORT` | gRPC server port | 50052 | No |

---

## Database Schema

### users table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    platform_role VARCHAR(50) NOT NULL,  -- Client | Merchant | Admin
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_users_phone_number ON users(phone_number);
```

### refresh_tokens table
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,
    revoked_reason VARCHAR(255),
    device_info TEXT,
    ip_address VARCHAR(50)
);

-- Performance indexes
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id, expires_at) 
WHERE revoked_at IS NULL;
```

**Migrations:**
- `000001_create_users_table.up.sql`
- `000002_create_refresh_tokens_table.up.sql`

---

## API Reference

### Available Methods (15 Total)

**Authentication & Token Management:**
```
user.UserService.Register       - Create new user account
user.UserService.Login          - Authenticate user
user.UserService.RefreshToken   - Generate new access token
user.UserService.Logout         - Revoke refresh token
```

**User Profile Management:**
```
user.UserService.GetUserProfile     - Get user profile by ID
user.UserService.UpdateUserProfile  - Update user information
user.UserService.ChangePassword     - Change user password
```

**Admin Operations:**
```
user.UserService.ListUsers      - List users with pagination
user.UserService.SearchUsers    - Search users by query
user.UserService.DeleteUser     - Soft delete user
user.UserService.UpdateUserRole - Change user platform role
```

**Session Management:**
```
user.UserService.GetActiveSessions  - Get all active sessions
user.UserService.LogoutAllDevices   - Logout from all devices
user.UserService.RevokeSession      - Revoke specific session
```

**Statistics:**
```
user.UserService.GetUserStats   - Get user statistics
```

---

## Phase 1-2: Authentication & Token Management

### 1. Register

**RPC:** `user.UserService/Register`  
**Purpose:** Create new user account with dual tokens

**Request:**
```protobuf
message RegisterRequest {
    string phone_number = 1;  // Required, format: 10 digits starting with 0
    string password = 2;      // Required, min 6 characters
    string name = 3;          // Required
    string platform_role = 4; // Required: "Client" or "Merchant"
}
```

**Response:**
```protobuf
message RegisterResponse {
    User user = 1;
    string access_token = 2;
    string refresh_token = 3;
    int64 access_token_expires_at = 4;   // Unix timestamp
    int64 refresh_token_expires_at = 5;  // Unix timestamp
}
```

**Example (grpcurl):**
```bash
grpcurl -plaintext -d '{
  "phone_number": "0912345678",
  "password": "SecurePass123!",
  "name": "John Doe",
  "platform_role": "Client"
}' localhost:50052 user.UserService/Register
```

**Success Response:**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "phoneNumber": "0912345678",
    "name": "John Doe",
    "platformRole": "Client",
    "createdAt": "1765333624",
    "updatedAt": "1765333624"
  },
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "cbbabedd-e5b0-4264-b5d2-fbe543136615",
  "accessTokenExpiresAt": "1765334524",
  "refreshTokenExpiresAt": "1767925624"
}
```

**Error Codes:**
- `INVALID_ARGUMENT` - Invalid phone format or password too short
- `ALREADY_EXISTS` - Phone number already registered
- `INTERNAL` - Database error

---

### 2. Login

**RPC:** `user.UserService/Login`  
**Purpose:** Authenticate user and issue new tokens

**Request:**
```protobuf
message LoginRequest {
    string phone_number = 1;  // Required
    string password = 2;      // Required
}
```

**Response:** Same as RegisterResponse (user + dual tokens)

**Example (grpcurl):**
```bash
grpcurl -plaintext -d '{
  "phone_number": "0912345678",
  "password": "SecurePass123!"
}' localhost:50052 user.UserService/Login
```

**Success Response:**
```json
{
  "user": { /* same structure as register */ },
  "accessToken": "eyJhbGci...",
  "refreshToken": "2b1c5976-b065-4579-986b-2aec2fe224df",
  "accessTokenExpiresAt": "1765334529",
  "refreshTokenExpiresAt": "1767925629"
}
```

**Error Codes:**
- `INVALID_ARGUMENT` - Missing phone or password
- `UNAUTHENTICATED` - Invalid credentials
- `INTERNAL` - Database error

**Note:** Each login creates a NEW refresh token (supports multi-device)

---

### 3. RefreshToken

**RPC:** `user.UserService/RefreshToken`  
**Purpose:** Generate new access token using refresh token

**Request:**
```protobuf
message RefreshTokenRequest {
    string refresh_token = 1;  // Required
}
```

**Response:**
```protobuf
message RefreshTokenResponse {
    string access_token = 1;
    string refresh_token = 2;          // Same as input (not rotated)
    int64 access_token_expires_at = 3;
}
```

**Example (grpcurl):**
```bash
grpcurl -plaintext -d '{
  "refresh_token": "2b1c5976-b065-4579-986b-2aec2fe224df"
}' localhost:50052 user.UserService/RefreshToken
```

**Success Response:**
```json
{
  "accessToken": "eyJhbGci...",
  "refreshToken": "2b1c5976-b065-4579-986b-2aec2fe224df",
  "accessTokenExpiresAt": "1765334588"
}
```

**Error Codes:**
- `INVALID_ARGUMENT` - Missing refresh token
- `UNAUTHENTICATED` - Token invalid, expired, or revoked
- `INTERNAL` - Database error

**Business Logic:**
1. Hash provided refresh token (SHA256)
2. Lookup in database by hash
3. Verify token is not revoked (`revoked_at IS NULL`)
4. Verify token is not expired (`expires_at > NOW()`)
5. Get user by `user_id` from token record
6. Generate new access token (15 min expiry)
7. Return new access token (refresh token unchanged)

---

### 4. Logout

**RPC:** `user.UserService/Logout`  
**Purpose:** Revoke refresh token (logout user)

**Request:**
```protobuf
message LogoutRequest {
    string refresh_token = 1;  // Required
}
```

**Response:**
```protobuf
message LogoutResponse {
    bool success = 1;
    string message = 2;
}
```

**Example (grpcurl):**
```bash
grpcurl -plaintext -d '{
  "refresh_token": "2b1c5976-b065-4579-986b-2aec2fe224df"
}' localhost:50052 user.UserService/Logout
```

**Success Response:**
```json
{
  "success": true,
  "message": "Successfully logged out"
}
```

**Error Codes:**
- `INVALID_ARGUMENT` - Missing refresh token
- `INTERNAL` - Database error

**Business Logic:**
1. Hash provided refresh token
2. Mark as revoked in database: `UPDATE refresh_tokens SET revoked_at = NOW(), revoked_reason = 'user_logout' WHERE token_hash = ?`
3. Return success

**Note:** Old access token remains valid until expiry (max 15 minutes)

---

## Phase 3-4: User Profile Management

### 5. GetUserProfile
**RPC:** `user.UserService/GetUserProfile`

### 6. UpdateUserProfile
**RPC:** `user.UserService/UpdateUserProfile`

### 7. ChangePassword
**RPC:** `user.UserService/ChangePassword`

---

## Phase 5-6: Admin Operations

### 8. ListUsers
**RPC:** `user.UserService/ListUsers`

### 9. SearchUsers
**RPC:** `user.UserService/SearchUsers`

### 10. DeleteUser
**RPC:** `user.UserService/DeleteUser`

### 11. UpdateUserRole
**RPC:** `user.UserService/UpdateUserRole`

---

## Phase 7: Session Management

### 12. GetActiveSessions
**RPC:** `user.UserService/GetActiveSessions`

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:50052 user.UserService/GetActiveSessions
```

**Response:**
```json
{
  "sessions": [
    {
      "id": "81d59c89-94c8-496c-bb86-8421133846d8",
      "deviceInfo": "login",
      "createdAt": "1765347386",
      "expiresAt": "1767939386"
    }
  ],
  "totalCount": 1
}
```

### 13. LogoutAllDevices
**RPC:** `user.UserService/LogoutAllDevices`

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:50052 user.UserService/LogoutAllDevices
```

**Response:**
```json
{
  "success": true,
  "revokedCount": 2,
  "message": "Successfully logged out from 2 device(s)"
}
```

### 14. RevokeSession
**RPC:** `user.UserService/RevokeSession`

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "token_id": "81d59c89-94c8-496c-bb86-8421133846d8"
}' localhost:50052 user.UserService/RevokeSession
```

**Response:**
```json
{
  "success": true,
  "message": "Session revoked successfully"
}
```

---

## Phase 8: Statistics

### 15. GetUserStats
**RPC:** `user.UserService/GetUserStats`

**Purpose:** Get comprehensive statistics about user accounts and sessions

**Request:**
```protobuf
message GetUserStatsRequest {}
```

**Response:**
```protobuf
message GetUserStatsResponse {
    int32 total_users = 1;
    int32 total_deleted_users = 2;
    int32 total_active_sessions = 3;
    int32 users_by_role_client = 4;
    int32 users_by_role_merchant = 5;
    int32 users_by_role_admin = 6;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{}' localhost:50052 user.UserService/GetUserStats
```

**Response:**
```json
{
  "totalUsers": 5,
  "totalDeletedUsers": 1,
  "totalActiveSessions": 2,
  "usersByRoleClient": 3,
  "usersByRoleMerchant": 2,
  "usersByRoleAdmin": 0
}
```

**Business Logic:**
- Counts active users (not deleted)
- Counts deleted users
- Counts active sessions across all users
- Breaks down users by platform role (Client/Merchant/Admin)

---

## Testing Guide

### Prerequisites
```bash
# Install grpcurl
brew install grpcurl  # macOS
# or download from https://github.com/fullstorydev/grpcurl/releases
```

### Test Scenario 1: Complete Flow

```bash
# 1. Register new user
REGISTER_RESPONSE=$(grpcurl -plaintext -d '{
  "phone_number": "0999888777",
  "password": "Test123456",
  "name": "Test User",
  "platform_role": "Client"
}' localhost:50052 user.UserService/Register)

echo "$REGISTER_RESPONSE"

# Extract refresh token (requires jq)
REFRESH_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.refreshToken')

# 2. Refresh access token
grpcurl -plaintext -d "{
  \"refresh_token\": \"$REFRESH_TOKEN\"
}" localhost:50052 user.UserService/RefreshToken

# 3. Logout
grpcurl -plaintext -d "{
  \"refresh_token\": \"$REFRESH_TOKEN\"
}" localhost:50052 user.UserService/Logout

# 4. Try refresh after logout (should fail)
grpcurl -plaintext -d "{
  \"refresh_token\": \"$REFRESH_TOKEN\"
}" localhost:50052 user.UserService/RefreshToken
# Expected: ERROR: refresh token has been revoked
```

### Test Scenario 2: Multi-Device

```bash
# Device 1: Login
DEVICE1=$(grpcurl -plaintext -d '{
  "phone_number": "0999888777",
  "password": "Test123456"
}' localhost:50052 user.UserService/Login)

TOKEN1=$(echo "$DEVICE1" | jq -r '.refreshToken')

# Device 2: Login (same user)
DEVICE2=$(grpcurl -plaintext -d '{
  "phone_number": "0999888777",
  "password": "Test123456"
}' localhost:50052 user.UserService/Login)

TOKEN2=$(echo "$DEVICE2" | jq -r '.refreshToken')

# Both tokens should work
grpcurl -plaintext -d "{\"refresh_token\": \"$TOKEN1\"}" localhost:50052 user.UserService/RefreshToken
grpcurl -plaintext -d "{\"refresh_token\": \"$TOKEN2\"}" localhost:50052 user.UserService/RefreshToken

# Logout device 1 only
grpcurl -plaintext -d "{\"refresh_token\": \"$TOKEN1\"}" localhost:50052 user.UserService/Logout

# Device 2 should still work
grpcurl -plaintext -d "{\"refresh_token\": \"$TOKEN2\"}" localhost:50052 user.UserService/RefreshToken
```

### Test Scenario 3: Database Verification

```bash
# Check active tokens for user
docker exec -it policy_postgres psql -U postgres -d user_db -c "
SELECT 
  u.phone_number,
  COUNT(rt.id) FILTER (WHERE rt.revoked_at IS NULL) as active_tokens,
  COUNT(rt.id) FILTER (WHERE rt.revoked_at IS NOT NULL) as revoked_tokens,
  COUNT(rt.id) as total_tokens
FROM users u
LEFT JOIN refresh_tokens rt ON u.id = rt.user_id
WHERE u.phone_number = '0999888777'
GROUP BY u.id, u.phone_number;
"
```

---

## Security

### Token Configuration

**Access Token (JWT):**
- Algorithm: HS256 (HMAC-SHA256)
- Expiry: 15 minutes
- Storage: Client-side (memory or secure storage)
- Type: Stateless (no DB lookup needed)

**Refresh Token (UUID):**
- Format: UUID v4 (random, 36 characters)
- Expiry: 30 days
- Storage: Server-side (hashed with SHA256)
- Type: Stateful (stored in DB, can be revoked)

### Password Security
- Hashing: bcrypt with cost factor 10
- Minimum length: 6 characters (configurable)
- Stored: Hashed only, never plain text

### Best Practices

**Client-side:**
```javascript
// Store access token in memory (for web apps)
let accessToken = response.accessToken;

// Store refresh token in httpOnly cookie (Gateway handles this)
// OR secure storage for mobile apps

// API request
fetch('/api/endpoint', {
  headers: {
    'Authorization': `Bearer ${accessToken}`
  }
});

// Handle 401 Unauthorized
if (response.status === 401) {
  // Refresh access token
  const newToken = await refreshAccessToken(refreshToken);
  // Retry request with new token
}
```

**Server-side (Gateway):**
```go
// Verify JWT on every API call
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        // Verify JWT signature
        // Check expiry
        // Extract user_id and platform_role
        // Add to context
        c.Next()
    }
}
```

### Security Checklist
- [x] Passwords hashed with bcrypt
- [x] Refresh tokens hashed with SHA256
- [x] JWT signed with secret (HS256)
- [x] Short-lived access tokens (15 min)
- [x] Revocation support (logout)
- [x] Database uses parameterized queries (prevent SQL injection)
- [x] No sensitive data in JWT
- [x] Foreign key constraints
- [x] Indexes for performance

---

## Troubleshooting

### Issue 1: Service not starting

**Symptom:**
```
Failed to connect to database: connection refused
```

**Solution:**
```bash
# Check PostgreSQL is running
docker-compose ps policy_postgres

# Check logs
docker-compose logs policy_postgres

# Restart
docker-compose restart policy_postgres

# Verify connection string in .env
DATABASE_URL=postgresql://postgres:postgres@policy_postgres:5432/user_db?sslmode=disable
```

---

### Issue 2: Method not implemented

**Symptom:**
```
ERROR: method RefreshToken not implemented
```

**Solution:**
```bash
# Rebuild service (handlers might not be compiled)
docker-compose up --build -d user_service

# Check logs for compilation errors
docker-compose logs user_service
```

---

### Issue 3: Invalid or expired refresh token

**Symptom:**
```
ERROR: Code: Unauthenticated, Message: refresh token has been revoked
```

**Possible Causes:**
1. Token was revoked (logout was called)
2. Token expired (> 30 days old)
3. Token doesn't exist in database
4. Token hash mismatch

**Debug:**
```sql
-- Check token in database
SELECT 
  id, 
  expires_at,
  revoked_at,
  revoked_reason,
  expires_at > NOW() as is_not_expired,
  revoked_at IS NULL as is_not_revoked
FROM refresh_tokens 
WHERE token_hash = encode(digest('your-token-here', 'sha256'), 'hex');
```

**Solution:**
- If revoked: User needs to login again
- If expired: User needs to login again
- If doesn't exist: Invalid token, login again

---

### Issue 4: Database migration failed

**Symptom:**
```
migration failed: relation "refresh_tokens" already exists
```

**Solution:**
```bash
# Drop table manually (CAUTION: destroys data)
docker exec -it policy_postgres psql -U postgres -d user_db -c "
DROP TABLE IF EXISTS refresh_tokens CASCADE;
"

# Re-run migration
docker-compose up user_migrate
```

---

### Issue 5: JWT verification fails in Gateway

**Symptom:**
```
invalid token signature
```

**Solution:**
```bash
# Verify JWT_SECRET is same in User Service and Gateway
echo $JWT_SECRET

# Check JWT structure
echo "your-jwt-here" | awk -F'.' '{print $2}' | base64 -d | python -m json.tool

# Verify algorithm is HS256 (not RS256)
```

---

## Documentation

### Related Documents
- [CHANGELOG.md](../docs/CHANGELOG.md) - Version history and implementation details

### API Documentation
- Proto file: `shared/pkg/api/user/user.proto`
- Generated code: `shared/pkg/api/user/*.pb.go`

### Database Migrations
- Location: `user/migrations/`
- Tool: golang-migrate
- Applied automatically by `user_migrate` container

---

## Development

### Project Structure
```
user/
|-- cmd/
|   +-- server/
|       +-- main.go              # Entry point
|-- internal/
|   |-- configs/
|   |   +-- config.go            # Configuration
|   |-- domain/
|   |   |-- user.go              # User entity
|   |   +-- refresh_token.go    # RefreshToken entity
|   |-- handler/
|   |   +-- user_handler.go     # gRPC handlers
|   |-- repository/
|   |   |-- user_repository.go          # User data access
|   |   +-- refresh_token_repository.go # Token data access
|   +-- service/
|       |-- user_service.go      # Business logic
|       +-- token_helper.go      # Token generation
|-- migrations/
|   |-- 000001_create_users_table.up.sql
|   +-- 000002_create_refresh_tokens_table.up.sql
|-- Dockerfile
|-- go.mod
+-- README.md
```

### Adding New Features

**Example: Add GetUserProfile RPC**

1. **Update Proto:**
```protobuf
// shared/pkg/api/user/user.proto
message GetUserProfileRequest {
    string user_id = 1;
}

message GetUserProfileResponse {
    User user = 1;
}

service UserService {
    // ... existing RPCs
    rpc GetUserProfile(GetUserProfileRequest) returns (GetUserProfileResponse);
}
```

2. **Generate Code:**
```bash
cd shared
./generate.sh
```

3. **Add Service Method:**
```go
// user/internal/service/user_service.go
func (s *userService) GetUserProfile(ctx context.Context, userID string) (*domain.User, error) {
    return s.repo.GetByID(ctx, userID)
}
```

4. **Add Handler:**
```go
// user/internal/handler/user_handler.go
func (h *UserHandler) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest) (*pb.GetUserProfileResponse, error) {
    user, err := h.service.GetUserProfile(ctx, req.UserId)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    return &pb.GetUserProfileResponse{User: domainToProto(user)}, nil
}
```

5. **Test:**
```bash
grpcurl -plaintext -d '{"user_id": "uuid"}' localhost:50052 user.UserService/GetUserProfile
```

---

## Support

**Issues:** [GitHub Issues](https://github.com/thatlq1812/policy-system/issues)  
**Documentation:** [docs/](../docs/)  
**Maintainer:** Policy System Team

---

**Version:** 1.0.0  
**Last Updated:** December 10, 2025  
**Status:** Production Ready - All 15 Methods Implemented
