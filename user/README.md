# User Service

Authentication and user management service using gRPC with dual token system (Access + Refresh tokens).

## Overview

**Port:** 50052  
**Protocol:** gRPC  
**Database:** PostgreSQL (user_db)  
**Language:** Go 1.21+

## Architecture

```
Client/Gateway
    ↓
User Service (gRPC)
    ↓
PostgreSQL
    ├── users table
    └── refresh_tokens table
```

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 15+

### Run with Docker
```bash
# From project root
docker-compose up -d user_service

# Check logs
docker-compose logs -f user_service

# Run migrations
docker-compose up user_migrate
```

### Run Locally
```bash
cd user

# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/user_db?sslmode=disable"
export JWT_SECRET="your-secret-key-min-32-chars"
export JWT_EXPIRY_HOURS="720"
export SERVER_PORT="50052"

# Run service
go run cmd/server/main.go
```

## Database Schema

### users
```sql
id              UUID PRIMARY KEY
phone_number    VARCHAR(20) UNIQUE NOT NULL
password_hash   VARCHAR(255) NOT NULL
name            VARCHAR(255) NOT NULL
platform_role   VARCHAR(50) NOT NULL  -- Client | Merchant | Admin
created_at      TIMESTAMP
updated_at      TIMESTAMP
is_deleted      BOOLEAN DEFAULT FALSE
```

### refresh_tokens
```sql
id              UUID PRIMARY KEY
user_id         UUID REFERENCES users(id)
token_hash      VARCHAR(255) UNIQUE NOT NULL
expires_at      TIMESTAMP NOT NULL
created_at      TIMESTAMP
revoked_at      TIMESTAMP
revoked_reason  VARCHAR(255)
device_info     TEXT
ip_address      VARCHAR(50)
```

## API Methods

### 1. Register
**RPC:** `user.UserService/Register`

**Purpose:** Create new user account with dual tokens

**Request:**
```json
{
  "phone_number": "0912345678",
  "password": "SecurePass123!",
  "name": "John Doe",
  "platform_role": "Client"
}
```

**Response:**
```json
{
  "user": {
    "id": "uuid",
    "phone_number": "0912345678",
    "name": "John Doe",
    "platform_role": "Client",
    "created_at": "1733740800",
    "updated_at": "1733740800"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "uuid-format-token",
  "access_token_expires_at": "1733741700",
  "refresh_token_expires_at": "1736332800"
}
```

**Business Logic:**
1. Validate phone number format (10 digits, starts with 0)
2. Check phone uniqueness
3. Hash password (bcrypt)
4. Create user in database
5. Generate access token (JWT, 15 min expiry)
6. Generate refresh token (UUID, 30 days expiry)
7. Hash and store refresh token in database
8. Return both tokens to client

**Errors:**
- `INVALID_ARGUMENT`: Invalid phone format or password requirements
- `ALREADY_EXISTS`: Phone number already registered
- `INTERNAL`: Database or system error

**Example:**
```bash
grpcurl -plaintext -d '{
  "phone_number": "0912345678",
  "password": "SecurePass123!",
  "name": "John Doe",
  "platform_role": "Client"
}' localhost:50052 user.UserService/Register
```

---

### 2. Login
**RPC:** `user.UserService/Login`

**Purpose:** Authenticate user and issue new tokens

**Request:**
```json
{
  "phone_number": "0912345678",
  "password": "SecurePass123!"
}
```

**Response:**
```json
{
  "user": { /* same as register */ },
  "access_token": "eyJhbGci...",
  "refresh_token": "uuid...",
  "access_token_expires_at": "timestamp",
  "refresh_token_expires_at": "timestamp"
}
```

**Business Logic:**
1. Find user by phone number
2. Verify password (bcrypt compare)
3. Generate new access token
4. Generate new refresh token
5. Store refresh token in database
6. Return user + tokens

**Errors:**
- `UNAUTHENTICATED`: Invalid phone or password
- `INTERNAL`: Database error

**Example:**
```bash
grpcurl -plaintext -d '{
  "phone_number": "0912345678",
  "password": "SecurePass123!"
}' localhost:50052 user.UserService/Login
```

---

### 3. RefreshToken
**RPC:** `user.UserService/RefreshToken`

**Purpose:** Get new access token using valid refresh token (when access token expires)

**Request:**
```json
{
  "refresh_token": "uuid-format-token"
}
```

**Response:**
```json
{
  "access_token": "new-jwt-token",
  "refresh_token": "same-refresh-token",
  "access_token_expires_at": "timestamp",
  "refresh_token_expires_at": "0"
}
```

