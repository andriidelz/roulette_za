-- migrations/001_initial_schema.sql

-- Користувачі
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10),
    balance FLOAT DEFAULT 0,
    today_bets INT DEFAULT 0,
    banned BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Ігри
CREATE TABLE IF NOT EXISTS games (
    id SERIAL PRIMARY KEY,
    result VARCHAR(10) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_games_hash ON games (hash);

-- Ставки
CREATE TABLE IF NOT EXISTS bets (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    game_id INT NOT NULL REFERENCES games(id),
    option VARCHAR(10) NOT NULL,
    won BOOLEAN DEFAULT FALSE,
    points INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bets_user_id ON bets (user_id);
CREATE INDEX IF NOT EXISTS idx_bets_game_id ON bets (game_id);

-- Статистика користувача
CREATE TABLE IF NOT EXISTS user_stats (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE NOT NULL REFERENCES users(id),
    total_bets INT DEFAULT 0,
    won_bets INT DEFAULT 0,
    total_points INT DEFAULT 0,
    daily_bets INT DEFAULT 0,
    weekly_bets INT DEFAULT 0,
    monthly_bets INT DEFAULT 0,
    daily_points INT DEFAULT 0,
    weekly_points INT DEFAULT 0,
    monthly_points INT DEFAULT 0,
    last_reset TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Тижневий рейтинг
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

-- Налаштування
CREATE TABLE IF NOT EXISTS settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    default_value TEXT,
    description TEXT
);

-- Локалізації
CREATE TABLE IF NOT EXISTS localizations (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL,
    language VARCHAR(10) NOT NULL,
    value TEXT,
    UNIQUE (key, language)
);

CREATE INDEX IF NOT EXISTS idx_localizations_key ON localizations (key);
CREATE INDEX IF NOT EXISTS idx_localizations_language ON localizations (language);

-- Призові фонди
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

-- Сповіщення
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

-- Виведення коштів
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

-- Базові налаштування
INSERT INTO settings (key, value, default_value, description) VALUES
    ('daily_bets_limit', '100', '100', 'Ліміт ставок за день для можливості ставити на zero'),
    ('weekly_prize_amount', '1000', '1000', 'Сума тижневого призового фонду'),
    ('weekly_prize_top', '100', '100', 'Кількість призових місць у тижневому рейтингу'),
    ('minimum_withdrawal', '10', '10', 'Мінімальна сума для виведення коштів');

-- Базові локалізації українською
INSERT INTO localizations (key, language, value) VALUES
    ('welcome', 'uk', 'Вітаємо у боті рулетки! Тут ви можете робити ставки на червоне, чорне або зеро і змагатися за місце в рейтингу.'),
    ('help', 'uk', 'Доступні команди:\n/play - Почати гру\n/profile - Ваш профіль\n/stats - Ваша статистика\n/rating - Тижневий рейтинг\n/superrating - Супер рейтинг\n/balance - Ваш баланс\n/withdraw - Вивести кошти\n/faq - Часті питання'),
    ('game_instructions', 'uk', 'Оберіть ставку: червоне, чорне або зеро. Кожне вгадування дає 1 бал. Зеро дає 10 балів, але ставити на нього можна тільки після 100 ставок за день.'),
    ('profile_template', 'uk', 'Профіль: %s\nБаланс: %.2f\nЗагальна кількість ставок: %d\nВиграшних ставок: %d\nЕфективність: %.2f%%\nЗагальна кількість балів: %d'),
    ('stats_template', 'uk', 'Статистика:\n\nЗа день:\nСтавок: %d\nБалів: %d\n\nЗа тиждень:\nСтавок: %d\nБалів: %d\n\nЗа місяць:\nСтавок: %d\nБалів: %d\n\nЗагалом:\nСтавок: %d\nБалів: %d'),
    ('rating_header', 'uk', 'Тижневий рейтинг (топ-10):\n\n'),
    ('super_rating_header', 'uk', 'Супер рейтинг (топ-10):\n\n'),
    ('balance_template', 'uk', 'Ваш баланс: %.2f'),
    ('withdraw_instructions', 'uk', 'Для виведення коштів вкажіть суму і реквізити. Мінімальна сума для виведення: 10.'),
    ('faq', 'uk', 'Часті питання:\n\n1. Як нараховуються бали?\nЗа кожне вгадування кольору нараховується 1 бал. За вгадування зеро - 10 балів.\n\n2. Як потрапити в рейтинг?\nПросто грайте і заробляйте бали. У рейтинг потрапляють 100 найкращих гравців тижня.\n\n3. Як розподіляється призовий фонд?\nПризовий фонд розподіляється пропорційно кількості балів серед 100 найкращих гравців тижня.'),
    ('win', 'uk', 'Ви вгадали! Ставка: %s. Отримано балів: %d\n\nОберіть наступну ставку:'),
    ('win_zero', 'uk', 'Ви вгадали ЗЕРО! Отримано балів: %d\n\nОберіть наступну ставку:'),
    ('lose', 'uk', 'Ви не вгадали. Ваша ставка: %s. Випало: %s\n\nОберіть наступну ставку:'),
    ('zero_limit', 'uk', 'Ви ще не можете поставити на Zerо, яке може принести 10 балів в рейтинг. Залишилось сьогодні зробити ще %d ставок. До цього моменту випадення Zero зараховується для вас програшем'),
    ('bet_error', 'uk', 'Помилка при ставці. Спробуйте ще раз.'),
    ('error', 'uk', 'Сталася помилка. Спробуйте ще раз пізніше.'),
    ('btn_play', 'uk', '🎮 Грати'),
    ('btn_profile', 'uk', '👤 Профіль'),
    ('btn_stats', 'uk', '📊 Статистика'),
    ('btn_rating', 'uk', '🏆 Рейтинг'),
    ('btn_balance', 'uk', '💰 Баланс'),
    ('btn_faq', 'uk', '❓ FAQ'),
    ('btn_bet_red', 'uk', '🔴 Червоне'),
    ('btn_bet_black', 'uk', '⚫ Чорне'),
    ('btn_bet_zero', 'uk', '0️⃣ Зеро'),
    ('btn_bet_zero_locked', 'uk', '🔒 Зеро (заблоковано)'),
    ('btn_back', 'uk', '◀️ Назад');

-- Базові локалізації англійською
INSERT INTO localizations (key, language, value) VALUES
    ('welcome', 'en', 'Welcome to the Roulette Bot! Here you can bet on red, black or zero and compete for a place in the rating.'),
    ('help', 'en', 'Available commands:\n/play - Start the game\n/profile - Your profile\n/stats - Your statistics\n/rating - Weekly rating\n/superrating - Super rating\n/balance - Your balance\n/withdraw - Withdraw funds\n/faq - Frequently asked questions'),
    ('game_instructions', 'en', 'Choose your bet: red, black or zero. Each correct guess gives 1 point. Zero gives 10 points, but you can bet on it only after 100 bets per day.'),
    ('profile_template', 'en', 'Profile: %s\nBalance: %.2f\nTotal bets: %d\nWon bets: %d\nEfficiency: %.2f%%\nTotal points: %d'),
    ('stats_template', 'en', 'Statistics:\n\nDaily:\nBets: %d\nPoints: %d\n\nWeekly:\nBets: %d\nPoints: %d\n\nMonthly:\nBets: %d\nPoints: %d\n\nTotal:\nBets: %d\nPoints: %d'),
    ('rating_header', 'en', 'Weekly rating (top 10):\n\n'),
    ('super_rating_header', 'en', 'Super rating (top 10):\n\n'),
    ('balance_template', 'en', 'Your balance: %.2f'),
    ('withdraw_instructions', 'en', 'To withdraw funds, specify the amount and details. Minimum amount for withdrawal: 10.'),
    ('faq', 'en', 'Frequently asked questions:\n\n1. How are points awarded?\nFor each correct color guess, 1 point is awarded. For guessing zero - 10 points.\n\n2. How to get into the rating?\nJust play and earn points. The top 100 players of the week get into the rating.\n\n3. How is the prize fund distributed?\nThe prize fund is distributed in proportion to the number of points among the top 100 players of the week.'),
    ('win', 'en', 'You guessed it! Bet: %s. Points earned: %d\n\nChoose your next bet:'),
    ('win_zero', 'en', 'You guessed ZERO! Points earned: %d\n\nChoose your next bet:'),
    ('lose', 'en', 'You did not guess. Your bet: %s. Result: %s\n\nChoose your next bet:'),
    ('zero_limit', 'en', 'You cannot bet on Zero yet, which can bring 10 points to the rating. You need to make %d more bets today. Until then, if Zero comes up, it counts as a loss for you'),
    ('bet_error', 'en', 'Error when betting. Please try again.'),
    ('error', 'en', 'An error occurred. Please try again later.'),
    ('btn_play', 'en', '🎮 Play'),
    ('btn_profile', 'en', '👤 Profile'),
    ('btn_stats', 'en', '📊 Statistics'),
    ('btn_rating', 'en', '🏆 Rating'),
    ('btn_balance', 'en', '💰 Balance'),
    ('btn_faq', 'en', '❓ FAQ'),
    ('btn_bet_red', 'en', '🔴 Red'),
    ('btn_bet_black', 'en', '⚫ Black'),
    ('btn_bet_zero', 'en', '0️⃣ Zero'),
    ('btn_bet_zero_locked', 'en', '🔒 Zero (locked)'),
    ('btn_back', 'en', '◀️ Back');

-- Таблиця для хешів
CREATE TABLE IF NOT EXISTS hash_entries (
    id SERIAL PRIMARY KEY,
    number BIGINT NOT NULL,
    salt_hex VARCHAR(64) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hash_entries_hash ON hash_entries (hash);
