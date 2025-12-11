# Gateway Service - API Gateway

HTTP/REST API Gateway for routing and authenticating requests to gRPC microservices.

**Version:** 0.1.0 | **Status:** Planning Phase | **Port:** 8080 (HTTP/REST) | **Framework:** Gin

---

## Table of Contents

1. [Overview](#overview)
2. [Security](#security)
3. [Quick Start](#quick-start)
4. [Environment Variables](#environment-variables)
5. [API Reference](#api-reference)
6. [Troubleshooting](#troubleshooting)

---

## Overview

### Features

- HTTP/REST to gRPC protocol translation
- JWT authentication and authorization
- Centralized error handling
- Request logging
- Proxying to User, Document, and Consent Services

### Architecture

```
Client (HTTP/REST)
    |
Gateway Service :8080
    |-- JWT Verification Middleware
    |-- Error Handling Middleware
    |-- gRPC Clients
        |-- User Service :50052
        |-- Document Service :50051
        +-- Consent Service :50053
```

---

## Security

### Admin Account Management

For security reasons, Admin accounts **cannot be created through the public registration API**. This prevents unauthorized users from self-promoting to Admin role.

#### Super Admin Bootstrap

A Super Admin account is created automatically during database initialization:

**Credentials:**
- Phone: `0900000000`
- Password: `SuperAdmin@123`
- Role: `Admin`

**⚠️ IMPORTANT:** Change this password immediately after first login in production environments!

#### Creating Additional Admin Accounts

Only existing Admin users can create new Admin accounts using the admin-only endpoint:

**Endpoint:** `POST /api/v1/admin/create-admin`

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/create-admin \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "0902000000",
    "password": "NewAdmin@123",
    "name": "Second Admin"
  }'
```

**Response:**
```json
{
  "code": "201",
  "message": "Admin user created successfully",
  "data": {
    "user": {
      "id": "uuid",
      "phone_number": "0902000000",
      "name": "Second Admin",
      "platform_role": "Admin",
      "created_at": 1733740800
    }
  }
}
```

**Security Features:**
- ✅ Admin role blocked from public registration endpoint
- ✅ Validation at gateway level (HTTP 400)
- ✅ Explicit security check (HTTP 403)
- ✅ Admin-only endpoint requires authentication
- ✅ Only existing Admin can create new Admin accounts

---

## Quick Start

### Run with Docker (Recommended)
```bash
# From project root d:/w2
docker-compose up -d gateway

# Check logs
docker-compose logs -f gateway

# Verify health check
curl http://localhost:8080/health
```

### Run Locally (Development)
```bash
cd gateway

# Install dependencies
go mod download

# Set environment variables (example .env)
# JWT_SECRET="your-secret-key-min-32-chars-recommended-64"
# USER_SERVICE_URL="localhost:50052"
# DOCUMENT_SERVICE_URL="localhost:50051"
# CONSENT_SERVICE_URL="localhost:50053"
# GRPC_PORT="8080"

# Run service
go run cmd/server/main.go

# Or build binary
go build -o bin/gateway cmd/server/main.go
./bin/gateway
```

---

## Environment Variables
| Variable             | Description                          | Default         | Required |
|----------------------|--------------------------------------|-----------------|----------|
| `JWT_SECRET`         | Secret key for JWT signing           | -               | Yes      |
| `USER_SERVICE_URL`   | User Service gRPC endpoint           | `localhost:50052` | Yes      |
| `DOCUMENT_SERVICE_URL` | Document Service gRPC endpoint       | `localhost:50051` | Yes      |
| `CONSENT_SERVICE_URL` | Consent Service gRPC endpoint        | `localhost:50053` | Yes      |
| `GRPC_PORT`          | HTTP/REST server port                | `8080`          | No       |

---

## API Reference

### Health Check

**GET /health**

Purpose: Check if the gateway service is operational.

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

### Authentication Endpoints (`/api/auth`)

**1. POST /api/auth/register**

Purpose: Register a new user account.

**Request Body:**
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

**2. POST /api/auth/login**

Purpose: Authenticate user and obtain access and refresh tokens.

**Request Body:**
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

**3. POST /api/auth/refresh**

Purpose: Obtain a new access token using a valid refresh token.

**Request Body:**
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
    "refresh_token": "uuid-refresh-token"
  }'
```

**4. POST /api/auth/logout**

Purpose: Revoke a specific refresh token, invalidating the session.

**Request Body:**
```json
{
  "refresh_token": "uuid-refresh-token"
}
```

**Response:** `200 OK`
```json
{
  "message": "Successfully logged out"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "uuid-refresh-token"
  }'
```

---

### User Profile Endpoints (`/api/users`)

**All endpoints require a valid JWT Access Token in the `Authorization` header.**

**1. GET /api/users/profile**

Purpose: Retrieve the authenticated user's profile.

**Request:** `Authorization: Bearer <access_token>`

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
curl -X GET http://localhost:8080/api/users/profile \
  -H "Authorization: Bearer <your-access-token>"
```

**2. PUT /api/users/profile**

Purpose: Update the authenticated user's profile information.

**Request Body:** (fields are optional)
```json
{
  "name": "Jane Doe",
  "phone_number": "0987654321"
}
```

**Response:** `200 OK`
```json
{
  "id": "user-uuid",
  "phone_number": "0987654321",
  "name": "Jane Doe",
  "platform_role": "Client",
  "created_at": 1733740800,
  "updated_at": 1733740800
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/users/profile \
  -H "Authorization: Bearer <your-access-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Jane Doe"
  }'
```

**3. PUT /api/users/password**

Purpose: Change the authenticated user's password.

**Request Body:**
```json
{
  "old_password": "SecurePass123!",
  "new_password": "NewSecurePass456!"
}
```

**Response:** `200 OK`
```json
{
  "message": "Password changed successfully"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/users/password \
  -H "Authorization: Bearer <your-access-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "SecurePass123!",
    "new_password": "NewSecurePass456!"
  }'
```

---

### Admin Endpoints (`/api/admin`)

**All endpoints require a valid JWT Access Token in the `Authorization` header and `Admin` platform role.**

**1. GET /api/admin/users**

Purpose: List users with optional pagination and role filtering.

**Query Parameters:**
- `page`: Page number (default: 1)
- `pageSize`: Items per page (default: 10, max: 100)
- `role`: Filter by platform role (e.g., `Client`, `Merchant`, `Admin`)
- `includeDeleted`: Include soft-deleted users (default: false)

**Response:** `200 OK`
```json
{
  "users": [ /* array of user objects */ ],
  "total_count": 100,
  "total_pages": 10,
  "current_page": 1
}
```

**Example:**
```bash
curl -X GET "http://localhost:8080/api/admin/users?page=1&pageSize=5&role=Client" \
  -H "Authorization: Bearer <admin-access-token>"
```

**2. GET /api/admin/users/search**

Purpose: Search users by name or phone number.

**Query Parameters:**
- `query`: Search term (name or phone number)
- `limit`: Maximum number of results (default: 10)

**Response:** `200 OK`
```json
{
  "users": [ /* array of user objects */ ]
}
```

**Example:**
```bash
curl -X GET "http://localhost:8080/api/admin/users/search?query=John&limit=5" \
  -H "Authorization: Bearer <admin-access-token>"
```

**3. DELETE /api/admin/users/:id**

Purpose: Soft delete a user by ID.

**Path Parameters:**
- `:id`: User UUID

**Request Body:**
```json
{
  "reason": "Inactivity"
}
```

**Response:** `200 OK`
```json
{
  "message": "User soft-deleted successfully"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/admin/users/user-uuid-to-delete \
  -H "Authorization: Bearer <admin-access-token>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Inactivity"}'
```

**4. PUT /api/admin/users/:id/role**

Purpose: Update a user's platform role.

**Path Parameters:**
- `:id`: User UUID

**Request Body:**
```json
{
  "new_platform_role": "Admin"
}
```

**Response:** `200 OK`
```json
{
  "id": "user-uuid",
  "phone_number": "0912345678",
  "name": "John Doe",
  "platform_role": "Admin",
  "created_at": 1733740800,
  "updated_at": 1733740800
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/admin/users/user-uuid-to-update/role \
  -H "Authorization: Bearer <admin-access-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "new_platform_role": "Merchant"
  }'
```

---

### Session Management Endpoints (`/api/sessions`)

**All endpoints require a valid JWT Access Token in the `Authorization` header.**

**1. GET /api/sessions**

Purpose: Retrieve all active sessions for the authenticated user.

**Request:** `Authorization: Bearer <access_token>`

**Response:** `200 OK`
```json
{
  "sessions": [
    {
      "id": "session-uuid-1",
      "user_id": "user-uuid",
      "expires_at": 1736332800,
      "device_info": "Chrome on Windows",
      "ip_address": "192.168.1.10"
    },
    { /* ... other sessions */ }
  ],
  "total_active_sessions": 2
}
```

**Example:**
```bash
curl -X GET http://localhost:8080/api/sessions \
  -H "Authorization: Bearer <your-access-token>"
```

**2. DELETE /api/sessions/all**

Purpose: Revoke all active sessions for the authenticated user.

**Request:** `Authorization: Bearer <access_token>`

**Response:** `200 OK`
```json
{
  "message": "All sessions revoked successfully",
  "revoked_count": 2
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/sessions/all \
  -H "Authorization: Bearer <your-access-token>"
```

**3. DELETE /api/sessions/:id**

Purpose: Revoke a specific session by its ID for the authenticated user.

**Path Parameters:**
- `:id`: Session (refresh token) UUID

**Request:** `Authorization: Bearer <access_token>`

**Response:** `200 OK`
```json
{
  "message": "Session revoked successfully"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/sessions/session-uuid-to-revoke \
  -H "Authorization: Bearer <your-access-token>"
```

---

### User Statistics Endpoints (`/api/stats`)

**All endpoints require a valid JWT Access Token in the `Authorization` header and `Admin` platform role.**

**1. GET /api/stats/users**

Purpose: Retrieve comprehensive statistics about user accounts.

**Request:** `Authorization: Bearer <admin-access-token>`

**Response:** `200 OK`
```json
{
  "total_users": 100,
  "active_users": 90,
  "deleted_users": 10,
  "total_active_sessions": 150,
  "users_by_role": {
    "Client": 70,
    "Merchant": 20,
    "Admin": 10
  }
}
```

**Example:**
```bash
curl -X GET http://localhost:8080/api/stats/users \
  -H "Authorization: Bearer <admin-access-token>"
```

---

## Troubleshooting

- **Gateway not starting:** Check `docker-compose logs gateway` for errors.
- **gRPC connection issues to microservices:** Verify `*_SERVICE_URL` environment variables and microservice container status.
- **JWT errors:** Ensure `JWT_SECRET` is set correctly and tokens are valid.
- **401 Unauthorized / 403 Forbidden:** Check JWT token validity and user's platform role.
