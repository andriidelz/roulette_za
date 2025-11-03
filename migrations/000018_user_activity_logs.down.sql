-- Drop indexes
DROP INDEX IF EXISTS idx_activity_logs_cleanup;
DROP INDEX IF EXISTS idx_activity_logs_telegram_created;
DROP INDEX IF EXISTS idx_activity_logs_created_at;
DROP INDEX IF EXISTS idx_activity_logs_action_type;
DROP INDEX IF EXISTS idx_activity_logs_telegram_id;
-- Drop table
DROP TABLE IF EXISTS user_activity_logs;
