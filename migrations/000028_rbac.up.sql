-- 1. First of all crate a function to update updated_at column on each update
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Modules table
CREATE TABLE modules (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger for modules
CREATE TRIGGER update_modules_modtime 
BEFORE UPDATE ON modules 
FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

-- Roles table
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Тригер for roles
CREATE TRIGGER update_roles_modtime 
BEFORE UPDATE ON roles 
FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

-- Roles and modules connection with permissions (PBAC)
CREATE TABLE role_modules (
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    module_id INTEGER REFERENCES modules(id) ON DELETE CASCADE,
    can_read BOOLEAN DEFAULT false,
    can_write BOOLEAN DEFAULT false,
    can_edit BOOLEAN DEFAULT false,
    can_delete BOOLEAN DEFAULT false,
    can_add_balance BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, module_id)
);

-- Admin users table
CREATE TABLE admin_users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,    
    first_name VARCHAR(100),
    last_name VARCHAR(100),    
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMP,
    created_by INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Trigger for admin_users
CREATE TRIGGER update_admin_users_modtime 
BEFORE UPDATE ON admin_users 
FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

-- Users and Roles connection (Many-to-Many)
CREATE TABLE admin_user_roles (
    user_id INTEGER REFERENCES admin_users(id) ON DELETE CASCADE,
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- Access attempt logging
CREATE TABLE access_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES admin_users(id),
    module_code VARCHAR(50),
    action VARCHAR(100),
    ip_address VARCHAR(45),
    user_agent TEXT,
    is_allowed BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for optimization
CREATE INDEX idx_admin_users_email ON admin_users(email);
CREATE INDEX idx_admin_users_is_active ON admin_users(is_active);
CREATE INDEX idx_roles_code ON roles(code);
CREATE INDEX idx_modules_code ON modules(code);
CREATE INDEX idx_access_logs_user_id ON access_logs(user_id);
CREATE INDEX idx_access_logs_created_at ON access_logs(created_at);
CREATE INDEX idx_role_modules_role_id ON role_modules(role_id);
CREATE INDEX idx_role_modules_module_id ON role_modules(module_id);
CREATE INDEX idx_admin_users_deleted_at ON admin_users(deleted_at);

-- Initial data
INSERT INTO modules (code, name, description) VALUES
    ('dashboard', 'Dashboard', 'Головна панель з статистикою'),
    ('users', 'Users Management', 'Управління користувачами'),
    ('statistics', 'Statistics', 'Статистика та аналітика'),
    ('withdrawals', 'Withdrawals', 'Управління виплатами'),
    ('notifications', 'Notifications', 'Система сповіщень'),
    ('settings', 'Settings', 'Налаштування системи'),
    ('localizations', 'Localizations', 'Управління локалізаціями'),
    ('ratings', 'Ratings', 'Рейтинги та змагання'),
    ('sources', 'Sources & Keys', 'Джерела трафіку'),
    ('hashes', 'Hashes', 'Перевірка хешів'),
    ('activity_analyzer', 'Activity Analyzer', 'Аналіз активності користувачів'),
    ('rbac_management', 'RBAC Management', 'Управління ролями та правами доступу'),
    ('administrator_management', 'Admins Management', 'Управління адміністраторами');

-- System roles
INSERT INTO roles (code, name, description, is_system) VALUES
    ('super_admin', 'Super Admin', 'Повний доступ до всіх модулів', true),
    ('admin', 'Admin', 'Адміністратор (обмежений доступ)', true), 
    ('editor', 'Editor', 'Редактор (Уведомления, Локализация)', true),
    ('copywriter', 'Copywriter', 'Копірайтер (Тільки читання текстів)', true),
    ('media_buyer', 'Media Buyer', 'Медіа-баєр (Реф. посилання)', true);

-- ============ PERMISSIONS FOR SUPER ADMIN (all modules, all permissions) ============
INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, true, true, true, true
FROM roles r 
CROSS JOIN modules m 
WHERE r.code = 'super_admin';

DELETE FROM role_modules 
WHERE role_id IN (SELECT id FROM roles WHERE code IN ('admin', 'media_buyer'));

-- ============ PERMISSIONS FOR ADMIN (all modules instead of rbac_management, read/write/edit) ============
INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, true, true, true, false
FROM roles r 
CROSS JOIN modules m 
WHERE r.code = 'admin' 
  AND m.code NOT IN ('rbac_management', 'administrator_management');

UPDATE role_modules 
SET can_edit = true 
WHERE role_id = (SELECT id FROM roles WHERE code = 'admin') 
  AND module_id = (SELECT id FROM modules WHERE code = 'settings');

UPDATE role_modules 
SET can_delete = false, can_add_balance = false
WHERE role_id = (SELECT id FROM roles WHERE code = 'admin') 
  AND module_id = (SELECT id FROM modules WHERE code = 'users');

-- ============ PERMISSIONS FOR EDITOR: Notifications + Localizations (Full) ============
INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, true, true, true, false
FROM roles r 
CROSS JOIN modules m 
WHERE r.code = 'editor' AND m.code IN ('notifications', 'localizations');

UPDATE role_modules 
SET can_delete = true
WHERE role_id = (SELECT id FROM roles WHERE code = 'editor')
  AND module_id IN (
    SELECT id FROM modules WHERE code IN ('notifications', 'localizations')
  );

-- ============ PERMISSIONS FOR COPYWRITER: Notifications + Localizations (Read Only) ============
INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, false, false, false, false
FROM roles r 
CROSS JOIN modules m 
WHERE r.code = 'copywriter' AND m.code IN ('notifications', 'localizations');

-- ============ PERMISSIONS FOR MEDIA BUYER: Sources (Read, Write, Edit) ============
INSERT INTO role_modules (role_id, module_id, can_read, can_write, can_edit, can_delete, can_add_balance)
SELECT r.id, m.id, true, true, true, false, false
FROM roles r 
CROSS JOIN modules m 
WHERE r.code = 'media_buyer' AND m.code = 'sources';

DELETE FROM role_modules 
WHERE role_id = (SELECT id FROM roles WHERE code = 'media_buyer') 
AND module_id = (SELECT id FROM modules WHERE code = 'settings');

-- Comment with instructions
COMMENT ON TABLE admin_users IS 'IMPORTANT: First admin user is created automatically (admin/admin). It is mandatory to change the password after the first login!';
