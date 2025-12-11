-- Rollback token blacklist table creation
DROP INDEX IF EXISTS idx_token_blacklist_user_id;
DROP INDEX IF EXISTS idx_token_blacklist_expires_at;
DROP TABLE IF EXISTS token_blacklist;
