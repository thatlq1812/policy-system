# Document Service - Policy Document Management

Policy document management service for creating, versioning, and publishing documents.

**Version:** 1.0.0 | **Status:** Production Ready | **Port:** 50051 (gRPC) | **Methods:** 4/4

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Environment Variables](#environment-variables)
4. [Database Schema](#database-schema)
5. [API Reference](#api-reference)
6. [Testing Guide](#testing-guide)
7. [Troubleshooting](#troubleshooting)

---

## Overview

### Features

- Document creation and versioning
- Document lifecycle management (Draft, Published, Archived)
- Retrieval of latest policy by platform
- Comprehensive policy history tracking

### Architecture

```
Client/Gateway
    |
Document Service (gRPC)
    |-- Service Layer (Business Logic)
    |-- Repository Layer (Data Access)
    +-- PostgreSQL Database
        +-- policy_documents table
```

---

## Quick Start

### Run with Docker (Recommended)
```bash
# From project root d:/w2
docker-compose up -d document_service

# Check logs
docker-compose logs -f document_service

# Run migrations
docker-compose up document_migrate

# Verify service
grpcurl -plaintext localhost:50051 list document.DocumentService
```

### Run Locally (Development)
```bash
cd document

# Install dependencies
go mod download

# Set environment variables (example .env)
# DATABASE_URL="postgresql://postgres:postgres@localhost:5432/document_db?sslmode=disable"
# GRPC_PORT="50051"

# Run service
go run cmd/server/main.go

# Or build binary
go build -o bin/document cmd/server/main.go
./bin/document
```

---

## Environment Variables
| Variable       | Description                          | Default | Required |
|----------------|--------------------------------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string         | -       | Yes      |
| `GRPC_PORT`    | gRPC server port                     | `50051` | No       |

---

## Database Schema

### policy_documents table
```sql
CREATE TABLE policy_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_name VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL, -- Client | Merchant
    is_mandatory BOOLEAN NOT NULL DEFAULT FALSE,
    effective_timestamp BIGINT NOT NULL,
    content_html TEXT NOT NULL,
    file_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP,
    deleted_by VARCHAR(255),
    is_latest BOOLEAN NOT NULL DEFAULT TRUE -- Indicates if this is the latest published version
);

-- Unique index for active, latest policy documents
CREATE UNIQUE INDEX idx_latest_active_policy 
ON policy_documents (platform, document_name)
WHERE is_deleted = FALSE AND is_latest = TRUE;

-- Index for efficient history retrieval
CREATE INDEX idx_policy_history 
ON policy_documents (platform, document_name, effective_timestamp DESC);
```

**Migrations:**
- `000001_create_policy_documents_table.up.sql`

---

## API Reference

### Available Methods (4 Total)

**Core Operations:**
```
document.DocumentService.CreatePolicy           - Create a new policy document
document.DocumentService.GetLatestPolicyByPlatform  - Retrieve the latest published policy for a platform
document.DocumentService.UpdatePolicy           - Create a new version of an existing policy
document.DocumentService.GetPolicyHistory       - Retrieve all versions of a policy
```

---

## Testing Guide

### Using grpcurl

To test the Document Service endpoints, ensure the service is running and use `grpcurl` from your terminal.

**Example: CreatePolicy**
```bash
grpcurl -plaintext -d '{
  "document_name": "Privacy Policy",
  "platform": "Client",
  "is_mandatory": true,
  "effective_timestamp": 1678886400,
  "content_html": "<p>New Privacy Policy Content</p>",
  "created_by": "admin-user"
}' localhost:50051 document.DocumentService/CreatePolicy
```

**Example: GetLatestPolicyByPlatform**
```bash
grpcurl -plaintext -d '{
  "platform": "Client",
  "document_name": "Privacy Policy"
}' localhost:50051 document.DocumentService/GetLatestPolicyByPlatform
```

**Example: UpdatePolicy**
```bash
grpcurl -plaintext -d '{
  "document_name": "Privacy Policy",
  "platform": "Client",
  "is_mandatory": true,
  "effective_timestamp": 1700000000,
  "content_html": "<p>Updated Privacy Policy Content</p>",
  "created_by": "admin-user"
}' localhost:50051 document.DocumentService/UpdatePolicy
```

**Example: GetPolicyHistory**
```bash
grpcurl -plaintext -d '{
  "platform": "Client",
  "document_name": "Privacy Policy"
}' localhost:50051 document.DocumentService/GetPolicyHistory
```

---

## Troubleshooting

- **Service not starting:** Check `docker-compose logs document_service` for errors.
- **Database connection issues:** Verify `DATABASE_URL` and PostgreSQL container status.
- **Migration errors:** Ensure migrations are applied correctly with `docker-compose up document_migrate`.

**Response:**
```json
{
  "document": {
    "id": "latest-version-uuid",
    "title": "Privacy Policy",
    "content": "Current version content...",
    "version": 3,
    "effective_date": "2025-03-01",
    "status": "Published",
    "owner": "admin-uuid",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
}
```

**Business Logic:**
1. Query WHERE title = ? AND status = 'Published' AND is_deleted = FALSE
2. ORDER BY version DESC LIMIT 1
3. Return latest published version

**Important:** Only returns Published documents. Drafts are not returned.

**Use Cases:**
- Display current policy to users
- Show active terms and conditions during signup
- Consent Service fetches current document for user acceptance

**Frontend Integration:**
```javascript
// Display privacy policy to user
const doc = await documentService.getDocumentByTitle({
  title: "Privacy Policy"
});
showModal(doc.content);
```

**Example:**
```bash
grpcurl -plaintext -d '{
  "title": "Privacy Policy"
}' localhost:50051 document.DocumentService/GetDocumentByTitle
```

---

### 4. ListDocuments
**RPC:** `document.DocumentService/ListDocuments`

**Purpose:** Get all documents (latest version of each title)

**Request:**
```json
{
  "status": "Published",
  "include_deleted": false
}
```

**Response:**
```json
{
  "documents": [
    {
      "id": "uuid-1",
      "title": "Privacy Policy",
      "content": "...",
      "version": 3,
      "status": "Published",
      // ... other fields
    },
    {
      "id": "uuid-2",
      "title": "Terms of Service",
      "content": "...",
      "version": 2,
      "status": "Published",
      // ... other fields
    }
  ],
  "total_count": 2
}
```

**Business Logic:**
1. Group by title
2. Get MAX(version) for each title
3. Filter by status (optional)
4. Filter by is_deleted (optional)
5. Return all matching documents

**Query Parameters:**
- `status`: Filter by status (Draft, Published, Archived) - optional
- `include_deleted`: Include soft-deleted documents - default false

**Use Cases:**
- Admin dashboard showing all policies
- User viewing available policies
- System listing documents for consent flow

**Example:**
```bash
# Get all published documents
grpcurl -plaintext -d '{
  "status": "Published"
}' localhost:50051 document.DocumentService/ListDocuments

# Get all documents including drafts
grpcurl -plaintext -d '{}' localhost:50051 document.DocumentService/ListDocuments
```

---

### 5. ListDocumentVersions
**RPC:** `document.DocumentService/ListDocumentVersions`

**Purpose:** Get version history of a document by title

**Request:**
```json
{
  "title": "Privacy Policy"
}
```

**Response:**
```json
{
  "documents": [
    {
      "id": "uuid-v3",
      "title": "Privacy Policy",
      "version": 3,
      "status": "Published",
      "effective_date": "2025-03-01",
      // ... other fields
    },
    {
      "id": "uuid-v2",
      "title": "Privacy Policy",
      "version": 2,
      "status": "Archived",
      "effective_date": "2025-02-01",
      // ... other fields
    },
    {
      "id": "uuid-v1",
      "title": "Privacy Policy",
      "version": 1,
      "status": "Archived",
      "effective_date": "2025-01-01",
      // ... other fields
    }
  ],
  "total_count": 3
}
```

**Business Logic:**
1. Query WHERE title = ? AND is_deleted = FALSE
2. ORDER BY version DESC
3. Return all versions

**Use Cases:**
- Admin reviews change history
- Audit trail for policy updates
- User views previous versions they consented to
- Legal compliance (track all changes)

**Example:**
```bash
grpcurl -plaintext -d '{
  "title": "Privacy Policy"
}' localhost:50051 document.DocumentService/ListDocumentVersions
```

---

### 6. UpdateDocument
**RPC:** `document.DocumentService/UpdateDocument`

**Purpose:** Update existing document (creates new version if published)

**Request:**
```json
{
  "id": "doc-uuid",
  "title": "Privacy Policy",
  "content": "Updated content...",
  "effective_date": "2025-04-01"
}
```

**Response:**
```json
{
  "document": {
    "id": "new-uuid",
    "title": "Privacy Policy",
    "content": "Updated content...",
    "version": 4,
    "effective_date": "2025-04-01",
    "status": "Draft",
    // ... other fields
  }
}
```

**Business Logic - Two Scenarios:**

**Scenario 1: Update Draft**
```
IF current status = Draft:
  → Update in place (same id, same version)
  → Change: content, effective_date
  → Status remains Draft
```

**Scenario 2: Update Published/Archived**
```
IF current status = Published OR Archived:
  → Create NEW document
  → New id (UUID)
  → version = current_version + 1
  → status = Draft
  → Old document remains unchanged
```

**Important Rules:**
1. Cannot update Published document directly (creates new version)
2. New version starts as Draft (must explicitly publish)
3. Old versions are never modified (immutable after publish)
4. All versions share same title (uniqueness constraint)

**Use Cases:**
- Admin edits draft before publishing
- Admin creates new version of published policy
- Compliance updates to terms and conditions

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": "doc-uuid",
  "title": "Privacy Policy",
  "content": "Updated privacy policy content...",
  "effective_date": "2025-04-01"
}' localhost:50051 document.DocumentService/UpdateDocument
```

---

### 7. DeleteDocument
**RPC:** `document.DocumentService/DeleteDocument`

**Purpose:** Soft delete document (set is_deleted = true)

**Request:**
```json
{
  "id": "doc-uuid"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Document deleted successfully"
}
```

**Business Logic:**
1. Find document by id
2. Set is_deleted = TRUE
3. Document no longer appears in listings
4. Data remains in database (soft delete)

**Important Notes:**
- Only deletes specific version (by id)
- Other versions of same title remain
- Deleted documents can be restored by setting is_deleted = FALSE

**Use Cases:**
- Admin removes incorrect version
- Clean up old drafts
- Compliance: Hide outdated policies

**Warning:** Be careful deleting Published documents. Users may have active consents linked to them.

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": "doc-uuid-to-delete"
}' localhost:50051 document.DocumentService/DeleteDocument
```

---

### 8. PublishDocument
**RPC:** `document.DocumentService/PublishDocument`

**Purpose:** Change document status from Draft to Published

**Request:**
```json
{
  "id": "doc-uuid"
}
```

**Response:**
```json
{
  "document": {
    "id": "doc-uuid",
    "title": "Privacy Policy",
    "version": 4,
    "status": "Published",
    // ... other fields
  }
}
```

**Business Logic:**
1. Get document by id
2. Check current status = Draft (only drafts can be published)
3. Update status = 'Published'
4. Set updated_at = NOW()

**State Transitions:**
```
Draft → Published ✅ (PublishDocument)
Published → Archived ✅ (ArchiveDocument)
Published → Draft ❌ (not allowed)
Archived → Published ❌ (not allowed)
```

**Use Cases:**
- Admin approves draft and makes it active
- New policy version goes live
- System activates updated terms

**Important:** Once published, document becomes visible to users through GetDocumentByTitle and ListDocuments.

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": "draft-doc-uuid"
}' localhost:50051 document.DocumentService/PublishDocument
```

---

### 9. ArchiveDocument
**RPC:** `document.DocumentService/ArchiveDocument`

**Purpose:** Change document status from Published to Archived

**Request:**
```json
{
  "id": "doc-uuid"
}
```

**Response:**
```json
{
  "document": {
    "id": "doc-uuid",
    "title": "Privacy Policy",
    "version": 3,
    "status": "Archived",
    // ... other fields
  }
}
```

**Business Logic:**
1. Get document by id
2. Check current status = Published
3. Update status = 'Archived'
4. Set updated_at = NOW()

**Use Cases:**
- Newer version is published, old version archived
- Policy no longer active but kept for records
- Compliance: Maintain history of all policies

**Automatic Archiving:**
When a new version is published, you may want to automatically archive the previous published version:

```go
// In PublishDocument method:
1. Set new version status = Published
2. Find previous Published version (same title, version - 1)
3. If found → Set status = Archived
```

**Note:** Archived documents are still accessible by id and appear in version history.

**Example:**
```bash
grpcurl -plaintext -d '{
  "id": "old-published-doc-uuid"
}' localhost:50051 document.DocumentService/ArchiveDocument
```

---

## Document Lifecycle

### Complete Workflow

```
1. CREATE
   ↓
[Draft v1]
   ↓ UpdateDocument (edit content)
[Draft v1] (updated)
   ↓ PublishDocument
[Published v1] ← GetDocumentByTitle returns this
   ↓ UpdateDocument (create new version)
[Published v1] + [Draft v2]
   ↓ PublishDocument(v2)
[Archived v1] + [Published v2] ← GetDocumentByTitle now returns v2
   ↓ UpdateDocument (create new version)
[Archived v1] + [Published v2] + [Draft v3]
   ↓ PublishDocument(v3)
[Archived v1] + [Archived v2] + [Published v3] ← Latest
```

### Status Flow Diagram

```
        CreateDocument
             ↓
        [Draft v1]
             ↓ PublishDocument
        [Published v1]
             ↓ UpdateDocument → [Draft v2]
             ↓ PublishDocument(v2)
        [Archived v1]
             
        [Published v2]
             ↓ UpdateDocument → [Draft v3]
             ↓ ArchiveDocument OR
             ↓ PublishDocument(v3)
        [Archived v2]
             
        [Published v3]
```

## Integration Guide

### For Frontend Developers

**1. Display Current Policy to User:**
```javascript
// Get current published version
const response = await documentService.getDocumentByTitle({
  title: "Privacy Policy"
});

// Display in modal or page
showPolicyModal({
  title: response.document.title,
  content: response.document.content,
  version: response.document.version,
  effectiveDate: response.document.effective_date
});
```

**2. Admin Dashboard - List All Policies:**
```javascript
// Get all published documents
const response = await documentService.listDocuments({
  status: "Published"
});

// Display in table
response.documents.forEach(doc => {
  addToTable({
    title: doc.title,
    version: doc.version,
    effectiveDate: doc.effective_date,
    status: doc.status
  });
});
```

**3. Admin - View Version History:**
```javascript
// Get all versions of a document
const response = await documentService.listDocumentVersions({
  title: "Privacy Policy"
});

// Display timeline
response.documents.forEach(version => {
  addToTimeline({
    version: version.version,
    status: version.status,
    effectiveDate: version.effective_date,
    publishedAt: version.updated_at
  });
});
```

**4. Admin - Create New Policy:**
```javascript
// Step 1: Create draft
const createResponse = await documentService.createDocument({
  title: "Cookie Policy",
  content: "We use cookies...",
  effective_date: "2025-06-01",
  owner: currentUserId
});

// Step 2: Review and publish
const publishResponse = await documentService.publishDocument({
  id: createResponse.document.id
});

console.log("Policy published:", publishResponse.document.status);
```

**5. Admin - Update Existing Policy:**
```javascript
// Step 1: Get current document
const current = await documentService.getDocumentByTitle({
  title: "Privacy Policy"
});

// Step 2: Update (creates new draft version)
const updated = await documentService.updateDocument({
  id: current.document.id,
  title: current.document.title,
  content: "Updated privacy policy content...",
  effective_date: "2025-07-01"
});

// updated.document.version = current.version + 1
// updated.document.status = "Draft"

// Step 3: Publish when ready
await documentService.publishDocument({
  id: updated.document.id
});
```

### For Consent Service Integration

**Get Current Document for Consent:**
```go
// In Consent Service
func (s *consentService) CreateConsent(userID string, documentTitle string) error {
    // Get current published version
    doc, err := s.documentClient.GetDocumentByTitle(ctx, &document.GetDocumentByTitleRequest{
        Title: documentTitle,
    })
    if err != nil {
        return fmt.Errorf("document not found: %w", err)
    }

    // Create consent with specific document id
    consent := &domain.Consent{
        UserID:     userID,
        DocumentID: doc.Document.Id,
        Version:    doc.Document.Version,
        Status:     "Accepted",
    }

    return s.repo.Create(ctx, consent)
}
```

**Verify User Accepted Latest Version:**
```go
func (s *consentService) IsConsentUpToDate(userID, documentTitle string) (bool, error) {
    // Get user's consent
    userConsent, err := s.repo.GetByUserAndTitle(ctx, userID, documentTitle)
    if err != nil {
        return false, err
    }

    // Get current published document
    currentDoc, err := s.documentClient.GetDocumentByTitle(ctx, &document.GetDocumentByTitleRequest{
        Title: documentTitle,
    })
    if err != nil {
        return false, err
    }

    // Compare versions
    return userConsent.Version >= currentDoc.Document.Version, nil
}
```

## Testing

### List All Available Methods
```bash
grpcurl -plaintext localhost:50051 list document.DocumentService
```

### Test Complete Flow
```bash
# 1. Create document (Draft v1)
CREATE=$(grpcurl -plaintext -d '{
  "title": "Test Policy",
  "content": "Initial content",
  "effective_date": "2025-01-01",
  "owner": "admin"
}' localhost:50051 document.DocumentService/CreateDocument)

DOC_ID=$(echo $CREATE | jq -r '.document.id')
echo "Created document: $DOC_ID"

# 2. Get by ID
grpcurl -plaintext -d "{\"id\": \"$DOC_ID\"}" \
  localhost:50051 document.DocumentService/GetDocumentByID

# 3. Publish document
grpcurl -plaintext -d "{\"id\": \"$DOC_ID\"}" \
  localhost:50051 document.DocumentService/PublishDocument

# 4. Get by title (returns published version)
grpcurl -plaintext -d '{"title": "Test Policy"}' \
  localhost:50051 document.DocumentService/GetDocumentByTitle

# 5. Update document (creates Draft v2)
UPDATE=$(grpcurl -plaintext -d "{
  \"id\": \"$DOC_ID\",
  \"title\": \"Test Policy\",
  \"content\": \"Updated content\",
  \"effective_date\": \"2025-02-01\"
}" localhost:50051 document.DocumentService/UpdateDocument)

NEW_DOC_ID=$(echo $UPDATE | jq -r '.document.id')
echo "New version: $NEW_DOC_ID"

# 6. List all versions
grpcurl -plaintext -d '{"title": "Test Policy"}' \
  localhost:50051 document.DocumentService/ListDocumentVersions

# 7. Publish new version
grpcurl -plaintext -d "{\"id\": \"$NEW_DOC_ID\"}" \
  localhost:50051 document.DocumentService/PublishDocument

# 8. Verify v1 is archived, v2 is published
grpcurl -plaintext -d '{"title": "Test Policy"}' \
  localhost:50051 document.DocumentService/ListDocumentVersions
```

## Configuration

### Environment Variables
```bash
# Database
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/document_db?sslmode=disable"

# Server
SERVER_PORT="50051"
```

## Troubleshooting

### Service won't start
```bash
# Check PostgreSQL
docker-compose ps policy_postgres

# Check migrations
docker-compose logs document_migrate

# Check service logs
docker-compose logs document_service
```

### Version conflict errors
```bash
# Check existing versions
docker exec -it policy_postgres psql -U postgres -d document_db -c \
  "SELECT title, version, status FROM policy_documents ORDER BY title, version;"

# The UNIQUE(title, version) constraint prevents duplicate versions
```

### Can't publish document
```bash
# Check current status
docker exec -it policy_postgres psql -U postgres -d document_db -c \
  "SELECT id, title, version, status FROM policy_documents WHERE id = 'your-doc-id';"

# Only Draft can be published
```

## Performance

### Database Indexes
```sql
-- Already created in migrations
CREATE INDEX idx_policy_documents_title ON policy_documents(title);
CREATE INDEX idx_policy_documents_status ON policy_documents(status);
CREATE INDEX idx_policy_documents_owner ON policy_documents(owner);
CREATE UNIQUE INDEX idx_policy_documents_title_version ON policy_documents(title, version);
```

### Query Optimization
- **GetDocumentByTitle:** Uses index on (title, status, version)
- **ListDocuments:** Uses index on status, groups by title
- **ListDocumentVersions:** Uses index on title, sorts by version

## Related Services

- **User Service:** Provides owner field (user_id)
- **Consent Service:** Links user consents to specific document versions
- **Gateway Service:** Exposes REST API for frontend

## Support

For issues or questions:
1. Check logs: `docker-compose logs -f document_service`
2. Check database: `docker exec -it policy_postgres psql -U postgres -d document_db`
3. Test gRPC: `grpcurl -plaintext localhost:50051 list document.DocumentService`
