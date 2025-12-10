# Consent Service - User Consent Management

User consent management service for tracking policy document acceptances with full audit trail and GDPR compliance.

**Version:** 1.0.0 | **Status:** Production Ready | **Port:** 50053 (gRPC) | **Methods:** 7/7

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Environment Variables](#environment-variables)
4. [Database Schema](#database-schema)
5. [API Reference](#api-reference)
6. [Testing Guide](#testing-guide)
7. [Security & Compliance](#security--compliance)

---

## Overview

### Features

- Document verification with Document Service
- Idempotent consent recording (duplicate prevention)
- Complete consent history tracking
- Version management with `is_latest` flag
- Comprehensive statistics and analytics
- GDPR-compliant audit trail
- Transaction support for atomic operations

### Architecture

```
Client/Gateway
    |
Consent Service (gRPC)
    |-- Service Layer (Business Logic)
    |-- Repository Layer (Data Access)
    |-- Document Service Client (gRPC)
    +-- PostgreSQL Database
        +-- user_consents table
```

---

## Quick Start

### Run with Docker (Recommended)
```bash
# From project root d:/w2
docker-compose up -d consent_service

# Check logs
docker-compose logs -f consent_service

# Run migrations
docker-compose up consent_migrate

# Verify service
grpcurl -plaintext localhost:50053 list consent.ConsentService
```

### Run Locally (Development)
```bash
cd consent

# Install dependencies
go mod download

# Set environment variables (example .env)
# DATABASE_URL="postgresql://postgres:postgres@localhost:5432/consent_db?sslmode=disable"
# DOCUMENT_SERVICE_URL="localhost:50051"
# GRPC_PORT="50053"

# Run service
go run cmd/server/main.go

# Or build binary
go build -o bin/consent cmd/server/main.go
./bin/consent
```

---

## Environment Variables
| Variable             | Description                          | Default         | Required |
|----------------------|--------------------------------------|-----------------|----------|
| `DATABASE_URL`       | PostgreSQL connection string         | -               | Yes      |
| `DOCUMENT_SERVICE_URL` | Document Service gRPC endpoint       | `localhost:50051` | Yes      |
| `GRPC_PORT`          | gRPC server port                     | `50053`         | No       |

---

## Database Schema

### user_consents table
```sql
CREATE TABLE user_consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    platform VARCHAR(50) NOT NULL, -- Client | Merchant
    document_id UUID NOT NULL,
    document_name VARCHAR(255) NOT NULL,
    version_timestamp BIGINT NOT NULL,
    agreed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    agreed_file_url TEXT,
    consent_method VARCHAR(50) NOT NULL, -- REGISTRATION | UI | API
    ip_address VARCHAR(50),
    user_agent TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP,
    is_latest BOOLEAN DEFAULT TRUE, -- Indicates if this is the latest/current consent
    revoked_at TIMESTAMP,
    revoked_reason TEXT,
    revoked_by VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient lookup of active consents by user and document
CREATE UNIQUE INDEX idx_active_consents 
ON user_consents (user_id, document_id, version_timestamp) 
WHERE is_deleted = FALSE AND is_latest = TRUE;

-- Index for efficient history queries
CREATE INDEX idx_user_document_history 
ON user_consents(user_id, document_id, version_timestamp DESC, is_latest);

-- Index for latest consents lookup
CREATE INDEX idx_latest_consents 
ON user_consents(user_id, is_latest) 
WHERE is_latest = TRUE AND is_deleted = FALSE;
```

**Migrations:**
- `000001_create_user_consents_table.up.sql`
- `000002_add_history_tracking.up.sql`

---

## API Reference

### Available Methods (7 Total)

**Core Operations:**
```
consent.ConsentService.RecordConsent        - Record user consent for documents
consent.ConsentService.CheckConsent         - Check if user has consented to a document version
consent.ConsentService.GetUserConsents      - Retrieve all consents for a user
consent.ConsentService.CheckPendingConsents - Identify policies user has not yet consented to
consent.ConsentService.RevokeConsent        - Soft delete (revoke) a specific user consent
```

**History & Analytics:**
```
consent.ConsentService.GetConsentHistory    - Retrieve full consent history for a user and document
consent.ConsentService.GetConsentStats      - Get aggregated consent statistics
```

---

## Testing Guide

### Using grpcurl

To test the Consent Service endpoints, ensure the service is running and use `grpcurl` from your terminal.

**Example: RecordConsent**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-id-123",
  "platform": "Client",
  "consents": [{
    "document_id": "doc-id-456",
    "document_name": "Privacy Policy",
    "version_timestamp": 1678886400,
    "agreed_file_url": "https://example.com/privacy_v1.pdf"
  }],
  "consent_method": "UI",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0"
}' localhost:50053 consent.ConsentService/RecordConsent
```

**Example: GetConsentHistory**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-id-123",
  "document_id": "doc-id-456"
}' localhost:50053 consent.ConsentService/GetConsentHistory
```

**Example: GetConsentStats**
```bash
grpcurl -plaintext -d '{"platform": "Client"}' localhost:50053 consent.ConsentService/GetConsentStats
```

---

## Security & Compliance

### GDPR Compliance Features
- **Right to Withdraw:** `RevokeConsent` method supports user withdrawal of consent.
- **Audit Trail:** Detailed tracking of consent actions including timestamps, IP, and user agent.
- **Version Tracking:** Consents are linked to specific document versions for clarity.
- **Data Portability:** `GetUserConsents` supports user data export.

### Best Practices
- All consent events are explicitly recorded, no implied consent.
- Strict version tracking for all policy documents.
- Comprehensive revocation reasons are stored.
- Idempotent operations ensure data consistency.
- Documents are verified with the Document Service prior to recording consent.
    "consented_at": "timestamp",
    "revoked_at": null,
    "revoked_reason": null,
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
}
```

**Use Cases:**
- Audit trail lookup
- Display consent details in admin panel
- Verify specific consent record

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": "consent-uuid"
}' localhost:50053 consent.ConsentService/GetConsentByID
```

---

### 3. GetUserConsents
**RPC:** `consent.ConsentService/GetUserConsents`

**Purpose:** Get all consents for a specific user

**Request:**
```json
{
  "user_id": "user-uuid"
}
```

**Response:**
```json
{
  "consents": [
    {
      "id": "consent-1",
      "user_id": "user-uuid",
      "document_id": "privacy-policy-uuid",
      "version": 3,
      "status": "Accepted",
      "consented_at": "timestamp",
      // ... other fields
    },
    {
      "id": "consent-2",
      "user_id": "user-uuid",
      "document_id": "terms-uuid",
      "version": 2,
      "status": "Accepted",
      "consented_at": "timestamp",
      // ... other fields
    }
  ],
  "total_count": 2
}
```

**Business Logic:**
1. Query WHERE user_id = ? AND is_deleted = FALSE
2. ORDER BY consented_at DESC
3. Return all matching consents

**Use Cases:**
- User profile: Show all accepted policies
- Compliance: Display user's consent history
- GDPR: Data export for user

**Frontend Display:**
```javascript
const response = await consentService.getUserConsents({
  user_id: currentUserId
});

response.consents.forEach(consent => {
  displayConsentCard({
    documentId: consent.document_id,
    version: consent.version,
    status: consent.status,
    acceptedDate: new Date(consent.consented_at * 1000)
  });
});
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid"
}' localhost:50053 consent.ConsentService/GetUserConsents
```

---

### 4. GetDocumentConsents
**RPC:** `consent.ConsentService/GetDocumentConsents`

**Purpose:** Get all users who consented to a specific document (admin analytics)

**Request:**
```json
{
  "document_id": "privacy-policy-v3-uuid"
}
```

**Response:**
```json
{
  "consents": [
    {
      "id": "consent-1",
      "user_id": "user-1-uuid",
      "document_id": "privacy-policy-v3-uuid",
      "version": 3,
      "status": "Accepted",
      "consented_at": "timestamp"
    },
    {
      "id": "consent-2",
      "user_id": "user-2-uuid",
      "document_id": "privacy-policy-v3-uuid",
      "version": 3,
      "status": "Accepted",
      "consented_at": "timestamp"
    }
  ],
  "total_count": 150
}
```

**Business Logic:**
1. Query WHERE document_id = ? AND is_deleted = FALSE
2. ORDER BY consented_at DESC
3. Return all matching consents

**Use Cases:**
- Admin: See who accepted specific policy version
- Analytics: Consent acceptance rate
- Compliance: Audit which users are on old policy versions

**Admin Dashboard Example:**
```javascript
// Show acceptance stats
const response = await consentService.getDocumentConsents({
  document_id: selectedDocumentId
});

const stats = {
  totalAccepted: response.consents.filter(c => c.status === "Accepted").length,
  totalRejected: response.consents.filter(c => c.status === "Rejected").length,
  totalRevoked: response.consents.filter(c => c.status === "Revoked").length
};

displayStats(stats);
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "document_id": "document-uuid"
}' localhost:50053 consent.ConsentService/GetDocumentConsents
```

---

### 5. UpdateConsentStatus
**RPC:** `consent.ConsentService/UpdateConsentStatus`

**Purpose:** Change consent status (Accepted → Revoked) with reason

**Request:**
```json
{
  "id": "consent-uuid",
  "status": "Revoked",
  "revoked_reason": "User requested data deletion"
}
```

**Response:**
```json
{
  "consent": {
    "id": "consent-uuid",
    "user_id": "user-uuid",
    "document_id": "document-uuid",
    "version": 3,
    "status": "Revoked",
    "consented_at": "original-timestamp",
    "revoked_at": "1733745000",
    "revoked_reason": "User requested data deletion",
    "updated_at": "1733745000"
  }
}
```

**Business Logic:**
1. Get consent by id
2. Validate new status (can only change to Revoked)
3. Update status = new_status
4. If status = Revoked:
   - Set revoked_at = NOW()
   - Set revoked_reason = provided reason
5. Set updated_at = NOW()

**Status Transitions:**
```
Accepted → Revoked ✅ (GDPR: user withdraws consent)
Rejected → Revoked ✅ (rare, but allowed)
Revoked → Accepted ❌ (must create new consent)
```

**GDPR Compliance:**
When user revokes consent:
1. Update consent status to Revoked
2. Trigger data deletion/anonymization process
3. Log revocation for audit trail

**Use Cases:**
- User exercises GDPR right to withdraw consent
- Admin revokes consent on user's behalf
- System automatically revokes on account deletion

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": "consent-uuid",
  "status": "Revoked",
  "revoked_reason": "User requested GDPR data deletion"
}' localhost:50053 consent.ConsentService/UpdateConsentStatus
```

---

### 6. DeleteConsent
**RPC:** `consent.ConsentService/DeleteConsent`

**Purpose:** Soft delete consent record (set is_deleted = true)

**Request:**
```json
{
  "id": "consent-uuid"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Consent deleted successfully"
}
```

**Business Logic:**
1. Find consent by id
2. Set is_deleted = TRUE
3. Set updated_at = NOW()
4. Consent no longer appears in queries

**Important:**
- Soft delete: Data remains in database
- Use UpdateConsentStatus(Revoked) for GDPR withdrawals
- Use DeleteConsent for correcting mistakes or data cleanup

**Use Cases:**
- Admin corrects duplicate consent record
- Clean up test data
- Remove erroneous entries

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": "consent-uuid"
}' localhost:50053 consent.ConsentService/DeleteConsent
```

---

### 7. CheckUserConsent
**RPC:** `consent.ConsentService/CheckUserConsent`

**Purpose:** Verify if user has active consent for specific document (by title)

**Request:**
```json
{
  "user_id": "user-uuid",
  "document_title": "Privacy Policy"
}
```

**Response:**
```json
{
  "has_consent": true,
  "consent": {
    "id": "consent-uuid",
    "user_id": "user-uuid",
    "document_id": "privacy-policy-v3-uuid",
    "version": 3,
    "status": "Accepted",
    "consented_at": "timestamp",
    // ... other fields
  },
  "is_latest_version": true
}
```

**Business Logic:**
1. **Call Document Service:** Get current published document by title
2. **Query consent:** WHERE user_id = ? AND document_id = current_document.id
3. **Check version:** Compare consent.version with current_document.version
4. **Return result:**
   - has_consent: true if consent exists and status = Accepted
   - consent: The consent record (if exists)
   - is_latest_version: true if consent.version == current_document.version

**Use Cases:**
- **Before feature access:** Check if user accepted terms
- **Show consent modal:** If user hasn't consented or version outdated
- **Compliance gate:** Block access until consent given

**Frontend Integration:**
```javascript
// Check if user accepted latest privacy policy
const response = await consentService.checkUserConsent({
  user_id: currentUserId,
  document_title: "Privacy Policy"
});

if (!response.has_consent) {
  // User never consented
  showConsentModal("Please accept our Privacy Policy");
} else if (!response.is_latest_version) {
  // User consented to old version
  showUpdateModal("Privacy Policy has been updated. Please review.");
} else {
  // User has valid consent, allow access
  proceedToFeature();
}
```

**Important:** This is the most commonly used method for consent verification.

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid",
  "document_title": "Privacy Policy"
}' localhost:50053 consent.ConsentService/CheckUserConsent
```

---

### 8. GetConsentStats (Admin)
**RPC:** `consent.ConsentService/GetConsentStats`

**Purpose:** Get consent statistics for admin dashboard

**Request:**
```json
{
  "document_id": "privacy-policy-v3-uuid"
}
```

**Response:**
```json
{
  "total_consents": 500,
  "accepted_count": 450,
  "rejected_count": 20,
  "revoked_count": 30,
  "acceptance_rate": 90.0,
  "latest_consent_at": "1733745000"
}
```

**Business Logic:**
1. COUNT all consents for document_id
2. COUNT by status (Accepted, Rejected, Revoked)
3. Calculate acceptance rate = (accepted / total) * 100
4. Get MAX(consented_at) for latest consent

**Use Cases:**
- Admin dashboard analytics
- Compliance reporting
- Track policy acceptance rates
- Monitor consent trends

**Dashboard Display:**
```javascript
const stats = await consentService.getConsentStats({
  document_id: selectedPolicyId
});

displayChart({
  title: "Privacy Policy v3 Acceptance",
  accepted: stats.accepted_count,
  rejected: stats.rejected_count,
  revoked: stats.revoked_count,
  rate: stats.acceptance_rate + "%"
});
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "document_id": "document-uuid"
}' localhost:50053 consent.ConsentService/GetConsentStats
```

---

## Service Dependencies

### Document Service Integration

Consent Service **depends on** Document Service to:
1. **Verify documents exist** before creating consent
2. **Extract version number** from document
3. **Check latest version** for consent validation

**gRPC Client Setup:**
```go
// In consent service main.go
documentConn, err := grpc.Dial(
    cfg.DocumentServiceURL,
    grpc.WithInsecure(),
)
documentClient := documentpb.NewDocumentServiceClient(documentConn)

