-- Видаляємо колонки
ALTER TABLE admin_users DROP COLUMN IF EXISTS created_by;

-- Видаляємо модуль (права видаляться автоматично через ON DELETE CASCADE у 000028)
DELETE FROM modules WHERE code = 'administrator_management';
