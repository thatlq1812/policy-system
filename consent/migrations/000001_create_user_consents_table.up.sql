CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Bảng lưu trữ lịch sử đồng ý policy của user
CREATE TABLE IF NOT EXISTS user_consents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Định danh người đồng ý
    user_id UUID NOT NULL,
    platform VARCHAR(50) NOT NULL, -- 'Client' hoặc 'Merchant'
    
    -- Định danh tài liệu
    document_id UUID NOT NULL,
    document_name VARCHAR(255) NOT NULL,
    
    -- Versioning
    version_timestamp BIGINT NOT NULL, -- Epoch time của document version
    
    -- Legal evidence (bằng chứng pháp lý)
    agreed_at TIMESTAMPTZ DEFAULT NOW(), -- Thời điểm đồng ý
    agreed_file_url VARCHAR(512), -- Link file snapshot tại thời điểm đồng ý
    consent_method VARCHAR(50) NOT NULL, -- 'REGISTRATION', 'UI', 'API'
    ip_address VARCHAR(45), -- IPv4/IPv6 (optional)
    user_agent TEXT, -- Browser/App info (optional)
    
    -- Soft delete
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    
    -- Audit trail
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Unique constraint: User chỉ consent 1 lần cho mỗi (document, version) - chỉ active records
CREATE UNIQUE INDEX idx_active_consents 
ON user_consents (user_id, document_id, version_timestamp) 
WHERE is_deleted = FALSE;

-- Query optimization indexes
CREATE INDEX idx_user_consents_user_id ON user_consents(user_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_user_consents_document ON user_consents(document_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_user_consents_platform ON user_consents(platform) WHERE is_deleted = FALSE;
CREATE INDEX idx_user_consents_deleted ON user_consents(is_deleted, deleted_at);

-- Trigger tự động update updated_at
CREATE OR REPLACE FUNCTION update_user_consents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_update_user_consents_updated_at 
BEFORE UPDATE ON user_consents 
FOR EACH ROW 
EXECUTE FUNCTION update_user_consents_updated_at();