INSERT INTO settings (key, value, default_value, description) VALUES
    ('captcha_ttl', '3', '3', 'Время ожидания капчи (мин)'),
    ('captcha_refresh_count', '3', '3', 'Кол-во обновлений'),
    ('captcha_need_count', '3', '3', 'Кол-во этапов'),
    ('captcha_wrong_count', '3', '3', 'Кол-во неправильнх ответов'),
    ('captcha_ban_count', '3', '3', 'Кол-во банов'),
    ('captcha_ban_short_ttl', '60', '60', 'Время бана short (мин)'),
    ('captcha_ban_long_ttl', '1440', '1440', 'Время бана long (мин)'),
    ('captcha_user_activity_ttl', '10', '10', 'Период действий для лимита (сек)'),
    ('captcha_bet_activity_ttl', '180', '180', 'Период ставок для лимита (сек)'),
    ('captcha_bet_duplicate_ttl', '1800', '1800', 'Период дубликатов ставок (сек)');

CREATE TABLE IF NOT EXISTS user_ban_logs (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    type_status VARCHAR(255),
    reason VARCHAR(255),
    active BOOLEAN DEFAULT FALSE,
    stage INT DEFAULT 0,
    wrong INT DEFAULT 0,
    refresh INT DEFAULT 0,
    until_to TIMESTAMP,
    updated_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_ban_logs_user_id ON user_ban_logs (user_id);
