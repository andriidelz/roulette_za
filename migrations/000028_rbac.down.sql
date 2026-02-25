-- Delete the column can_add_balance from role_modules
ALTER TABLE role_modules DROP COLUMN IF EXISTS can_add_balance;

-- Deletion in backwards order to avoid foreign key constraints issues
DROP TABLE IF EXISTS access_logs;
DROP TABLE IF EXISTS admin_user_roles;
DROP TABLE IF EXISTS role_modules;
DROP TABLE IF EXISTS admin_users;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS modules;

-- Delete the trigger functions
DROP FUNCTION IF EXISTS update_modified_column();
