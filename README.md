# Policy & Consent Management System

**Microservices-based backend system for managing privacy policies and user consents**

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)
[![gRPC](https://img.shields.io/badge/gRPC-1.57-green.svg)](https://grpc.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-blue.svg)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-blue.svg)](https://docs.docker.com/compose/)

---

## 📋 Overview

Enterprise-grade microservices system handling privacy policy management and user consent tracking for compliance with GDPR, CCPA, and other data protection regulations.

**Current Status:** Document Service complete ✅, User Service enhancement in progress 🔄 (dual token system planned)

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Gateway Service                       │
│                    (HTTP/REST → gRPC)                        │
│                      [Coming Soon]                           │
└──────────────┬──────────────┬──────────────┬────────────────┘
               │              │              │
       ┌───────▼──────┐ ┌─────▼──────┐ ┌────▼───────┐
       │  Document    │ │   User     │ │  Consent   │
       │  Service     │ │  Service   │ │  Service   │
       │  (50051)     │ │  (50052)   │ │  (50053)   │
       └───────┬──────┘ └─────┬──────┘ └────┬───────┘
               │              │              │
       ┌───────▼──────────────▼──────────────▼───────┐
       │          PostgreSQL (3 databases)           │
       │   document_db | user_db | consent_db        │
       └─────────────────────────────────────────────┘
```

### Services

| Service | Port | Status | Features |
|---------|------|--------|----------|
| **Document Service** | 50051 | ✅ Complete | Policy CRUD, Versioning, Audit Trail |
| **User Service** | 50052 | 🔄 In Progress | Basic Auth ✅, Dual Token System 📋 |
| **Consent Service** | 50053 | ✅ Complete | Consent Recording, Bulk Operations |
| **Gateway Service** | 8080 | 📋 Planned | HTTP/REST API, File Upload |

---

## ✨ Features

### Document Service (Policy Management)
- ✅ **Versioning System** - Immutable updates preserve complete audit trail
- ✅ **Flexible Scheduling** - Schedule policy changes with future effective dates
- ✅ **Audit Trail API** - Retrieve complete document history
- ✅ **Enhanced Validation** - Platform enum, file URL validation with extension whitelist
- ✅ **Multiple Content Types** - Support both HTML content and file URLs

### User Service (Authentication & User Management)
**Completed:**
- ✅ **User Registration** - Phone number, password with bcrypt hashing
- ✅ **JWT Authentication** - HS256 single token (30-day expiry)
- ✅ **Secure Password Storage** - bcrypt with cost factor 10

**Planned (Dual Token System):**
- 📋 **Access Token** - Short-lived (15 min), stateless, used for API calls
- 📋 **Refresh Token** - Long-lived (30 days), stored in DB, enables token renewal
- 📋 **Token Refresh** - Renew access token without re-login
- 📋 **Logout** - Revoke refresh tokens (single device or all devices)
- 📋 **User Management** - Get/Update profile, Change password, Delete account
- 📋 **Security** - Rate limiting, account lockout, audit logging

### Consent Service (Compliance)
- ✅ **Consent Recording** - Track user agreements with IP, user agent, timestamp
- ✅ **Consent Revocation** - Soft delete with audit trail
- ✅ **Bulk Operations** - Multi-consent recording in single transaction
- ✅ **Pending Consent Check** - Identify policies requiring user consent
- ✅ **GDPR Compliance** - Complete consent lifecycle tracking

---

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.25+ (for local development)
- grpcurl (for testing)

### Installation

```bash
# Clone repository
git clone https://github.com/thatlq1812/policy-system.git
cd policy-system

# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f document_service
```

### Verify Services

```bash
# Document Service (Port 50051)
grpcurl -plaintext localhost:50051 list document.DocumentService

# User Service (Port 50052)
grpcurl -plaintext localhost:50052 list user.UserService

# Consent Service (Port 50053)
grpcurl -plaintext localhost:50053 list consent.ConsentService
```

---

## 📡 API Examples

### Document Service

#### Create Policy
```bash
grpcurl -plaintext -d '{
  "document_name": "Privacy Policy",
  "platform": "Client",
  "is_mandatory": true,
  "effective_timestamp": 1733740800,
  "content_html": "<h1>Privacy Policy v1.0</h1>",
  "created_by": "admin@example.com"
}' localhost:50051 document.DocumentService/CreatePolicy
```

#### Update Policy (Create New Version)
```bash
grpcurl -plaintext -d '{
  "document_name": "Privacy Policy",
  "platform": "Client",
  "is_mandatory": true,
  "effective_timestamp": 0,
  "content_html": "<h1>Privacy Policy v2.0</h1>",
  "updated_by": "admin@example.com"
}' localhost:50051 document.DocumentService/UpdatePolicy
```

#### Get Policy History (Audit Trail)
```bash
grpcurl -plaintext -d '{
  "platform": "Client",
  "document_name": "Privacy Policy"
}' localhost:50051 document.DocumentService/GetPolicyHistory
```

### User Service

#### Register User
```bash
grpcurl -plaintext -d '{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "full_name": "John Doe",
  "phone_number": "+84123456789"
}' localhost:50052 user.UserService/Register
```

#### Login
```bash
grpcurl -plaintext -d '{
  "email": "user@example.com",
  "password": "SecurePass123!"
}' localhost:50052 user.UserService/Login
```

### Consent Service

#### Record Consent
```bash
grpcurl -plaintext -d '{
  "consents": [
    {
      "user_id": "uuid-here",
      "document_id": "uuid-here",
      "has_consented": true,
      "consent_method": "clicked_agree_button",
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0..."
    }
  ]
}' localhost:50053 consent.ConsentService/RecordConsent
```

---

## 🗄️ Database Schema

### Document Service (document_db)
```sql
policy_documents
├── id (UUID, PK)
├── document_name (VARCHAR)
├── platform (VARCHAR) -- "Client" or "Merchant"
├── is_mandatory (BOOLEAN)
├── effective_timestamp (BIGINT) -- Version identifier
├── content_html (TEXT)
├── file_url (TEXT)
├── created_at (TIMESTAMP)
└── created_by (VARCHAR)

