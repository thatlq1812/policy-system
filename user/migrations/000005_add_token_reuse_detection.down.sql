-- Rollback token reuse detection
DROP INDEX IF EXISTS idx_refresh_tokens_last_used;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS last_used_at;
