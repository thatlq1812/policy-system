# Consent Service

User consent management service for tracking policy document acceptances using gRPC.

## Overview

**Port:** 50053  
**Protocol:** gRPC  
**Database:** PostgreSQL (consent_db)  
**Language:** Go 1.21+

## Architecture

```
Client/Gateway
    ↓
Consent Service (gRPC)
    ↓
PostgreSQL (consent_db)
    └── user_consents table
    
    → Calls Document Service (to verify documents)
```

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 15+
- Document Service running (port 50051)

### Run with Docker
```bash
# From project root
docker-compose up -d consent_service

# Check logs
docker-compose logs -f consent_service

# Run migrations
docker-compose up consent_migrate
```

### Run Locally
```bash
cd consent

# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/consent_db?sslmode=disable"
export DOCUMENT_SERVICE_URL="localhost:50051"
export SERVER_PORT="50053"

# Run service
go run cmd/server/main.go
```

## Database Schema

### user_consents
```sql
id              UUID PRIMARY KEY
user_id         UUID NOT NULL           -- From User Service
document_id     UUID NOT NULL           -- From Document Service
version         INTEGER NOT NULL
status          VARCHAR(50) NOT NULL    -- Accepted | Rejected | Revoked
consented_at    TIMESTAMP NOT NULL
revoked_at      TIMESTAMP
revoked_reason  VARCHAR(255)
created_at      TIMESTAMP
updated_at      TIMESTAMP
is_deleted      BOOLEAN DEFAULT FALSE

-- Unique constraint: one active consent per user per document
UNIQUE(user_id, document_id) WHERE is_deleted = FALSE
```

**Status Values:**
- `Accepted`: User agreed to the policy
- `Rejected`: User declined the policy
- `Revoked`: User withdrew consent (GDPR compliance)

## API Methods

### 1. CreateConsent
**RPC:** `consent.ConsentService/CreateConsent`

**Purpose:** Record user's acceptance of a policy document

**Request:**
```json
{
  "user_id": "user-uuid",
  "document_id": "document-uuid",
  "status": "Accepted"
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
    "status": "Accepted",
    "consented_at": "1733740800",
    "revoked_at": null,
    "revoked_reason": null,
    "created_at": "1733740800",
    "updated_at": "1733740800"
  }
}
```

**Business Logic:**
1. **Verify document exists** (call Document Service by document_id)
2. **Extract version** from document
3. Check if user already has consent for this document
4. If exists and active → Return error (duplicate)
5. If exists but revoked → Create new consent
6. Create consent record with status and version
7. Set consented_at = NOW()

**Important:** Version is automatically extracted from the document. This ensures consent is tied to specific version.

**Use Cases:**
- User accepts terms during signup
- User accepts updated privacy policy
- User consents to data processing

**Errors:**
- `ALREADY_EXISTS`: User already consented to this document
- `NOT_FOUND`: Document not found
- `INVALID_ARGUMENT`: Invalid status or missing fields

**Example:**
```bash
grpcurl -plaintext -d '{
  "user_id": "user-uuid",
  "document_id": "privacy-policy-v3-uuid",
  "status": "Accepted"
}' localhost:50053 consent.ConsentService/CreateConsent
```

---

### 2. GetConsentByID
**RPC:** `consent.ConsentService/GetConsentByID`

**Purpose:** Get specific consent record by UUID

**Request:**
```json
{
  "id": "consent-uuid"
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
    "status": "Accepted",
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
