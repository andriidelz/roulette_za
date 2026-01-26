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
    type_status VARCHAR(50),
    reason VARCHAR(50),
    reason_meta VARCHAR(500),
    active BOOLEAN DEFAULT FALSE,
    stage INT DEFAULT 0,
    wrong INT DEFAULT 0,
    refresh INT DEFAULT 0,
    until_to TIMESTAMP,
    updated_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_ban_logs_user_id ON user_ban_logs (user_id);

ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20);
UPDATE users SET status = 'ACTIVE' WHERE registered = true AND banned = false;
UPDATE users SET status = 'BANNED' WHERE registered = true AND banned = true;

INSERT INTO localizations (key, language, value) VALUES
    ('captcha_manual_ban', 'uk', 'Вас було забанено'),
    ('captcha_manual_captcha', 'uk', 'Вам пропонується пройти капчу'),
    ('captcha_manual_active', 'uk', 'Вас розблоковано'),
    ('captcha_refresh', 'uk', 'Оновити капчу'),
    ('captcha_blocked_action', 'uk', 'Пройдіть капчу'),
    ('captcha_stage_title', 'uk', 'Етап %d / %d'),
    ('captcha_next', 'uk', 'Вітаємо з успішним проходженням капчі! Пройдіть наступний етап'),
    ('captcha_manual_ban', 'ru', 'Вас был забанено'), 
    ('captcha_manual_captcha', 'ru', 'Вам предлагается пройти капчу'), 
    ('captcha_manual_active', 'ru', 'Вас разблокировано'),
    ('captcha_refresh', 'ru', 'Обновить капчу'),
    ('captcha_blocked_action', 'ru', 'Пройдите капчу'),
    ('captcha_stage_title', 'ru', 'Этап %d / %d'),
    ('captcha_next', 'ru', 'Поздравляем с успешным прохождением капчи! Пройдите следующий этап'),
    ('captcha_manual_ban', 'en', 'You have been banned'),
    ('captcha_manual_captcha', 'en', 'You are asked to pass the captcha'),
    ('captcha_manual_active', 'en', 'You are unblocked'),
    ('captcha_refresh', 'en', 'Refresh captcha'),
    ('captcha_blocked_action', 'en', 'Resolve captcha'),
    ('captcha_stage_title', 'en', 'Stage %d / %d'),
    ('captcha_next', 'en', 'Congratulations on successfully passing the captcha! Go to the next stage');