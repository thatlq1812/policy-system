-- Rollback history tracking changes

-- Drop indexes
DROP INDEX IF EXISTS idx_latest_consents;
DROP INDEX IF EXISTS idx_user_document_history;

-- Remove columns
ALTER TABLE user_consents DROP COLUMN IF EXISTS revoked_by;
ALTER TABLE user_consents DROP COLUMN IF EXISTS revoked_reason;
ALTER TABLE user_consents DROP COLUMN IF EXISTS revoked_at;
ALTER TABLE user_consents DROP COLUMN IF EXISTS is_latest;
