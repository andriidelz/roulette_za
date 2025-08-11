-- Добавляем столбец last_activity_at в таблицу users
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMP;

-- Устанавливаем значение по умолчанию для существующих записей
UPDATE users SET last_activity_at = COALESCE(updated_at, created_at) WHERE last_activity_at IS NULL;

-- Добавляем индекс для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_users_last_activity_at ON users(last_activity_at);