**Business Logic:**
1. Hash the provided refresh token
2. Look up token in database by hash
3. Verify token is not revoked
4. Verify token is not expired
5. Get user details
6. Generate new access token
7. Return new access token (same refresh token)

**Errors:**
- `UNAUTHENTICATED`: Invalid, expired, or revoked refresh token
- `INTERNAL`: Database error

**When to Call:**
- When access token expires (15 minutes)
- Before making API call if token expiry is near
- On 401 Unauthorized response from gateway

**Example:**
```bash
grpcurl -plaintext -d '{
  "refresh_token": "your-refresh-token-here"
}' localhost:50052 user.UserService/RefreshToken
```

---

### 4. Logout
**RPC:** `user.UserService/Logout`

**Purpose:** Revoke refresh token (logout from current device)

**Request:**
```json
{
  "refresh_token": "uuid-format-token"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Successfully logged out"
}
```

**Business Logic:**
1. Hash the refresh token
2. Mark token as revoked in database
3. Set revoked_at = NOW(), revoked_reason = "user_logout"
4. Access token will expire naturally (max 15 min)

**Note:** After logout, the access token remains valid until expiry. For immediate invalidation, implement token blacklist in gateway (future enhancement).

**Example:**
```bash
grpcurl -plaintext -d '{
  "refresh_token": "your-refresh-token-here"
}' localhost:50052 user.UserService/Logout
```

---

### 5. GetUserProfile
**RPC:** `user.UserService/GetUserProfile`

**Purpose:** Get current user profile (user_id from JWT in gateway)

**Request:**
```json
{
  "user_id": "uuid-from-jwt"
}
```

**Response:**
```json
{
  "user": {
    "id": "uuid",
    "phone_number": "0912345678",
    "name": "John Doe",
    "platform_role": "Client",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
}
```

**Gateway Integration:**
```
1. Client sends: Authorization: Bearer <access_token>
2. Gateway decodes JWT → extracts user_id
3. Gateway calls UserService.GetUserProfile(user_id)
4. Return profile to client
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid-here"
}' localhost:50052 user.UserService/GetUserProfile
```

---

### 6. UpdateUserProfile
**RPC:** `user.UserService/UpdateUserProfile`

**Purpose:** Update user name and/or phone number

**Request:**
```json
{
  "user_id": "uuid",
  "name": "New Name",
  "phone_number": "0987654321"
}
```

**Response:**
```json
{
  "user": { /* updated user */ },
  "message": "Profile updated successfully"
}
```

**Business Logic:**
1. Get current user
2. Validate new phone format (if provided)
3. Check phone uniqueness (if changed)
4. Update database
5. Return updated user

**Validation Rules:**
- At least one field (name or phone_number) must be provided
- Phone must be 10 digits, start with 0
- Phone must be unique across all users

**Errors:**
- `INVALID_ARGUMENT`: Invalid format or phone already in use
- `NOT_FOUND`: User not found

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid",
  "name": "New Name"
}' localhost:50052 user.UserService/UpdateUserProfile
```

---

### 7. ChangePassword
**RPC:** `user.UserService/ChangePassword`

**Purpose:** Change user password with old password verification

**Request:**
```json
{
  "user_id": "uuid",
  "old_password": "OldPass123!",
  "new_password": "NewPass456!"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Password changed successfully. Please login again on all devices."
}
```

**Business Logic:**
1. Get user from database
2. Verify old password (bcrypt)
3. Validate new password strength
4. Hash new password
5. Update password in database
6. **Revoke all refresh tokens** (force re-login on all devices)

**Password Requirements:**
- Minimum 8 characters
- Maximum 72 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 digit

**Security Note:** All refresh tokens are revoked to prevent unauthorized access with old sessions.

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid",
  "old_password": "OldPass123!",
  "new_password": "NewPass456!"
}' localhost:50052 user.UserService/ChangePassword
```

---

### 8. ListUsers (Admin Only)
**RPC:** `user.UserService/ListUsers`

**Purpose:** Get paginated list of all users (admin dashboard)

**Request:**
```json
{
  "page": 1,
  "page_size": 20,
  "platform_role": "Client",
  "include_deleted": false
}
```

**Response:**
```json
{
  "users": [
    { /* user object */ },
    { /* user object */ }
  ],
  "total_count": 150,
  "page": 1,
  "page_size": 20,
  "total_pages": 8
}
```

