-- Create databases for each service
CREATE DATABASE document_db;
CREATE DATABASE user_db;
CREATE DATABASE consent_db;

-- Grant permissions (optional, vì đang dùng superuser postgres)
GRANT ALL PRIVILEGES ON DATABASE document_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE user_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE consent_db TO postgres;