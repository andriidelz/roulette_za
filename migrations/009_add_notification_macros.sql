-- Добавляем поле для хранения макросов в таблицу notification_recipients
ALTER TABLE notification_recipients ADD COLUMN IF NOT EXISTS macros JSONB DEFAULT '{}'::jsonb;

-- Удаление поля user_id в таблице notification_tasks
ALTER TABLE notification_tasks 
DROP COLUMN IF EXISTS user_id;
