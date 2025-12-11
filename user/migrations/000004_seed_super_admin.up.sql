-- Create Super Admin account
-- Password: Admin@123456 (bcrypt hashed)
-- Phone: 0900000000

INSERT INTO users (
    id,
    phone_number,
    password_hash,
    name,
    platform_role,
    created_at,
    updated_at,
    is_deleted
) VALUES (
    gen_random_uuid(),
    '0900000000',
    '$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGQsicN.i.qE0U90IUB8R9u', -- Admin@123456
    'Super Admin',
    'Admin',
    NOW(),
    NOW(),
    FALSE
) ON CONFLICT (phone_number) DO NOTHING;

-- Note: This creates a default Super Admin account for system bootstrap
-- Change password immediately after first login in production