-- Indexes for performance
INDEX idx_policy_platform_name ON (platform, document_name)
INDEX idx_policy_effective_ts ON (effective_timestamp DESC)
```

### User Service (user_db)
```sql
users
├── id (UUID, PK)
├── email (VARCHAR, UNIQUE)
├── password_hash (VARCHAR) -- bcrypt
├── full_name (VARCHAR)
├── phone_number (VARCHAR)
├── created_at (TIMESTAMP)
└── updated_at (TIMESTAMP)
```

### Consent Service (consent_db)
```sql
user_consents
├── id (UUID, PK)
├── user_id (UUID)
├── document_id (UUID)
├── has_consented (BOOLEAN)
├── consent_method (VARCHAR)
├── consented_at (TIMESTAMP)
├── ip_address (VARCHAR)
├── user_agent (TEXT)
├── is_deleted (BOOLEAN) -- Soft delete
└── deleted_at (TIMESTAMP)
```

---

## 🔧 Configuration

### Environment Variables

```bash
# Document Service
DATABASE_URL=postgres://user:pass@localhost:5432/document_db
GRPC_PORT=50051
DB_MAX_CONN=10

# User Service
DATABASE_URL=postgres://user:pass@localhost:5432/user_db
GRPC_PORT=50052
JWT_SECRET=your-64-byte-hex-secret
JWT_EXPIRY_DAYS=30

# Consent Service
DATABASE_URL=postgres://user:pass@localhost:5432/consent_db
GRPC_PORT=50053
DB_MAX_CONN=10
```

### Docker Compose

```yaml
# docker-compose.yml structure
services:
  policy_postgres:      # Shared PostgreSQL instance
  document_service:     # Port 50051
  document_migrate:     # Auto-migrations
  user_service:         # Port 50052
  user_migrate:         # Auto-migrations
  consent_service:      # Port 50053
  consent_migrate:      # Auto-migrations
```

---

## 🧪 Testing

### Testing Document Service

```bash
# Run comprehensive test suite
cd docs
bash DOCUMENT_SERVICE_COMPLETE.md  # Contains all test commands

