-- document/migrations/000002_add_admin_platform.down.sql
-- Revert Admin platform addition

ALTER TABLE policy_documents 
DROP CONSTRAINT IF EXISTS policy_documents_platform_check;

ALTER TABLE policy_documents
ADD CONSTRAINT policy_documents_platform_check 
CHECK (platform IN ('Client', 'Merchant'));