**Query Parameters:**
- `page`: Page number (1-indexed, default 1)
- `page_size`: Items per page (default 20, max 100)
- `platform_role`: Filter by role (optional)
- `include_deleted`: Show soft-deleted users (default false)

**Business Logic:**
1. Build WHERE clause based on filters
2. Count total matching users
3. Calculate pagination (offset, limit)
4. Query users with ORDER BY created_at DESC
5. Return paginated result

**Gateway Protection:**
```go
// Only Admin role can access
if userRole != "Admin" {
    return 403 Forbidden
}
```

**Example:**
```bash
# Get first page
grpcurl -plaintext -d '{
  "page": 1,
  "page_size": 10
}' localhost:50052 user.UserService/ListUsers

# Filter by role
grpcurl -plaintext -d '{
  "page": 1,
  "page_size": 20,
  "platform_role": "Merchant"
}' localhost:50052 user.UserService/ListUsers
```

---

### 9. SearchUsers (Admin Only)
**RPC:** `user.UserService/SearchUsers`

**Purpose:** Search users by phone number or name

**Request:**
```json
{
  "query": "0912",
  "limit": 10
}
```

**Response:**
```json
{
  "users": [
    { /* matching user */ }
  ],
  "total_count": 5
}
```

**Search Logic:**
1. Search by phone_number ILIKE '%query%'
2. Search by name ILIKE '%query%'
3. Rank results:
   - Exact phone match first
   - Phone starts with query second
   - Name matches third
4. Limit results (default 10, max 100)

**Use Cases:**
- Admin searching for user by partial phone
- Admin searching for user by name
- Quick lookup in admin panel

**Example:**
```bash
# Search by phone
grpcurl -plaintext -d '{
  "query": "0912",
  "limit": 5
}' localhost:50052 user.UserService/SearchUsers

# Search by name
grpcurl -plaintext -d '{
  "query": "John",
  "limit": 5
}' localhost:50052 user.UserService/SearchUsers
```

---

### 10. DeleteUser (Admin Only)
**RPC:** `user.UserService/DeleteUser`

**Purpose:** Soft delete user account (set is_deleted = true)

**Request:**
```json
{
  "user_id": "uuid",
  "reason": "User requested account deletion"
}
```

**Response:**
```json
{
  "success": true,
  "message": "User deleted successfully"
}
```

**Business Logic:**
1. Verify user exists
2. Set is_deleted = TRUE in database
3. Revoke all refresh tokens
4. Store deletion reason (for audit)

**Note:** This is soft delete. User data remains in database but user cannot login. For GDPR compliance, implement hard delete job separately.

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid",
  "reason": "User requested account deletion"
}' localhost:50052 user.UserService/DeleteUser
```

---

### 11. UpdateUserRole (Admin Only)
**RPC:** `user.UserService/UpdateUserRole`

**Purpose:** Change user platform role (Client → Merchant → Admin)

**Request:**
```json
{
  "user_id": "uuid",
  "new_platform_role": "Merchant"
}
```

**Response:**
```json
{
  "user": { /* updated user */ },
  "message": "User role updated successfully. User must re-login."
}
```

**Business Logic:**
1. Validate new role (must be: Client, Merchant, or Admin)
2. Update platform_role in database
3. **Revoke all refresh tokens** (force re-login to get new role in JWT)

**Valid Roles:**
- `Client`: Regular user
- `Merchant`: Business user with additional permissions
- `Admin`: Full system access

**Security:** After role change, user must re-login to get new access token with updated role.

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid",
  "new_platform_role": "Merchant"
}' localhost:50052 user.UserService/UpdateUserRole
```

---

### 12. GetActiveSessions
**RPC:** `user.UserService/GetActiveSessions`

**Purpose:** Get all active sessions (devices) for user

**Request:**
```json
{
  "user_id": "uuid"
}
```

**Response:**
```json
{
  "sessions": [
    {
      "id": "session-uuid",
      "device_info": "Chrome on Windows",
      "ip_address": "192.168.1.1",
      "created_at": "timestamp",
      "expires_at": "timestamp",
      "is_current": true
    }
  ],
  "total_count": 3
}
```

**Business Logic:**
1. Query refresh_tokens WHERE user_id AND not revoked AND not expired
2. Return list ordered by created_at DESC