// Pass to service layer
consentService := service.NewConsentService(
    consentRepo,
    documentClient, // Inject document client
)
```

**Example: Get Document Version:**
```go
// In consent service business logic
func (s *consentService) CreateConsent(ctx context.Context, userID, documentID, status string) (*domain.Consent, error) {
    // Call Document Service
    docResponse, err := s.documentClient.GetDocumentByID(ctx, &documentpb.GetDocumentByIDRequest{
        Id: documentID,
    })
    if err != nil {
        return nil, fmt.Errorf("document not found: %w", err)
    }

    // Extract version
    version := docResponse.Document.Version

    // Create consent with version
    consent := &domain.Consent{
        UserID:      userID,
        DocumentID:  documentID,
        Version:     version,
        Status:      status,
        ConsentedAt: time.Now(),
    }

    return s.repo.Create(ctx, consent)
}
```

## Complete Consent Flow

### Scenario 1: New User Signup

```
1. User fills signup form
2. Frontend displays Privacy Policy modal
3. User clicks "I Accept"

Frontend:
POST /api/auth/register {
  phone: "0912345678",
  password: "Pass123",
  privacy_policy_accepted: true
}

Gateway:
1. Call UserService.Register → Get user_id
2. Call DocumentService.GetDocumentByTitle("Privacy Policy") → Get document_id
3. Call ConsentService.CreateConsent(user_id, document_id, "Accepted")
4. Return success to frontend

