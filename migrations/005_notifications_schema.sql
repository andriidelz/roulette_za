-- Обновление таблицы notifications
ALTER TABLE notifications 
ADD COLUMN IF NOT EXISTS title TEXT,
ADD COLUMN IF NOT EXISTS image_url TEXT,
ADD COLUMN IF NOT EXISTS button_text TEXT,
ADD COLUMN IF NOT EXISTS button_url TEXT,
ADD COLUMN IF NOT EXISTS button_callback TEXT,
ADD COLUMN IF NOT EXISTS delivered BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS read_at TIMESTAMP;

-- Создаем таблицу шаблонов уведомлений
CREATE TABLE IF NOT EXISTS notification_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'manual', 'automatic'
    trigger_event VARCHAR(50), -- Событие для автоматических уведомлений
    title_key VARCHAR(255), -- Ключ локализации для заголовка
    message_key VARCHAR(255), -- Ключ локализации для сообщения
    image_url TEXT,
    button_text_key VARCHAR(255), -- Ключ локализации для текста кнопки
    button_url TEXT, -- URL для кнопки
    button_callback TEXT, -- Callback для кнопки
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Создаем таблицу задач на отправку уведомлений
CREATE TABLE IF NOT EXISTS notification_tasks (
    id SERIAL PRIMARY KEY,
    template_id INT REFERENCES notification_templates(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    target_type VARCHAR(50) NOT NULL, -- 'all', 'country', 'activity', 'custom'
    target_params JSONB, -- Параметры таргетинга в JSON формате
    scheduled_at TIMESTAMP, -- Время запланированной отправки
    started_at TIMESTAMP, -- Время начала отправки
    completed_at TIMESTAMP, -- Время завершения отправки
    total_users INT DEFAULT 0, -- Общее количество пользователей для отправки
    sent_count INT DEFAULT 0, -- Количество отправленных уведомлений
    delivered_count INT DEFAULT 0, -- Количество доставленных уведомлений
    read_count INT DEFAULT 0, -- Количество прочитанных уведомлений
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Создаем таблицу для хранения получателей конкретной задачи уведомлений
CREATE TABLE IF NOT EXISTS notification_recipients (
    id SERIAL PRIMARY KEY,
    task_id INT NOT NULL REFERENCES notification_tasks(id),
    user_id INT NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'sent', 'delivered', 'read', 'failed'
    scheduled_at TIMESTAMP, -- Время запланированной отправки для конкретного пользователя
    sent_at TIMESTAMP, -- Время отправки
    delivered_at TIMESTAMP, -- Время доставки
    read_at TIMESTAMP, -- Время прочтения
    error_message TEXT, -- Сообщение об ошибке
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Создаем индексы
CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON notification_templates(type);
CREATE INDEX IF NOT EXISTS idx_notification_templates_active ON notification_templates(active);
CREATE INDEX IF NOT EXISTS idx_notification_tasks_status ON notification_tasks(status);
CREATE INDEX IF NOT EXISTS idx_notification_tasks_scheduled_at ON notification_tasks(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_task_id ON notification_recipients(task_id);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_user_id ON notification_recipients(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_status ON notification_recipients(status);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_scheduled_at ON notification_recipients(scheduled_at);

-- Пример автоматических шаблонов уведомлений
INSERT INTO notification_templates (name, type, trigger_event, title_key, message_key, active) VALUES
('Зачисление вознаграждения', 'automatic', 'balance_updated', 'balance_updated_title', 'balance_updated_message', true),
('Вход в еженедельный топ', 'automatic', 'top_rating_entered', 'top_rating_title', 'top_rating_message', true);

-- Добавляем локализации для шаблонов
INSERT INTO localizations (key, language, value) VALUES
('balance_updated_title', 'en', 'Reward credited to your balance'),
('balance_updated_title', 'ru', 'Зачисление вознаграждения на баланс'),
('balance_updated_title', 'uk', 'Зарахування винагороди на баланс'),

('balance_updated_message', 'en', 'You have received a reward of {amount} to your internal balance!'),
('balance_updated_message', 'ru', 'Вы получили вознаграждение в размере {amount} на внутренний баланс!'),
('balance_updated_message', 'uk', 'Ви отримали винагороду у розмірі {amount} на внутрішній баланс!'),

('top_rating_title', 'en', 'Weekly Top Rating'),
('top_rating_title', 'ru', 'Еженедельный топ рейтинг'),
('top_rating_title', 'uk', 'Щотижневий топ рейтинг'),

('top_rating_message', 'en', 'Congratulations! You are now in the weekly top rating at position {position}!'),
('top_rating_message', 'ru', 'Поздравляем! Вы вошли в еженедельный топ рейтинг на позиции {position}!'),
('top_rating_message', 'uk', 'Вітаємо! Ви увійшли до щотижневого топ рейтингу на позиції {position}!');

-- Добавляем новое поле для хранения локализованных изображений
ALTER TABLE notification_templates
ADD COLUMN IF NOT EXISTS image_urls JSONB DEFAULT '{}'::jsonb;

-- Миграция данных: копируем данные из старого поля в новое
UPDATE notification_templates 
SET image_urls = json_build_object('en', image_url, 'ru', image_url, 'uk', image_url)
WHERE image_url IS NOT NULL AND image_url != '';