**Use Cases:**
- User wants to see all logged-in devices
- Security feature: detect unauthorized access
- User can revoke suspicious sessions

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid"
}' localhost:50052 user.UserService/GetActiveSessions
```

---

### 13. LogoutAllDevices
**RPC:** `user.UserService/LogoutAllDevices`

**Purpose:** Revoke all refresh tokens (logout from all devices)

**Request:**
```json
{
  "user_id": "uuid"
}
```

**Response:**
```json
{
  "success": true,
  "revoked_count": 5,
  "message": "Successfully logged out from 5 device(s)"
}
```

**Business Logic:**
1. UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id
2. Return count of revoked tokens

**Use Cases:**
- User suspects account compromise
- User wants to logout from all devices
- Security measure after password change

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid"
}' localhost:50052 user.UserService/LogoutAllDevices
```

---

### 14. RevokeSession
**RPC:** `user.UserService/RevokeSession`

**Purpose:** Revoke specific session (logout from one device)

**Request:**
```json
{
  "user_id": "uuid",
  "session_id": "session-uuid"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Session revoked successfully"
}
```

**Business Logic:**
1. Verify session belongs to user
2. Revoke token by ID
3. Return success

**Use Case:** User sees unfamiliar device in active sessions and wants to logout that specific device.

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid",
  "session_id": "session-uuid"
}' localhost:50052 user.UserService/RevokeSession
```

---

## Token System

### Access Token (JWT)
**Expiry:** 15 minutes  
**Storage:** Client memory (not localStorage for security)  
**Format:** JWT with HS256 signature

**Claims:**
```json
{
  "user_id": "uuid",
  "platform_role": "Client",
  "type": "access",
  "exp": 1733741700,
  "iat": 1733740800
}
```

**Usage:**
```
Authorization: Bearer <access_token>
```

### Refresh Token (UUID)
**Expiry:** 30 days  
**Storage:** httpOnly cookie (handled by gateway)  
**Format:** UUID v4

**Database Storage:**
- Token is hashed (SHA256) before storing
- Compare: hash(client_token) == stored_hash
- Revocable: Set revoked_at timestamp

### Token Flow

**Initial Login:**
```
1. Client → Login(phone, password)
2. Service → Generate access + refresh tokens
3. Service → Store hashed refresh token in DB
4. Client ← Both tokens
5. Client → Store access token in memory
6. Client → Store refresh token in httpOnly cookie
```

**API Call:**
```
1. Client → API request with Authorization: Bearer <access_token>
2. Gateway → Verify JWT signature + expiry
3. If valid → Forward to service
4. If expired → Return 401
5. Client → Call RefreshToken() with refresh token
6. Service → Verify refresh token in DB
7. Client ← New access token
8. Client → Retry original request
```

**Logout:**
```
1. Client → Logout(refresh_token)
2. Service → Mark token as revoked
3. Client → Delete tokens from storage
4. Access token expires in max 15 minutes
```

## Integration Guide

### For Frontend Developers

**1. Register Flow:**
```javascript
const response = await grpcClient.register({
  phone_number: "0912345678",
  password: "SecurePass123!",
  name: "John Doe",
  platform_role: "Client"
});

// Store tokens
localStorage.setItem('access_token', response.access_token);
// Refresh token should be in httpOnly cookie (gateway handles this)
```

**2. Login Flow:**
```javascript
const response = await grpcClient.login({
  phone_number: "0912345678",
  password: "SecurePass123!"
});

localStorage.setItem('access_token', response.access_token);
// Redirect to dashboard
```

**3. API Call with Auto-Refresh:**
```javascript
async function apiCall(endpoint, data) {
  try {
    const accessToken = localStorage.getItem('access_token');
    return await fetch(endpoint, {
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
      body: JSON.stringify(data)
    });
  } catch (error) {
    if (error.status === 401) {
      // Access token expired - refresh it
      const newToken = await refreshAccessToken();
      localStorage.setItem('access_token', newToken);
      
      // Retry original request
      return apiCall(endpoint, data);
    }
    throw error;
  }
}

async function refreshAccessToken() {
  // Refresh token is in cookie - gateway extracts it
  const response = await grpcClient.refreshToken({});
  return response.access_token;
}
```

**4. Logout:**
```javascript
await grpcClient.logout({
  refresh_token: getRefreshTokenFromCookie()
});

localStorage.removeItem('access_token');
// Redirect to login
```

### For Gateway Developers

**JWT Middleware:**
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.JSON(401, gin.H{"error": "Missing authorization header"})
            c.Abort()
            return
        }

        // Remove "Bearer " prefix
        tokenString = strings.TrimPrefix(tokenString, "Bearer ")

        // Verify JWT
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(jwtSecret), nil
        })

        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Extract claims
        claims := token.Claims.(jwt.MapClaims)
        c.Set("user_id", claims["user_id"])
        c.Set("platform_role", claims["platform_role"])

        c.Next()
    }
}
```

