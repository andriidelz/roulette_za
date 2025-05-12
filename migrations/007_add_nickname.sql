-- Создаем файл миграции 007_add_nickname.sql

-- Добавляем поле nickname в таблицу пользователей
ALTER TABLE users
ADD COLUMN IF NOT EXISTS nickname VARCHAR(50) DEFAULT NULL;

-- Добавляем индекс для быстрого поиска по никнейму
CREATE INDEX IF NOT EXISTS idx_users_nickname ON users(nickname);

-- Комментарий к полю
COMMENT ON COLUMN users.nickname IS 'Никнейм пользователя для отображения в рейтингах и публичных частях приложения';

-- Первоначальное заполнение nickname из username для существующих пользователей
-- (необязательно, но может быть полезно)
UPDATE users SET nickname = username WHERE nickname IS NULL AND username IS NOT NULL;
