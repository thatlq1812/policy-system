DROP TRIGGER IF EXISTS trigger_update_user_consents_updated_at ON user_consents;
DROP FUNCTION IF EXISTS update_user_consents_updated_at();
DROP INDEX IF EXISTS idx_user_consents_deleted;
DROP INDEX IF EXISTS idx_user_consents_platform;
DROP INDEX IF EXISTS idx_user_consents_document;
DROP INDEX IF EXISTS idx_user_consents_user_id;
DROP INDEX IF EXISTS idx_active_consents;
DROP TABLE IF EXISTS user_consents;