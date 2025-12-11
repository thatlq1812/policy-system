-- Create token_blacklist table for revoking access tokens
-- This table stores JWT access tokens that have been explicitly revoked (e.g., on logout)
-- Tokens are identified by their JTI (JWT ID) claim for efficient lookup

CREATE TABLE IF NOT EXISTS token_blacklist (
    id SERIAL PRIMARY KEY,
    
    -- JWT Token Identifier (jti claim) - unique identifier for the token
    jti VARCHAR(255) NOT NULL UNIQUE,
    
    -- User ID who owns the token (for reference and cleanup)
    user_id VARCHAR(255) NOT NULL,
    
    -- When the token expires (from exp claim)
    -- Tokens can be safely removed from blacklist after expiration
    expires_at TIMESTAMP NOT NULL,
    
    -- When the token was blacklisted
    blacklisted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Reason for blacklisting (e.g., 'logout', 'password_changed', 'security_revoke')
    reason VARCHAR(100),
    
    -- Index for fast JTI lookup (primary use case)
    CONSTRAINT idx_jti UNIQUE (jti)
);

-- Index for cleanup queries (finding expired tokens)
CREATE INDEX idx_token_blacklist_expires_at ON token_blacklist(expires_at);

-- Index for user-specific queries
CREATE INDEX idx_token_blacklist_user_id ON token_blacklist(user_id);

-- Add comment for documentation
COMMENT ON TABLE token_blacklist IS 'Stores revoked JWT access tokens to prevent their use after logout or security events';
COMMENT ON COLUMN token_blacklist.jti IS 'JWT ID claim - unique identifier for each access token';
COMMENT ON COLUMN token_blacklist.expires_at IS 'Token expiration time from JWT exp claim - used for automatic cleanup';
