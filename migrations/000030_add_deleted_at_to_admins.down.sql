DROP INDEX IF EXISTS idx_admin_users_deleted_at;
ALTER TABLE admin_users DROP COLUMN IF EXISTS deleted_at;
