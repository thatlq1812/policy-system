-- document/migrations/000001_create_policy_documents_table.up.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Bảng lưu trữ thông tin metadata của tài liệu chính sách
CREATE TABLE IF NOT EXISTS policy_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_name VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL, -- 'FuNong Client' hoặc 'FuNong Merchant' [cite: 87, 403]
    is_mandatory BOOLEAN NOT NULL DEFAULT FALSE, -- Giá trị 'Cần chú ý' [cite: 100]
    
    -- Timestamp này dùng làm Version (Epoch time timestamp upload file của tài liệu) [cite: 371]
    effective_timestamp BIGINT NOT NULL, -- 'Thời gian phát hành' (Unix epoch time) [cite: 370]
    
    content_html TEXT, -- Nội dung soạn thảo văn bản HTML [cite: 105]
    file_url VARCHAR(512), -- Link file PDF/Image (sau khi upload)
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100) -- Ghi nhận người vận hành (email/tên) [cite: 89]
);

-- Ràng buộc để đảm bảo không có hai tài liệu cùng tên trên cùng một nền tảng với cùng effective_timestamp
CREATE INDEX idx_policy_platform_name ON policy_documents(platform, document_name);
CREATE INDEX idx_policy_effective_ts ON policy_documents(effective_timestamp DESC);