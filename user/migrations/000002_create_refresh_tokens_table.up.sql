-- Create refresh_tokens table for dual token authentication
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,
    revoked_reason VARCHAR(255),
    device_info TEXT,
    ip_address VARCHAR(50)
);

-- Index for finding tokens by user_id (logout all devices)
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Index for fast token verification
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- Index for cleanup expired tokens
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Composite index for active tokens query
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id, expires_at) 
WHERE revoked_at IS NULL;