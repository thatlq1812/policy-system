-- user/migrations/000001_create_users_table.up.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Bảng lưu trữ thông tin tài khoản người dùng
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,  -- CRITICAL: hash, not plain text
    name VARCHAR(100),
    platform_role VARCHAR(20) NOT NULL CHECK (platform_role IN ('Client', 'Merchant')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE
);

-- Index cho tìm kiếm nhanh qua số điện thoại (used in login)
CREATE INDEX idx_users_phone_number ON users (phone_number);

-- Index cho filtering active users
CREATE INDEX idx_users_active ON users (is_deleted) WHERE is_deleted = FALSE;

-- Index cho platform role analytics (optional)
CREATE INDEX idx_users_role ON users (platform_role);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();