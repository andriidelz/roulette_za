-- migrations/001_initial_schema.sql

-- Пользователи
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

-- Статистика пользователя
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
    ('daily_bets_limit', '100', '100', 'Лимит ставок за день для возможности ставить на zero'),
    ('weekly_prize_amount', '1000', '1000', 'Сумма недельного призового фонда'),
    ('weekly_prize_top', '100', '100', 'Количество призовых мест в недельном рейтинге'),
    ('minimum_withdrawal', '10', '10', 'Минимальная сумма для вывода средств');

-- Локализации на украинском языке
INSERT INTO localizations (key, language, value) VALUES
    ('welcome', 'uk', 'Вітаємо у боті рулетки! Тут ви можете робити ставки на червоне, чорне або зеро і змагатися за місце в рейтингу.'),
    
    ('help', 'uk', 'Доступні команди:

/play - Почати гру
/profile - Ваш профіль
/stats - Ваша статистика
/rating - Тижневий рейтинг
/superrating - Супер рейтинг
/balance - Ваш баланс
/withdraw - Вивести кошти
/faq - Часті питання'),
    
    ('game_instructions', 'uk', 'Оберіть ставку: червоне, чорне або зеро.

Кожне вгадування дає 1 бал в рейтинг. Зеро дає 10 балів, але ставити на нього можна тільки після 100 ставок за день.'),
    
    ('profile_template', 'uk', 'Профіль: %s
Баланс: %.2f
Загальна кількість ставок: %d
Виграшних ставок: %d
Ефективність: %.2f%%
Загальна кількість балів: %d'),
    
    ('stats_template', 'uk', 'Статистика:

За день:
Ставок: %d
Балів: %d

За тиждень:
Ставок: %d
Балів: %d

За місяць:
Ставок: %d
Балів: %d

Загалом:
Ставок: %d
Балів: %d'),
    
    ('rating_header', 'uk', 'Тижневий рейтинг (топ-10):

'),
    ('super_rating_header', 'uk', 'Супер рейтинг (топ-10):

'),
    
    ('balance_template', 'uk', 'Ваш баланс: %.2f'),
    ('balanceaccok', 'uk', 'На вашому балансі %.2f USDT. Суми достатньо для виведення. Ви можете оформити виведення в пункті меню Виведення.'),
    ('balanceacclow', 'uk', 'На вашому балансі %.2f USDT. Суми недостатньо для виведення. Грайте більше, щоб увійти до топ гравців тижня та розподілити призовий фонд!'),
    
    ('withdraw_instructions', 'uk', 'Для виведення коштів вкажіть суму і реквізити. Мінімальна сума для виведення: 10.'),
    ('withdrawok', 'uk', 'На вашому балансі %.2f USDT. Ви можете оформити виведення, продовживши.'),
    ('withdrawlow', 'uk', 'На вашому балансі %.2f USDT. Суми недостатньо для виведення. Грайте більше, щоб увійти до топ гравців тижня та розподілити призовий фонд!'),
    
    ('faq', 'uk', 'Часті питання:

1. Як нараховуються бали?
За кожне вгадування кольору нараховується 1 бал. За вгадування зеро - 10 балів.

2. Як потрапити в рейтинг?
Просто грайте і заробляйте бали. У рейтинг потрапляють 100 найкращих гравців тижня.

3. Як розподіляється призовий фонд?
Призовий фонд розподіляється пропорційно кількості балів серед 100 найкращих гравців тижня.'),
    
    ('win', 'uk', 'Ви вгадали! Ставка: %s. Отримано балів: %d

Оберіть наступну ставку:'),
    ('win_zero', 'uk', 'Ви вгадали ЗЕРО! Отримано балів: %d

Оберіть наступну ставку:'),
    ('lose', 'uk', 'Ви не вгадали. Ваша ставка: %s. Випало: %s

Оберіть наступну ставку:'),
    
    ('zero_limit', 'uk', 'Ви ще не можете поставити на Zerо, яке може принести 10 балів в рейтинг. Залишилось сьогодні зробити ще %d ставок. До цього моменту випадення Zero зараховується для вас програшем'),
    
    ('nomorebids', 'uk', 'Ставки прийняті! Ставок більше немає!'),
    ('nextbid15', 'uk', 'Раунд #%s
До наступного визначення ставки залишилось 15 секунд.

Зробіть свій вибір'),
    ('nextbid5', 'uk', 'Раунд #%s
До наступного визначення ставки залишилось 5 секунд.

Зробіть свій вибір'),
    
    ('blackresult', 'uk', 'На рулетці чорне!'),
    ('redresult', 'uk', 'На рулетці червоне!'),
    ('zeroresult', 'uk', 'На рулетці Zero!'),
    
    ('bet_error', 'uk', 'Помилка при ставці. Спробуйте ще раз.'),
    ('error', 'uk', 'Сталася помилка. Спробуйте ще раз пізніше.'),
    ('bet_already_made', 'uk', 'Ви вже зробили ставку в цьому раунді. Дочекайтеся результату.'),
    
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
    ('btn_back', 'uk', '◀️ Назад'),
    
    ('round_info', 'uk', 'Раунд #%s\nХеш: %s'),
    ('verification_info', 'uk', 'Перевірка результату:\nРаунд #%s\nЧисло: %d\nСіль: %s\nХеш: %s'),
    ('new_round', 'uk', 'Почався новий раунд #%s\nХеш: %s\n\nЗробіть вашу ставку:');

-- Локализации на английском языке
INSERT INTO localizations (key, language, value) VALUES
    ('welcome', 'en', 'Welcome to the Roulette Bot! Here you can bet on red, black or zero and compete for a place in the rating.'),
    
    ('help', 'en', 'Available commands:

/play - Start the game
/profile - Your profile
/stats - Your statistics
/rating - Weekly rating
/superrating - Super rating
/balance - Your balance
/withdraw - Withdraw funds
/faq - Frequently asked questions'),
    
    ('game_instructions', 'en', 'Choose your bet: red, black or zero.

Each correct guess gives 1 point. Zero gives 10 points, but you can bet on it only after 100 bets per day.'),
    
    ('profile_template', 'en', 'Profile: %s
Balance: %.2f
Total bets: %d
Won bets: %d
Efficiency: %.2f%%
Total points: %d'),
    
    ('stats_template', 'en', 'Statistics:

Daily:
Bets: %d
Points: %d

Weekly:
Bets: %d
Points: %d

Monthly:
Bets: %d
Points: %d

Total:
Bets: %d
Points: %d'),
    
    ('rating_header', 'en', 'Weekly rating (top 10):

'),
    ('super_rating_header', 'en', 'Super rating (top 10):

'),
    
    ('balance_template', 'en', 'Your balance: %.2f'),
    ('balanceaccok', 'en', 'Your balance is %.2f USDT. The amount is sufficient for withdrawal. You can make a withdrawal in the Withdraw menu item.'),
    ('balanceacclow', 'en', 'Your balance is %.2f USDT. The amount is insufficient for withdrawal. Play more to enter the top players of the week and distribute the prize fund!'),
    
    ('withdraw_instructions', 'en', 'To withdraw funds, specify the amount and details. Minimum amount for withdrawal: 10.'),
    ('withdrawok', 'en', 'Your balance is %.2f USDT. You can make a withdrawal by continuing.'),
    ('withdrawlow', 'en', 'Your balance is %.2f USDT. The amount is insufficient for withdrawal. Play more to enter the top players of the week and distribute the prize fund!'),
    
    ('faq', 'en', 'Frequently asked questions:

1. How are points awarded?
For each correct color guess, 1 point is awarded. For guessing zero - 10 points.

2. How to get into the rating?
Just play and earn points. The top 100 players of the week get into the rating.

3. How is the prize fund distributed?
The prize fund is distributed in proportion to the number of points among the top 100 players of the week.'),
    
    ('win', 'en', 'You guessed it! Bet: %s. Points earned: %d

Choose your next bet:'),
    ('win_zero', 'en', 'You guessed ZERO! Points earned: %d

Choose your next bet:'),
    ('lose', 'en', 'You did not guess. Your bet: %s. Result: %s

Choose your next bet:'),
    
    ('zero_limit', 'en', 'You cannot bet on Zero yet, which can bring 10 points to the rating. You need to make %d more bets today. Until then, if Zero comes up, it counts as a loss for you'),
    
    ('nomorebids', 'en', 'Bets accepted! No more bets!'),
    ('nextbid15', 'en', 'Round #%s
There are 15 seconds left until the next bet determination.

Make your choice'),
    ('nextbid5', 'en', 'Round #%s
There are 5 seconds left until the next bet determination.

Make your choice'),
    
    ('blackresult', 'en', 'Black on the roulette!'),
    ('redresult', 'en', 'Red on the roulette!'),
    ('zeroresult', 'en', 'Zero on the roulette!'),
    
    ('bet_error', 'en', 'Error when betting. Please try again.'),
    ('error', 'en', 'An error occurred. Please try again later.'),
    ('bet_already_made', 'en', 'You have already made a bet in this round. Please wait for the result.'),
    
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
    ('btn_back', 'en', '◀️ Back'),
    
    ('round_info', 'en', 'Round #%s\nHash: %s'),
    ('verification_info', 'en', 'Result verification:\nRound #%s\nNumber: %d\nSalt: %s\nHash: %s'),
    ('new_round', 'en', 'New round #%s started\nHash: %s\n\nMake your bet:');

-- Локализации на русском языке
INSERT INTO localizations (key, language, value) VALUES
    ('welcome', 'ru', 'Добро пожаловать в бот рулетки! Здесь вы можете делать ставки на красное, черное или зеро и соревноваться за место в рейтинге.'),
    
    ('help', 'ru', 'Доступные команды:

/play - Начать игру
/profile - Ваш профиль
/stats - Ваша статистика
/rating - Недельный рейтинг
/superrating - Супер рейтинг
/balance - Ваш баланс
/withdraw - Вывод средств
/faq - Частые вопросы'),
    
    ('game_instructions', 'ru', 'Выберите ставку: красное, черное или зеро.

Каждое угадывание дает 1 балл в рейтинг. Зеро дает 10 баллов, но ставить на него можно только после 100 ставок за день.'),
    
    ('profile_template', 'ru', 'Профиль: %s
Баланс: %.2f
Общее количество ставок: %d
Выигрышных ставок: %d
Эффективность: %.2f%%
Общее количество баллов: %d'),
    
    ('stats_template', 'ru', 'Статистика:

За день:
Ставок: %d
Баллов: %d

За неделю:
Ставок: %d
Баллов: %d

За месяц:
Ставок: %d
Баллов: %d

Всего:
Ставок: %d
Баллов: %d'),
    
    ('rating_header', 'ru', 'Недельный рейтинг (топ-10):

'),
    ('super_rating_header', 'ru', 'Супер рейтинг (топ-10):

'),
    
    ('balance_template', 'ru', 'Ваш баланс: %.2f'),
    ('balanceaccok', 'ru', 'На вашем балансе %.2f USDT. Суммы достаточно для вывода. Вы можете оформить вывод в пункте меню Вывод средств.'),
    ('balanceacclow', 'ru', 'На вашем балансе %.2f USDT. Суммы недостаточно для вывода. Играйте больше, чтобы войти в топ игроков недели и распределить призовой фонд!'),
    
    ('withdraw_instructions', 'ru', 'Для вывода средств укажите сумму и реквизиты. Минимальная сумма для вывода: 10.'),
    ('withdrawok', 'ru', 'На вашем балансе %.2f USDT. Вы можете оформить вывод, продолжив.'),
    ('withdrawlow', 'ru', 'На вашем балансе %.2f USDT. Суммы недостаточно для вывода. Играйте больше, чтобы войти в топ игроков недели и распределить призовой фонд!'),
    
    ('faq', 'ru', 'Частые вопросы:

1. Как начисляются баллы?
За каждое угадывание цвета начисляется 1 балл. За угадывание зеро - 10 баллов.

2. Как попасть в рейтинг?
Просто играйте и зарабатывайте баллы. В рейтинг попадают 100 лучших игроков недели.

3. Как распределяется призовой фонд?
Призовой фонд распределяется пропорционально количеству баллов среди 100 лучших игроков недели.'),
    
    ('win', 'ru', 'Вы угадали! Ставка: %s. Получено баллов: %d

Выберите следующую ставку:'),
    ('win_zero', 'ru', 'Вы угадали ЗЕРО! Получено баллов: %d

Выберите следующую ставку:'),
    ('lose', 'ru', 'Вы не угадали. Ваша ставка: %s. Выпало: %s

Выберите следующую ставку:'),
    
    ('zero_limit', 'ru', 'Вы еще не можете поставить на Зеро, которое может принести 10 баллов в рейтинг. Осталось сегодня сделать еще %d ставок. До этого момента выпадение Zero засчитывается для вас проигрышем'),
    
    ('nomorebids', 'ru', 'Ставки приняты! Ставок больше нет!'),
    ('nextbid15', 'ru', 'Раунд #%s
До следующего определения ставки осталось 15 секунд.

Сделайте свой выбор'),
    ('nextbid5', 'ru', 'Раунд #%s
До следующего определения ставки осталось 5 секунд.

Сделайте свой выбор'),
    
    ('blackresult', 'ru', 'На рулетке черное!'),
    ('redresult', 'ru', 'На рулетке красное!'),
    ('zeroresult', 'ru', 'На рулетке Zero!'),
    
    ('bet_error', 'ru', 'Ошибка при ставке. Попробуйте еще раз.'),
    ('error', 'ru', 'Произошла ошибка. Попробуйте еще раз позже.'),
    ('bet_already_made', 'ru', 'Вы уже сделали ставку в этом раунде. Дождитесь результата.'),
    
    ('btn_play', 'ru', '🎮 Играть'),
    ('btn_profile', 'ru', '👤 Профиль'),
    ('btn_stats', 'ru', '📊 Статистика'),
    ('btn_rating', 'ru', '🏆 Рейтинг'),
    ('btn_balance', 'ru', '💰 Баланс'),
    ('btn_faq', 'ru', '❓ FAQ'),
    ('btn_bet_red', 'ru', '🔴 Красное'),
    ('btn_bet_black', 'ru', '⚫ Черное'),
    ('btn_bet_zero', 'ru', '0️⃣ Зеро'),
    ('btn_bet_zero_locked', 'ru', '🔒 Зеро (заблокировано)'),
    ('btn_back', 'ru', '◀️ Назад'),
    
    ('round_info', 'ru', 'Раунд #%s\nХеш: %s'),
    ('verification_info', 'ru', 'Проверка результата:\nРаунд #%s\nЧисло: %d\nСоль: %s\nХеш: %s'),
    ('new_round', 'ru', 'Начался новый раунд #%s\nХеш: %s\n\nСделайте вашу ставку:');
