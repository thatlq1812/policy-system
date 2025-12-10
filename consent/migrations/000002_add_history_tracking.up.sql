-- Add history tracking fields to user_consents table
-- These fields support consent versioning and audit trail

-- Add is_latest flag to track current active consent
ALTER TABLE user_consents ADD COLUMN is_latest BOOLEAN DEFAULT TRUE;

-- Add revocation tracking fields
ALTER TABLE user_consents ADD COLUMN revoked_at TIMESTAMP;
ALTER TABLE user_consents ADD COLUMN revoked_reason TEXT;
ALTER TABLE user_consents ADD COLUMN revoked_by VARCHAR(255);

-- Create index for efficient history queries
CREATE INDEX idx_user_document_history ON user_consents(user_id, document_id, version_timestamp DESC, is_latest);

-- Create index for latest consents lookup
CREATE INDEX idx_latest_consents ON user_consents(user_id, is_latest) WHERE is_latest = TRUE AND is_deleted = FALSE;

-- Comments explaining the schema
COMMENT ON COLUMN user_consents.is_latest IS 'Indicates if this is the latest/current consent for this user+document combination';
COMMENT ON COLUMN user_consents.revoked_at IS 'Timestamp when consent was revoked (if applicable)';
COMMENT ON COLUMN user_consents.revoked_reason IS 'Reason for revocation (e.g., user_request, new_version, admin_action)';
COMMENT ON COLUMN user_consents.revoked_by IS 'User ID or system identifier that revoked the consent';
