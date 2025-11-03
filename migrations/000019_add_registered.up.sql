-- Добавляем столбец registered в таблицу users
ALTER TABLE users ADD COLUMN IF NOT EXISTS registered BOOLEAN DEFAULT FALSE;

-- true if user finish registration
UPDATE users SET registered = TRUE WHERE age_verified = true AND country != '' AND language_code != '';

