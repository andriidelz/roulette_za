-- User activity logs table for tracking all user actions
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    -- Action type (composite, e.g., "bet", "callback_bet_red", "message_text", "message_sticker")
    action_type VARCHAR(100) NOT NULL,
    -- Action data in JSON format (flexible, can store any additional information)
    action_data JSONB,
    -- Message ID from Telegram (optional, for linking to specific message)
    message_id BIGINT,
    -- Timestamp
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
-- Indexes for fast queries
CREATE INDEX idx_activity_logs_telegram_id ON user_activity_logs(telegram_id);
CREATE INDEX idx_activity_logs_action_type ON user_activity_logs(action_type);
CREATE INDEX idx_activity_logs_created_at ON user_activity_logs(created_at DESC);
CREATE INDEX idx_activity_logs_telegram_created ON user_activity_logs(telegram_id, created_at DESC);
CREATE INDEX idx_activity_logs_cleanup ON user_activity_logs(created_at, id);
-- Comment on table
COMMENT ON TABLE user_activity_logs IS 'Logs of all user actions for bot activity analysis and bot detection';
COMMENT ON COLUMN user_activity_logs.telegram_id IS 'Telegram user ID from update';
COMMENT ON COLUMN user_activity_logs.action_type IS 'Type of action: bet, command_start, callback_*, message_*, etc.';
COMMENT ON COLUMN user_activity_logs.action_data IS 'JSON data with action details (command args, bet options, message content, etc.)';
COMMENT ON COLUMN user_activity_logs.chat_id IS 'Telegram chat ID (for groups it differs from user telegram_id)';
