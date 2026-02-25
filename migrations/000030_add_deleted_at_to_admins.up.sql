ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_admin_users_deleted_at ON admin_users(deleted_at)

-- Додаємо реєстрацію модуля для RBAC (фікс доступу)
INSERT INTO modules (code, name, description) 
VALUES ('localizations', 'Локалізації', 'Керування перекладами системи')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, true, true, true, false
FROM roles r CROSS JOIN modules m 
WHERE r.code = 'super_admin' AND m.code = 'localizations'
ON CONFLICT DO NOTHING;
