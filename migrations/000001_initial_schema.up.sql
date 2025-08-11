-- Пользователи
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    source VARCHAR(10),
    ref_key VARCHAR(10),
    language_code VARCHAR(10),
    country VARCHAR(2),  -- ISO 3166-1 alpha-2 код страны
    wallet_address VARCHAR(255), -- Адрес кошелька USDT
    avatar_url VARCHAR(512), -- URL аватара пользователя
    balance FLOAT DEFAULT 0,
    banned BOOLEAN DEFAULT FALSE,
    age_verified BOOLEAN DEFAULT NULL, -- Подтверждение возраста пользователя
    nickname VARCHAR(50) DEFAULT NULL, -- Никнейм для отображения в рейтинге
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Добавляем индекс для быстрого поиска по никнейму
CREATE INDEX IF NOT EXISTS idx_users_nickname ON users(nickname);

-- Комментарий к полю
COMMENT ON COLUMN users.nickname IS 'Никнейм пользователя для отображения в рейтингах и публичных частях приложения';

-- Хеши (раунды игры)
CREATE TABLE IF NOT EXISTS hash_entries (
    id SERIAL PRIMARY KEY,
    number BIGINT NOT NULL,
    salt_hex VARCHAR(64) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    revealed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hash_entries_hash ON hash_entries (hash);
CREATE INDEX IF NOT EXISTS idx_hash_entries_is_completed ON hash_entries (is_completed);

-- Ставки
CREATE TABLE IF NOT EXISTS bets (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    hash_entry_id INT NOT NULL REFERENCES hash_entries(id),
    option VARCHAR(10) NOT NULL,
    won BOOLEAN DEFAULT FALSE,
    points INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bets_user_id ON bets (user_id);
CREATE INDEX IF NOT EXISTS idx_bets_hash_entry_id ON bets (hash_entry_id);
CREATE INDEX IF NOT EXISTS idx_bets_created_at ON bets (created_at);

-- Недельный рейтинг
CREATE TABLE IF NOT EXISTS weekly_ratings (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    week INT NOT NULL,
    year INT NOT NULL,
    points INT DEFAULT 0,
    bets INT DEFAULT 0,
    efficiency FLOAT DEFAULT 0,
    position INT DEFAULT 0,
    prize FLOAT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (user_id, week, year)
);

CREATE INDEX IF NOT EXISTS idx_weekly_ratings_user_id ON weekly_ratings (user_id);
CREATE INDEX IF NOT EXISTS idx_weekly_ratings_week ON weekly_ratings (week);
CREATE INDEX IF NOT EXISTS idx_weekly_ratings_year ON weekly_ratings (year);
CREATE INDEX IF NOT EXISTS idx_weekly_ratings_points ON weekly_ratings(points DESC);
CREATE INDEX IF NOT EXISTS idx_weekly_ratings_position ON weekly_ratings(position);

-- Супер рейтинг
CREATE TABLE IF NOT EXISTS super_ratings (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    period VARCHAR(20) NOT NULL,
    points INT DEFAULT 0,
    positions INT DEFAULT 0,
    position INT DEFAULT 0,
    prize FLOAT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (user_id, period)
);

CREATE INDEX IF NOT EXISTS idx_super_ratings_user_id ON super_ratings (user_id);
CREATE INDEX IF NOT EXISTS idx_super_ratings_period ON super_ratings (period);
CREATE INDEX IF NOT EXISTS idx_super_ratings_points ON super_ratings(points DESC);
CREATE INDEX IF NOT EXISTS idx_super_ratings_position ON super_ratings(position);

-- Настройки
CREATE TABLE IF NOT EXISTS settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    default_value TEXT,
    description TEXT
);

-- Локализации
CREATE TABLE IF NOT EXISTS localizations (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL,
    language VARCHAR(10) NOT NULL,
    value TEXT,
    image TEXT,
    UNIQUE (key, language)
);

CREATE INDEX IF NOT EXISTS idx_localizations_key ON localizations (key);
CREATE INDEX IF NOT EXISTS idx_localizations_language ON localizations (language);

-- Источники
CREATE TABLE IF NOT EXISTS source_keys (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL,
    name TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (key)
);

CREATE INDEX IF NOT EXISTS idx_source_keys_key ON source_keys (key);

-- Призовые фонды
CREATE TABLE IF NOT EXISTS prize_funds (
    id SERIAL PRIMARY KEY,
    week INT NOT NULL,
    year INT NOT NULL,
    amount FLOAT DEFAULT 0,
    top_count INT DEFAULT 100,
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (week, year)
);

CREATE INDEX IF NOT EXISTS idx_prize_funds_week ON prize_funds (week);
CREATE INDEX IF NOT EXISTS idx_prize_funds_year ON prize_funds (year);
CREATE INDEX IF NOT EXISTS idx_prize_funds_week_year ON prize_funds(week, year);

-- Уведомления
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    type VARCHAR(50) NOT NULL,
    message TEXT,
    title TEXT,
    image_url TEXT,
    button_text TEXT,
    button_url TEXT,
    button_callback TEXT,
    read BOOLEAN DEFAULT FALSE,
    delivered BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications (type);

-- Шаблоны уведомлений
CREATE TABLE IF NOT EXISTS notification_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'manual', 'automatic'
    trigger_event VARCHAR(50), -- Событие для автоматических уведомлений
    title_key VARCHAR(255), -- Ключ локализации для заголовка
    message_key VARCHAR(255), -- Ключ локализации для сообщения
    image_url TEXT,
    image_urls JSONB DEFAULT '{}'::jsonb, -- Локализованные изображения
    button_text_key VARCHAR(255), -- Ключ локализации для текста кнопки
    button_url TEXT, -- URL для кнопки
    button_callback TEXT, -- Callback для кнопки
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Задачи на отправку уведомлений
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

-- Получатели конкретной задачи уведомлений
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

-- Пример автоматических шаблонов уведомлений
INSERT INTO notification_templates (name, type, trigger_event, title_key, message_key, active) VALUES
('Зачисление вознаграждения', 'automatic', 'balance_updated', 'balance_updated_title', 'balance_updated_message', true),
('Вход в еженедельный топ', 'automatic', 'top_rating_entered', 'top_rating_title', 'top_rating_message', true);


-- Индексы для уведомлений
CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON notification_templates(type);
CREATE INDEX IF NOT EXISTS idx_notification_templates_active ON notification_templates(active);
CREATE INDEX IF NOT EXISTS idx_notification_tasks_status ON notification_tasks(status);
CREATE INDEX IF NOT EXISTS idx_notification_tasks_scheduled_at ON notification_tasks(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_task_id ON notification_recipients(task_id);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_user_id ON notification_recipients(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_status ON notification_recipients(status);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_scheduled_at ON notification_recipients(scheduled_at);

-- Вывод средств
CREATE TABLE IF NOT EXISTS withdrawals (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    amount FLOAT NOT NULL,
    status VARCHAR(20) NOT NULL,
    wallet VARCHAR(255),
    provider_name VARCHAR(50) DEFAULT 'oxapay',
    provider_id VARCHAR(255),
    transaction_hash VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals (user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals (status);
CREATE INDEX IF NOT EXISTS idx_provider_name ON withdrawals (provider_name);
CREATE INDEX IF NOT EXISTS idx_provider_id ON withdrawals (provider_id);

-- Создаем таблицу для отслеживания обновлений статуса выплат (для аудита)
CREATE TABLE IF NOT EXISTS withdrawal_status_logs (
    id SERIAL PRIMARY KEY,
    withdrawal_id BIGINT NOT NULL REFERENCES withdrawals(id),
    old_status VARCHAR(50),
    new_status VARCHAR(50) NOT NULL,
    provider_name VARCHAR(50) NOT NULL,
    transaction_hash VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Базовые настройки
INSERT INTO settings (key, value, default_value, description) VALUES
    ('daily_bets_limit', '2880', '2880', 'Лимит ставок за день'),
    ('daily_bets_zero_limit', '100', '100', 'Лимит ставок за день для возможности ставить на zero'),
    ('weekly_prize_amount', '1000', '1000', 'Сумма недельного призового фонда'),
    ('weekly_prize_top', '100', '100', 'Количество призовых мест в недельном рейтинге'),
    ('minimum_withdrawal', '10', '10', 'Минимальная сумма для вывода средств'),
    ('prize_distribution_day', '1', '1', 'День недели для раздачи призов (1-7, где 1 - Понедельник)'),
    ('prize_distribution_time', '00:00', '00:00', 'Время раздачи призов (UTC+0)');