Result:
✅ User created
✅ Consent recorded with version
✅ User can access app
```

### Scenario 2: Policy Update - User Must Re-consent

```
1. Admin publishes Privacy Policy v4
2. User logs in (still has consent for v3)
3. System checks consent version

Backend:
response = ConsentService.CheckUserConsent(user_id, "Privacy Policy")
if !response.is_latest_version:
    return {
        require_consent: true,
        document: /* Privacy Policy v4 */,
        old_version: 3,
        new_version: 4
    }

Frontend:
Display modal: "Privacy Policy has been updated. Please review."
Show document content with Accept/Reject buttons

User clicks Accept:
→ ConsentService.CreateConsent(user_id, document_v4_id, "Accepted")

Result:
✅ New consent created for v4
✅ Old consent for v3 remains in history
✅ User can continue using app
```

### Scenario 3: GDPR - User Revokes Consent

```
User Profile → Privacy Settings → "Withdraw Data Processing Consent"

Frontend:
1. Show warning: "This will delete your data"
2. User confirms

Backend:
1. Get consent_id from database
2. ConsentService.UpdateConsentStatus(consent_id, "Revoked", "User requested data deletion")
3. Trigger data deletion job (separate service)

Result:
✅ Consent status = Revoked
✅ revoked_at = NOW()
✅ revoked_reason = "User requested data deletion"
✅ Data deletion process initiated
```

### Scenario 4: Feature Gating by Consent

```
User tries to access Premium Features