**Role-Based Access:**
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

// Usage
router.GET("/admin/users", AuthMiddleware(), RequireAdmin(), listUsersHandler)
```

## Testing

### List All Available Methods
```bash
grpcurl -plaintext localhost:50052 list user.UserService
```

### Test Complete Flow
```bash
# 1. Register
REGISTER=$(grpcurl -plaintext -d '{
  "phone_number": "0999888777",
  "password": "Test123456",
  "name": "Test User",
  "platform_role": "Client"
}' localhost:50052 user.UserService/Register)

echo $REGISTER

# Extract tokens (requires jq)
ACCESS_TOKEN=$(echo $REGISTER | jq -r '.accessToken')
REFRESH_TOKEN=$(echo $REGISTER | jq -r '.refreshToken')
USER_ID=$(echo $REGISTER | jq -r '.user.id')

# 2. Get Profile
grpcurl -plaintext -d "{\"user_id\": \"$USER_ID\"}" \
  localhost:50052 user.UserService/GetUserProfile

# 3. Refresh Token
grpcurl -plaintext -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}" \
  localhost:50052 user.UserService/RefreshToken

# 4. Logout
grpcurl -plaintext -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}" \
  localhost:50052 user.UserService/Logout
```

## Configuration

### Environment Variables
```bash
# Database
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/user_db?sslmode=disable"

# JWT Settings
JWT_SECRET="your-secret-key-minimum-32-characters-long"
JWT_EXPIRY_HOURS="720"  # 30 days (deprecated, using constants now)

# Server
SERVER_PORT="50052"

# Database Pool (optional)
DB_MAX_CONNS="25"
DB_MIN_CONNS="5"
DB_MAX_CONN_LIFE="300"
```

### Token Expiry Configuration
Edit `user/internal/service/token_helpers.go`:
```go
const (
    AccessTokenExpiry  = 15 * time.Minute    // Change here
    RefreshTokenExpiry = 30 * 24 * time.Hour // Change here
)
```

## Troubleshooting

### Service won't start
```bash
# Check PostgreSQL is running
docker-compose ps policy_postgres

# Check database exists
docker exec -it policy_postgres psql -U postgres -l

# Check migrations ran
docker-compose logs user_migrate
```

### Token verification fails
```bash
# Verify JWT secret is same in User Service and Gateway
# Check token is not expired
# Decode JWT: https://jwt.io/

# Check refresh token in database
docker exec -it policy_postgres psql -U postgres -d user_db -c \
  "SELECT id, user_id, revoked_at, expires_at FROM refresh_tokens LIMIT 5;"
```

### Can't connect from gateway
```bash
# Test gRPC connection
grpcurl -plaintext localhost:50052 list

# Check service is listening
netstat -an | grep 50052

# Check docker network
docker network inspect policy-system_default
```

## Performance

### Database Indexes
```sql
-- Already created in migrations
CREATE INDEX idx_users_phone_number ON users(phone_number);
CREATE INDEX idx_users_platform_role ON users(platform_role);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

### Connection Pool Settings
Recommended for production:
```
DB_MAX_CONNS=25
DB_MIN_CONNS=5
DB_MAX_CONN_LIFE=300
```

### Token Cleanup Job
Run daily to delete expired tokens:
```bash
docker-compose up user_cleanup
```

## Security Best Practices

1. **HTTPS Only:** Always use HTTPS in production
2. **Token Storage:**
   - Access Token: Memory only (not localStorage)
   - Refresh Token: httpOnly cookie
3. **Password Policy:** Enforced (8+ chars, uppercase, lowercase, digit)
4. **Rate Limiting:** Apply in gateway (5 req/sec for auth endpoints)
5. **Token Rotation:** Consider rotating refresh token on each refresh
6. **Revocation:** Always revoke all tokens on password change or role change

## Related Services

- **Gateway Service:** HTTP/REST to gRPC proxy, JWT verification
- **Consent Service:** Requires user_id from access token
- **Document Service:** Requires user_id for owner field

## Support

For issues or questions:
1. Check logs: `docker-compose logs -f user_service`
2. Check database state: `docker exec -it policy_postgres psql -U postgres -d user_db`
3. Test gRPC methods: `grpcurl -plaintext localhost:50052 list user.UserService`