# Quick smoke test
grpcurl -plaintext localhost:50051 list document.DocumentService
# Expected: 4 methods (CreatePolicy, GetLatestPolicyByPlatform, UpdatePolicy, GetPolicyHistory)
```

### Test Statistics
- **Document Service:** 30+ test cases, 100% pass rate
- **User Service:** Core auth flows tested
- **Consent Service:** CRUD and bulk operations tested

---

## 📚 Documentation

### Main Documentation
- **[IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md)** - Complete project roadmap with lessons learned
- **[CHANGELOG.md](docs/CHANGELOG.md)** - Detailed version history
- **[DOCUMENT_SERVICE_COMPLETE.md](docs/DOCUMENT_SERVICE_COMPLETE.md)** - Comprehensive Document Service report

### Architecture & Design
- Microservices pattern with gRPC
- Layered architecture: Domain → Repository → Service → Handler
- Immutable versioning for audit compliance
- Soft delete pattern for data retention
- No foreign keys (microservices independence)

### Key Design Decisions
1. **Immutable Versioning:** UPDATE = INSERT new record with new timestamp
2. **Unix Epoch Timestamps:** Version identifiers instead of integer versions
3. **Soft Delete:** Audit trail and legal compliance
4. **JWT Authentication:** HS256 with 64-byte secret
5. **Bulk Operations:** Transaction-based for atomicity

---

## 🔐 Security

### Implemented
- ✅ Password hashing with bcrypt (cost factor 10)
- ✅ JWT authentication with expiry
- ✅ SQL injection prevention (parameterized queries)
- ✅ Input validation at service layer
- ✅ File URL validation with extension whitelist
- ✅ GDPR compliance (consent tracking with IP/user agent)

### Best Practices
- Environment variables for secrets
- Never store passwords in plaintext
- Validate all external inputs
- Use HTTPS in production (Gateway will handle)
- Regular security audits

---

## 🚀 Performance

### Optimization Strategies
- **Database Indexes:** Non-unique indexes on frequently queried columns
- **Connection Pooling:** pgxpool with configurable max connections
- **Docker Images:** Multi-stage builds (~20MB vs ~800MB)
- **Query Optimization:** ORDER BY + LIMIT for latest version queries
- **Bulk Operations:** Single transaction for multiple inserts

### Benchmarks (Preliminary)
- CreatePolicy: ~10ms average
- GetLatestPolicy: ~5ms average (with index)
- UpdatePolicy: ~15ms average (check + insert)
- GetPolicyHistory: ~20ms for 10 versions

---

## 📈 Roadmap

### ✅ Completed (Phase 1-3)
- [x] Infrastructure setup (Docker, Go workspace, protobuf)
- [x] Document Service with versioning system (100% complete)
- [x] User Service basic authentication (Register + Login with single JWT)
- [x] Consent Service with bulk operations (100% complete)
- [x] Database migrations and indexes
- [x] Comprehensive testing (30+ test cases for Document Service)

### 🔄 In Progress (Phase 4a)
- [ ] **User Service Enhancement** (Priority 1 - Current Focus)
  - [ ] Dual Token System (Access Token + Refresh Token)
  - [ ] Token refresh mechanism
  - [ ] Logout (single device + all devices)
  - [ ] Additional user management RPCs (GetProfile, UpdateProfile, ChangePassword)
  - [ ] Security enhancements (rate limiting, account lockout, audit logging)
  - **Estimated:** 1 week full-time effort
  - **Reason:** Required before Gateway to enable proper session management

### 📋 Planned (Phase 4b)
- [ ] **Gateway Service** (HTTP/REST → gRPC proxy)
  - [ ] RESTful API design with dual token support
  - [ ] Authentication middleware (JWT verification)
  - [ ] Token refresh endpoint
  - [ ] File upload support (S3/MinIO)
  - [ ] API composition and enrichment
  - **Estimated:** 1-2 weeks full-time effort

### 🎯 Future Enhancements (Phase 5+)
- [ ] Unit testing framework
- [ ] Integration tests
- [ ] Structured logging (levels, correlation IDs)
- [ ] Monitoring and metrics (Prometheus)
- [ ] Health check endpoints
- [ ] Rate limiting
- [ ] API documentation (Swagger/OpenAPI)
- [ ] Admin dashboard

---

## 🤝 Contributing

### Development Workflow
1. Create feature branch: `git checkout -b feature/description`
2. Implement changes with tests
3. Update documentation
4. Run tests: `go test ./...`
5. Build locally: `docker-compose up --build`
6. Submit PR with detailed description

### Code Standards
- Follow Go best practices (gofmt, golint)
- Write clear commit messages (conventional commits)
- Add tests for new features
- Update documentation
- Use American English in code/docs

---

## 📝 License

This project is proprietary and confidential.

---

## 👥 Team

**Backend Intern:** thatlq1812  
**Role:** System architect, backend developer  
**Stack:** Go, gRPC, PostgreSQL, Docker

---

## 📞 Support

### Issues & Questions
- Create GitHub issue for bugs/features
- Check documentation first
- Include logs and reproduction steps

### Useful Commands

```bash
# Rebuild specific service
docker-compose up --build -d document_service

# View logs
docker-compose logs -f document_service

# Connect to database
docker exec -it policy_postgres psql -U postgres -d document_db

# Stop all services
docker-compose down

# Clean rebuild (remove volumes)
docker-compose down -v && docker-compose up --build -d
```

---

## 🎓 Lessons Learned

### Technical Insights
1. **Validation Order Matters:** Apply defaults before validation
2. **Database Constraints:** Too rigid for versioning systems
3. **Proto Field Naming:** Use plural for repeated fields
4. **Resource Management:** Always defer Close() for connections
5. **Domain Constants:** Better than magic strings

### Architectural Insights
1. **Immutable Versioning:** Simple INSERT strategy works best
2. **Layered Architecture:** Keeps concerns separated
3. **Repository Pattern:** Enables testability and flexibility
4. **Error Wrapping:** Preserves debugging context

### Business Insights
1. **Audit Trail:** Legal compliance requirement
2. **Point-in-Time Queries:** Enable regulatory reporting
3. **Flexible Timestamps:** Enable policy scheduling
4. **Clear Error Messages:** Improve developer experience

---

**Last Updated:** December 9, 2025  
**Version:** 0.2.0  
**Status:** Document Service Complete ✅, Gateway Service In Planning
