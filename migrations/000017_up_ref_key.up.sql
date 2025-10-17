-- Добавляем поле source и ref_key в таблицу пользователей
ALTER TABLE users ALTER COLUMN source TYPE VARCHAR(20);
ALTER TABLE users ALTER COLUMN ref_key TYPE VARCHAR(20);