Backend:
response = ConsentService.CheckUserConsent(user_id, "Premium Terms")
if !response.has_consent:
    return 403 Forbidden {
        error: "Must accept Premium Terms",
        document_id: "premium-terms-uuid"
    }

Frontend:
Display Premium Terms modal
User accepts → CreateConsent → Feature unlocked

Result:
✅ Feature access controlled by consent
✅ Compliance with terms requirements
✅ Audit trail of user agreements
```

## Integration Guide

### For Frontend Developers

**1. Check if User Needs to Consent:**
```javascript
async function checkConsentRequired() {
  const response = await consentService.checkUserConsent({
    user_id: currentUserId,
    document_title: "Privacy Policy"
  });

  if (!response.has_consent) {
    // Never consented
    showConsentModal("new");
  } else if (!response.is_latest_version) {
    // Outdated consent
    showConsentModal("update", {
      oldVersion: response.consent.version,
      currentConsent: response.consent
    });
  } else {
    // Valid consent
    return true;
  }
}
```

**2. Record User Consent:**
```javascript
async function acceptPolicy(documentId) {
  try {
    const response = await consentService.createConsent({
      user_id: currentUserId,
      document_id: documentId,
      status: "Accepted"
    });

    console.log("Consent recorded:", response.consent.id);
    hideModal();
    proceedToApp();
  } catch (error) {
    if (error.code === "ALREADY_EXISTS") {
      alert("You have already accepted this policy");
    }
  }
}
```

**3. Display User's Consent History:**
```javascript
async function loadConsentHistory() {
  const response = await consentService.getUserConsents({
    user_id: currentUserId
  });

  const consentList = response.consents.map(consent => ({
    documentId: consent.document_id,
    version: consent.version,
    status: consent.status,
    acceptedDate: new Date(consent.consented_at * 1000),
    revokedDate: consent.revoked_at ? new Date(consent.revoked_at * 1000) : null
  }));

  displayConsentTable(consentList);
}
```

**4. Revoke Consent (GDPR):**
```javascript
async function revokeConsent(consentId) {
  const confirmed = confirm(
    "Are you sure you want to withdraw your consent? This may result in data deletion."
  );

  if (confirmed) {
    await consentService.updateConsentStatus({
      id: consentId,
      status: "Revoked",
      revoked_reason: "User requested data deletion via profile settings"
    });

    alert("Consent has been revoked. Data deletion process initiated.");
    reloadConsentHistory();
  }
}
```

### For Backend/Gateway Developers

**Consent Verification Middleware:**
```go
func RequireConsent(documentTitle string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("user_id") // From JWT

        // Check consent
        response, err := consentClient.CheckUserConsent(ctx, &consent.CheckUserConsentRequest{
            UserId:        userID,
            DocumentTitle: documentTitle,
        })

        if err != nil || !response.HasConsent {
            c.JSON(403, gin.H{
                "error": "Consent required",
                "document_title": documentTitle,
                "message": "Please accept the required policy document",
            })
            c.Abort()
            return
        }

        if !response.IsLatestVersion {
            c.JSON(409, gin.H{
                "error": "Consent outdated",
                "document_title": documentTitle,
                "current_version": response.Consent.Version,
                "message": "Policy has been updated. Please review and accept.",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// Usage
router.GET("/premium/features", 
    authMiddleware,
    RequireConsent("Premium Terms"),
    premiumFeaturesHandler)
```

## Testing

### List All Available Methods
```bash
grpcurl -plaintext localhost:50053 list consent.ConsentService
```

### Test Complete Flow
```bash
# Prerequisites: Document Service must be running
# 1. Get a document ID from Document Service
DOC_ID=$(grpcurl -plaintext -d '{"title": "Privacy Policy"}' \
  localhost:50051 document.DocumentService/GetDocumentByTitle | jq -r '.document.id')

echo "Document ID: $DOC_ID"

# 2. Create consent
CONSENT=$(grpcurl -plaintext -d "{
  \"user_id\": \"test-user-uuid\",
  \"document_id\": \"$DOC_ID\",
  \"status\": \"Accepted\"
}" localhost:50053 consent.ConsentService/CreateConsent)

CONSENT_ID=$(echo $CONSENT | jq -r '.consent.id')
echo "Consent ID: $CONSENT_ID"

# 3. Get consent by ID
grpcurl -plaintext -d "{\"id\": \"$CONSENT_ID\"}" \
  localhost:50053 consent.ConsentService/GetConsentByID

# 4. Get user's all consents
grpcurl -plaintext -d '{"user_id": "test-user-uuid"}' \
  localhost:50053 consent.ConsentService/GetUserConsents

# 5. Check if user has valid consent
grpcurl -plaintext -d '{
  "user_id": "test-user-uuid",
  "document_title": "Privacy Policy"
}' localhost:50053 consent.ConsentService/CheckUserConsent

# 6. Update consent status (revoke)
grpcurl -plaintext -d "{
  \"id\": \"$CONSENT_ID\",
  \"status\": \"Revoked\",
  \"revoked_reason\": \"Test revocation\"
}" localhost:50053 consent.ConsentService/UpdateConsentStatus

# 7. Verify revocation
grpcurl -plaintext -d "{\"id\": \"$CONSENT_ID\"}" \
  localhost:50053 consent.ConsentService/GetConsentByID
```

## Configuration

### Environment Variables
```bash
# Database
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/consent_db?sslmode=disable"

# Service Dependencies
DOCUMENT_SERVICE_URL="localhost:50051"

# Server
SERVER_PORT="50053"
```

## Troubleshooting

### Service won't start
```bash
# Check PostgreSQL
docker-compose ps policy_postgres

# Check Document Service is running
grpcurl -plaintext localhost:50051 list

# Check migrations
docker-compose logs consent_migrate

# Check service logs
docker-compose logs consent_service
```

### "Document not found" error
```bash
# Verify Document Service connection
docker network inspect policy-system_default

# Check document exists
grpcurl -plaintext -d '{"id": "your-doc-id"}' \
  localhost:50051 document.DocumentService/GetDocumentByID

# Check docker-compose network settings
# Consent Service should use: document_service:50051 (not localhost)
```

### Duplicate consent error
```bash
# Check existing consent
docker exec -it policy_postgres psql -U postgres -d consent_db -c \
  "SELECT * FROM user_consents WHERE user_id = 'user-uuid' AND document_id = 'doc-uuid';"

# If duplicate exists and should be updated:
# 1. Revoke old consent via UpdateConsentStatus
# 2. Create new consent
```

### Version mismatch issues
```bash
# Check document version
grpcurl -plaintext -d '{"id": "doc-id"}' \
  localhost:50051 document.DocumentService/GetDocumentByID

# Check consent version
docker exec -it policy_postgres psql -U postgres -d consent_db -c \
  "SELECT id, user_id, version, status FROM user_consents WHERE document_id = 'doc-id';"
```

## Performance

### Database Indexes
```sql
-- Already created in migrations
CREATE INDEX idx_user_consents_user_id ON user_consents(user_id);
CREATE INDEX idx_user_consents_document_id ON user_consents(document_id);
CREATE INDEX idx_user_consents_status ON user_consents(status);
CREATE UNIQUE INDEX idx_user_consents_user_document ON user_consents(user_id, document_id) 
  WHERE is_deleted = FALSE;
```

### Optimization Tips
- **Cache CheckUserConsent results** for 5-10 minutes (per user)
- **Batch consent checks** if validating multiple documents
- **Use connection pooling** for Document Service gRPC client

## Security & Compliance

### GDPR Compliance Features
1. **Right to Withdraw:** UpdateConsentStatus(Revoked)
2. **Audit Trail:** All consents timestamped, revocation reasons stored
3. **Version Tracking:** User knows which version they agreed to
4. **Data Export:** GetUserConsents for user data export

### Best Practices
1. **Always record consent:** Never assume implied consent
2. **Version tracking:** Store document version with consent
3. **Revocation reasons:** Document why consent was revoked
4. **Idempotent operations:** Use same request ID for retries
5. **Document verification:** Always verify documents exist before consent

---

## 🆕 New Features (v1.0.0)

### GetConsentHistory
Get complete consent history for user+document combination

```bash
grpcurl -plaintext -d '{
  "user_id": "user-123",
  "document_id": "privacy-policy"
}' localhost:50053 consent.ConsentService/GetConsentHistory
```

**Response:**
```json
{
  "history": [
    {
      "id": "consent-3",
      "version_timestamp": 1733900000,
      "is_latest": true,
      "agreed_at": 1733900000
    },
    {
      "id": "consent-2",
      "version_timestamp": 1733800000,
      "is_latest": false,
      "revoked_at": 1733900000,
      "revoked_reason": "new_version"
    },
    {
      "id": "consent-1",
      "version_timestamp": 1733700000,
      "is_latest": false
    }
  ],
  "total": 3
}
```

### GetConsentStats
Get aggregated statistics about consents

```bash
# All platforms
grpcurl -plaintext -d '{}' localhost:50053 consent.ConsentService/GetConsentStats

# Specific platform
grpcurl -plaintext -d '{
  "platform": "Client"
}' localhost:50053 consent.ConsentService/GetConsentStats
```

**Response:**
```json
{
  "total_consents": 150,
  "active_consents": 120,
  "revoked_consents": 10,
  "consents_by_document": {
    "Privacy Policy": 50,
    "Terms of Service": 50,
    "Cookie Policy": 50
  },
  "consents_by_platform": {
    "Client": 100,
    "Merchant": 50
  },
  "consents_by_method": {
    "REGISTRATION": 80,
    "UI": 60,
    "API": 10
  }
}
```

### Document Verification
All RecordConsent calls now verify documents exist in Document Service

```bash
# This will fail if document doesn't exist or version mismatch
grpcurl -plaintext -d '{
  "user_id": "user-123",
  "platform": "Client",
  "consents": [{
    "document_id": "doc-1",
    "document_name": "Privacy Policy",
    "version_timestamp": 1733900000
  }],
  "consent_method": "UI"
}' localhost:50053 consent.ConsentService/RecordConsent
```

### Idempotent Operations
Recording same consent twice returns existing consent, not error

```bash
# First call - creates consent
grpcurl -plaintext -d '{...}' localhost:50053 consent.ConsentService/RecordConsent

# Second call - returns same consent, no duplicate
grpcurl -plaintext -d '{...}' localhost:50053 consent.ConsentService/RecordConsent
```
4. **Regular audits:** Monitor consent stats, identify outliers
5. **Clear communication:** Show users what they're consenting to

## Related Services

- **Document Service:** Source of policy documents (port 50051)
- **User Service:** Provides user_id from authentication (port 50052)
- **Gateway Service:** Exposes REST API for frontend

## Support

For issues or questions:
1. Check logs: `docker-compose logs -f consent_service`
2. Check database: `docker exec -it policy_postgres psql -U postgres -d consent_db`
3. Verify Document Service: `grpcurl -plaintext localhost:50051 list`
4. Test consent flow: See "Test Complete Flow" section above
