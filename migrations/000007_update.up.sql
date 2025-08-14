-- Добавляем поле source и ref_key в таблицу пользователей
ALTER TABLE users
ADD COLUMN IF NOT EXISTS source VARCHAR(10);
ALTER TABLE users
ADD COLUMN IF NOT EXISTS ref_key VARCHAR(10);