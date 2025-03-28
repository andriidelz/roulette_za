-- Пользователи
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10),
    country VARCHAR(2),  -- ISO 3166-1 alpha-2 код страны
    wallet_address VARCHAR(255), -- Адрес кошелька USDT
    avatar_url VARCHAR(512), -- Адрес кошелька USDT
    balance FLOAT DEFAULT 0,
    banned BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

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
    UNIQUE (key, language)
);

CREATE INDEX IF NOT EXISTS idx_localizations_key ON localizations (key);
CREATE INDEX IF NOT EXISTS idx_localizations_language ON localizations (language);

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
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications (type);

-- Вывод средств
CREATE TABLE IF NOT EXISTS withdrawals (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    amount FLOAT NOT NULL,
    status VARCHAR(20) NOT NULL,
    wallet VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals (user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals (status);

-- Базовые настройки
INSERT INTO settings (key, value, default_value, description) VALUES
    ('daily_bets_limit', '2880', '2880', 'Лимит ставок за день'),
    ('daily_bets_zero_limit', '100', '100', 'Лимит ставок за день для возможности ставить на zero'),
    ('weekly_prize_amount', '1000', '1000', 'Сумма недельного призового фонда'),
    ('weekly_prize_top', '100', '100', 'Количество призовых мест в недельном рейтинге'),
    ('minimum_withdrawal', '10', '10', 'Минимальная сумма для вывода средств'),
    ('prize_distribution_day', '1', '1', 'День недели для раздачи призов (1-7, где 1 - Понедельник)'),
    ('prize_distribution_time', '00:00', '00:00', 'Время раздачи призов (UTC+0)')