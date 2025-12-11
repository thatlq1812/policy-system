-- Remove Super Admin seed data
DELETE FROM users WHERE phone_number = '0900000000' AND platform_role = 'Admin';
