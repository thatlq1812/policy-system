-- Add last_used_at field for token reuse detection
ALTER TABLE refresh_tokens ADD COLUMN last_used_at TIMESTAMP;

-- Index for detecting suspicious token reuse patterns
CREATE INDEX idx_refresh_tokens_last_used ON refresh_tokens(token_hash, last_used_at);

-- Comment explaining the field
COMMENT ON COLUMN refresh_tokens.last_used_at IS 'Timestamp when token was last used for refresh. Helps detect token theft - if revoked token is reused, indicates compromise.';
