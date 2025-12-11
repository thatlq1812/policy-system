-- user/migrations/000003_add_admin_role.down.sql
-- Revert Admin role addition

ALTER TABLE users 
DROP CONSTRAINT users_platform_role_check;

ALTER TABLE users
ADD CONSTRAINT users_platform_role_check 
CHECK (platform_role IN ('Client', 'Merchant'));
