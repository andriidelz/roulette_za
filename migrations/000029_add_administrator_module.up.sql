-- 1. Додаємо поля для "Мінімальної моделі" з ТЗ, яких немає в 000028
ALTER TABLE admin_users 
ADD COLUMN IF NOT EXISTS created_by INTEGER REFERENCES admin_users(id);

-- 2. Додаємо модуль управління адмінами
INSERT INTO modules (code, name, description) 
VALUES ('administrator_management', 'Admins', 'Керування адміністраторами системи')
ON CONFLICT (code) DO NOTHING;

-- 3. Надаємо права Super Admin на цей новий модуль (PBAC)
INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, true, true, true, false
FROM roles r CROSS JOIN modules m 
WHERE r.code = 'super_admin' AND m.code = 'administrator_management'
ON CONFLICT DO NOTHING;

-- Права для Admin (read, write, edit, БЕЗ delete)
INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, true, true, false, false
FROM roles r CROSS JOIN modules m 
WHERE r.code = 'admin' AND m.code = 'administrator_management'
ON CONFLICT DO NOTHING;
