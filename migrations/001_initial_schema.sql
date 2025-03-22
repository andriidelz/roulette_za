-- migrations/001_initial_schema.sql

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
    ('daily_bets_limit', '2880', '2880', 'Лимит ставок за день'),
    ('daily_bets_zero_limit', '100', '100', 'Лимит ставок за день для возможности ставить на zero'),
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
    ('btn_bet_black', 'uk', '⚫️ Чорне'),
    ('btn_bet_zero', 'uk', '🟢 Zero'),
    ('btn_bet_zero_locked', 'uk', '🔒 Zero'),
    ('btn_back', 'uk', '◀️ Назад'),
    
    ('round_info', 'uk', 'Раунд #%s

Хеш: %s'),
    ('verification_info', 'uk', 'Перевірка результату:

Раунд #%s
Число: %d
Сіль: %s
Хеш: %s'),
    ('new_round', 'uk', 'Почався новий раунд #%s

Хеш: %s

Зробіть вашу ставку:'),
    
    ('startmessage1', 'uk', 'Вітаємо вас у Sprut Red&Black bot

Це єдиний бот, де ви можете щотижня вигравати реальні гроші за віртуальні вгадування.'),
    ('countrymes', 'uk', 'Для продовження виберіть, будь ласка, країну свого проживання. Це необхідно нам для майбутньої локалізації рейтингів гравців і збільшення ваших шансів на виграш'),
    ('btn_rules', 'uk', 'Правила'),
    ('btn_awards', 'uk', 'Нагороди'),
    ('btn_payments', 'uk', 'Платежі'),
    ('btn_fairplay', 'uk', 'Чесна гра'),
    ('btn_statistics', 'uk', 'Статистика'),
    ('btn_account', 'uk', 'Акаунт'),
    ('rules', 'uk', 'Правила гри:

1. Ви робите ставку на червоне, чорне або зеро.
2. Кожні 30 секунд визначається випадкове число від 0 до 36.
3. Числа від 1 до 36 відповідають червоному або чорному.
4. 0 відповідає зеро.
5. За кожне вірне передбачення ви отримуєте бали.
6. Ваші бали враховуються в щотижневому рейтингу.'),
    ('awards', 'uk', 'Нагороди:

1. Щотижня формується рейтинг гравців.
2. Топ-100 гравців розділяють призовий фонд.
3. Призовий фонд розподіляється пропорційно набраним балам.
4. Виплати відбуваються автоматично щопонеділка.
5. Мінімальна сума для виведення коштів: 10 USDT.'),
    ('payments', 'uk', 'Платежі:

1. Всі призові виплачуються в USDT (TRC-20).
2. Для отримання виплати необхідно вказати адресу гаманця.
3. Мінімальна сума для виведення: 10 USDT.
4. Обробка запитів на виведення займає до 24 годин.'),
    ('fairplay', 'uk', 'Чесна гра:

Результати рулетки гарантовано чесні та перевірені. Перед кожним раундом публікується хеш результату. Після раунду ви можете перевірити, що результат не був змінений.

Для перевірки:
1. Візьміть число і сіль, надані ботом.
2. Сформуйте рядок у форматі: [число]:[сіль]
3. Обчисліть SHA-256 хеш від цього рядка.
4. Порівняйте з хешем з початку раунду.');

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
    ('btn_bet_black', 'en', '⚫️ Black'),
    ('btn_bet_zero', 'en', '🟢 Zero'),
    ('btn_bet_zero_locked', 'en', '🔒 Zero'),
    ('btn_back', 'en', '◀️ Back'),
    
    ('round_info', 'en', 'Round #%s

Hash: %s'),
    ('verification_info', 'en', 'Result verification:

Round #%s
Number: %d
Salt: %s
Hash: %s'),
    ('new_round', 'en', 'New round #%s started

Hash: %s

Make your bet:'),
    
    ('startmessage1', 'en', 'Welcome to Sprut Red&Black bot

This is the only bot where you can win real money every week for virtual guesses.'),
    ('countrymes', 'en', 'To continue, please select your country of residence. This is necessary for the future localization of player ratings and increasing your chances of winning'),
    ('btn_rules', 'en', 'Rules'),
    ('btn_awards', 'en', 'Rewards'),
    ('btn_payments', 'en', 'Payments'),
    ('btn_fairplay', 'en', 'Fair Play'),
    ('btn_statistics', 'en', 'Statistics'),
    ('btn_account', 'en', 'Account'),
    ('rules', 'en', 'Game Rules:

1. You place a bet on red, black, or zero.
2. Every 30 seconds, a random number from 0 to 36 is determined.
3. Numbers from 1 to 36 correspond to red or black.
4. 0 corresponds to zero.
5. For each correct prediction, you earn points.
6. Your points are counted in the weekly ranking.'),
    ('awards', 'en', 'Rewards:

1. A player ranking is formed every week.
2. The top 100 players share the prize pool.
3. The prize pool is distributed in proportion to the points earned.
4. Payments are processed automatically every Monday.
5. Minimum withdrawal amount: 10 USDT.'),
    ('payments', 'en', 'Payments:

1. All prizes are paid in USDT (TRC-20).
2. To receive payment, you need to specify your wallet address.
3. Minimum withdrawal amount: 10 USDT.
4. Withdrawal requests are processed within 24 hours.'),
    ('fairplay', 'en', 'Fair Play:

Roulette results are guaranteed to be fair and verifiable. Before each round, a hash of the result is published. After the round, you can verify that the result was not changed.

To verify:
1. Take the number and salt provided by the bot.
2. Form a string in the format: [number]:[salt]
3. Calculate the SHA-256 hash of this string.
4. Compare with the hash from the beginning of the round.');

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
    ('btn_bet_black', 'ru', '⚫️ Черное'),
    ('btn_bet_zero', 'ru', '🟢 Zero'),
    ('btn_bet_zero_locked', 'ru', '🔒 Zero'),
    ('btn_back', 'ru', '◀️ Назад'),
    
    ('round_info', 'ru', 'Раунд #%s

Хеш: %s'),
    ('verification_info', 'ru', 'Проверка результата:

Раунд #%s
Число: %d
Соль: %s
Хеш: %s'),
    ('new_round', 'ru', 'Начался новый раунд #%s

Хеш: %s

Сделайте вашу ставку:'),
    
    ('startmessage1', 'ru', 'Приветствуем вас в  Sprut Red&Black bot

Это единственный бот где вы можете еженедельно выиграть реальные деньги за виртуальные угадывания.'),
    ('countrymes', 'ru', 'Для продолжения выбери пожалуйста страну своего проживания. Это необходимо нам для будущей локализации рейтингов игроков и увеличения твоих шансов на выигрыш'),
    ('btn_rules', 'ru', 'Правила'),
    ('btn_awards', 'ru', 'Награды'),
    ('btn_payments', 'ru', 'Платежи'),
    ('btn_fairplay', 'ru', 'Честная игра'),
    ('btn_statistics', 'ru', 'Статистика'),
    ('btn_account', 'ru', 'Аккаунт'),
    ('rules', 'ru', 'Правила игры:

1. Вы делаете ставку на красное, черное или зеро.
2. Каждые 30 секунд определяется случайное число от 0 до 36.
3. Числа от 1 до 36 соответствуют красному или черному.
4. 0 соответствует зеро.
5. За каждое верное предсказание вы получаете баллы.
6. Ваши баллы учитываются в еженедельном рейтинге.'),
    ('awards', 'ru', 'Награды:

1. Каждую неделю формируется рейтинг игроков.
2. Топ-100 игроков разделяют призовой фонд.
3. Призовой фонд распределяется пропорционально набранным баллам.
4. Выплаты происходят автоматически каждый понедельник.
5. Минимальная сумма для вывода средств: 10 USDT.'),
    ('payments', 'ru', 'Платежи:

1. Все призовые выплачиваются в USDT (TRC-20).
2. Для получения выплаты необходимо указать адрес кошелька.
3. Минимальная сумма для вывода: 10 USDT.
4. Обработка запросов на вывод занимает до 24 часов.'),
    ('fairplay', 'ru', 'Честная игра:

Результаты рулетки гарантированно честные и проверяемые. Перед каждым раундом публикуется хеш результата. После раунда вы можете проверить, что результат не был изменен.

Для проверки:
1. Возьмите число и соль, предоставленные ботом.
2. Сформируйте строку в формате: [число]:[соль]
3. Вычислите SHA-256 хеш от этой строки.
4. Сравните с хешем из начала раунда.');

-- Дополнительные локализации для всех языков
INSERT INTO localizations (key, language, value) VALUES 
    ('main_menu', 'en', 'Tap a button to continue');

INSERT INTO localizations (key, language, value) VALUES 
    ('main_menu', 'ru', 'Нажмите кнопку для продолжения');

INSERT INTO localizations (key, language, value) VALUES 
    ('main_menu', 'uk', 'Натисніть кнопку для продовження');

INSERT INTO localizations (key, language, value) VALUES 
    ('country_saved', 'en', 'Your country has been successfully saved! ✅');

INSERT INTO localizations (key, language, value) VALUES 
    ('country_saved', 'ru', 'Ваша страна успешно сохранена! ✅');

INSERT INTO localizations (key, language, value) VALUES 
    ('country_saved', 'uk', 'Ваша країна успішно збережена! ✅');

-- Настройки профиля
INSERT INTO localizations (key, language, value) VALUES
    ('settings_message', 'en', 'Settings Menu\n\nHere you can change your profile settings:'),
    ('btn_settings_language', 'en', '🌐 Language'),
    ('btn_settings_country', 'en', '🌍 Country'),
    ('btn_settings_name', 'en', '👤 First Name'),
    ('btn_settings_lastname', 'en', '👥 Last Name'),
    ('btn_back_to_main', 'en', '◀️ Back to Main Menu'),
    ('settings_language', 'en', 'Select your language:'),
    ('settings_name', 'en', 'Enter your first name:'),
    ('settings_lastname', 'en', 'Enter your last name:'),
    ('language_saved', 'en', 'Language successfully updated! ✅'),
    ('name_saved', 'en', 'First name successfully updated! ✅'),
    ('lastname_saved', 'en', 'Last name successfully updated! ✅');

INSERT INTO localizations (key, language, value) VALUES
    ('settings_message', 'ru', 'Меню настроек\n\nЗдесь вы можете изменить настройки профиля:'),
    ('btn_settings_language', 'ru', '🌐 Язык'),
    ('btn_settings_country', 'ru', '🌍 Страна'),
    ('btn_settings_name', 'ru', '👤 Имя'),
    ('btn_settings_lastname', 'ru', '👥 Фамилия'),
    ('btn_back_to_main', 'ru', '◀️ Вернуться в главное меню'),
    ('settings_language', 'ru', 'Выберите ваш язык:'),
    ('settings_name', 'ru', 'Введите ваше имя:'),
    ('settings_lastname', 'ru', 'Введите вашу фамилию:'),
    ('language_saved', 'ru', 'Язык успешно обновлен! ✅'),
    ('name_saved', 'ru', 'Имя успешно обновлено! ✅'),
    ('lastname_saved', 'ru', 'Фамилия успешно обновлена! ✅');

INSERT INTO localizations (key, language, value) VALUES
    ('settings_message', 'uk', 'Меню налаштувань\n\nТут ви можете змінити налаштування профілю:'),
    ('btn_settings_language', 'uk', '🌐 Мова'),
    ('btn_settings_country', 'uk', '🌍 Країна'),
    ('btn_settings_name', 'uk', '👤 Ім''я'),
    ('btn_settings_lastname', 'uk', '👥 Прізвище'),
    ('btn_back_to_main', 'uk', '◀️ Повернутися до головного меню'),
    ('settings_language', 'uk', 'Виберіть вашу мову:'),
    ('settings_name', 'uk', 'Введіть ваше ім''я'),
    ('settings_lastname', 'uk', 'Введіть ваше прізвище:'),
    ('language_saved', 'uk', 'Мову успішно оновлено! ✅'),
    ('name_saved', 'uk', 'Ім''я успішно оновлено! ✅'),
    ('lastname_saved', 'uk', 'Прізвище успішно оновлено! ✅');

-- Настройки кошелька
INSERT INTO localizations (key, language, value) VALUES
    ('btn_settings_wallet', 'en', '💰 USDT Wallet Address'),
    ('settings_wallet', 'en', 'Enter your USDT wallet address (TRC20):'),
    ('wallet_saved', 'en', 'Wallet address successfully updated! ✅'),
    ('invalid_wallet_format', 'en', 'Invalid wallet address format. Please enter a valid TRC20 wallet address starting with T.');

INSERT INTO localizations (key, language, value) VALUES
    ('btn_settings_wallet', 'ru', '💰 Адрес кошелька USDT'),
    ('settings_wallet', 'ru', 'Введите адрес вашего кошелька USDT (TRC20):'),
    ('wallet_saved', 'ru', 'Адрес кошелька успешно обновлен! ✅'),
    ('invalid_wallet_format', 'ru', 'Неверный формат адреса кошелька. Пожалуйста, введите действительный адрес кошелька TRC20, начинающийся с T.');

INSERT INTO localizations (key, language, value) VALUES
    ('btn_settings_wallet', 'uk', '💰 Адреса гаманця USDT'),
    ('settings_wallet', 'uk', 'Введіть адресу вашого гаманця USDT (TRC20):'),
    ('wallet_saved', 'uk', 'Адреса гаманця успішно оновлена! ✅'),
    ('invalid_wallet_format', 'uk', 'Невірний формат адреси гаманця. Будь ласка, введіть дійсну адресу гаманця TRC20, що починається з T.');

-- Статус и сообщения об ошибках
INSERT INTO localizations (key, language, value) VALUES
    ('unknown_command', 'en', 'Unknown command. Please use the menu to navigate.'),
    ('unknown_command', 'ru', 'Неизвестная команда. Пожалуйста, используйте меню для навигации.'),
    ('unknown_command', 'uk', 'Невідома команда. Будь ласка, використовуйте меню для навігації.');

-- Добавление локализаций для статистики (русский язык)
INSERT INTO localizations (key, language, value) 
VALUES 
('statisticsstart', 'ru', 'В данном разделе доступна ваша персональная статистика игры. Выберите период, за который вы хотите просмотреть статистику'),
('daystat', 'ru', 'Статистика за сегодня'),
('weekstat', 'ru', 'Статистика за неделю'),
('monthstat', 'ru', 'Статистика за месяц'),
('allstat', 'ru', 'Статистика за все время'),
('exitstat', 'ru', 'Вернуться в меню'),
('daystatm', 'ru', E'Ваша статистика за сутки\nСделано %d ставок (%d черное, %d красное, %d ZERO)\nиз них\nВыиграно %d ставки (%d черное, %d красное, %d ZERO)\nПроиграно %d ставки (%d черное, %d красное, %d ZERO)\n\nВы заработали %d рейтинговых балла'),
('weekstatm', 'ru', E'Ваша статистика за текущую неделю\nСделано %d ставок (%d черное, %d красное, %d ZERO)\nиз них\nВыиграно %d ставки (%d черное, %d красное, %d ZERO)\nПроиграно %d ставки (%d черное, %d красное, %d ZERO)\n\nВы заработали %d рейтинговых балла'),
('monthstatm', 'ru', E'Ваша статистика за текущий месяц\nСделано %d ставок (%d черное, %d красное, %d ZERO)\nиз них\nВыиграно %d ставки (%d черное, %d красное, %d ZERO)\nПроиграно %d ставки (%d черное, %d красное, %d ZERO)\n\nВы заработали %d рейтинговых балла'),
('allstatm', 'ru', E'Ваша статистика за все время\nСделано %d ставок (%d черное, %d красное, %d ZERO)\nиз них\nВыиграно %d ставки (%d черное, %d красное, %d ZERO)\nПроиграно %d ставки (%d черное, %d красное, %d ZERO)\n\nВы заработали %d рейтинговых балла'),
('statistics next', 'ru', 'Выберите другой временной период для получения статистики или вернитесь в главное меню');

-- Добавление локализаций для статистики (английский язык)
INSERT INTO localizations (key, language, value) 
VALUES 
('statisticsstart', 'en', 'Your personal game statistics are available in this section. Select the period for which you want to view statistics'),
('daystat', 'en', 'Today''s statistics'),
('weekstat', 'en', 'Weekly statistics'),
('monthstat', 'en', 'Monthly statistics'),
('allstat', 'en', 'All-time statistics'),
('exitstat', 'en', 'Return to menu'),
('daystatm', 'en', E'Your statistics for today\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
('weekstatm', 'en', E'Your statistics for the current week\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
('monthstatm', 'en', E'Your statistics for the current month\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
('allstatm', 'en', E'Your statistics for all time\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
('statistics next', 'en', 'Select another time period to get statistics or return to the main menu');

-- Добавление локализаций для статистики (украинский язык)
INSERT INTO localizations (key, language, value) 
VALUES 
('statisticsstart', 'uk', 'У цьому розділі доступна ваша персональна статистика гри. Виберіть період, за який ви хочете переглянути статистику'),
('daystat', 'uk', 'Статистика за сьогодні'),
('weekstat', 'uk', 'Статистика за тиждень'),
('monthstat', 'uk', 'Статистика за місяць'),
('allstat', 'uk', 'Статистика за весь час'),
('exitstat', 'uk', 'Повернутися в меню'),
('daystatm', 'uk', E'Ваша статистика за добу\nЗроблено %d ставок (%d чорне, %d червоне, %d ZERO)\nз них\nВиграно %d ставки (%d чорне, %d червоне, %d ZERO)\nПрограно %d ставки (%d чорне, %d червоне, %d ZERO)\n\nВи заробили %d рейтингових балів'),
('weekstatm', 'uk', E'Ваша статистика за поточний тиждень\nЗроблено %d ставок (%d чорне, %d червоне, %d ZERO)\nз них\nВиграно %d ставки (%d чорне, %d червоне, %d ZERO)\nПрограно %d ставки (%d чорне, %d червоне, %d ZERO)\n\nВи заробили %d рейтингових балів'),
('monthstatm', 'uk', E'Ваша статистика за поточний місяць\nЗроблено %d ставок (%d чорне, %d червоне, %d ZERO)\nз них\nВиграно %d ставки (%d чорне, %d червоне, %d ZERO)\nПрограно %d ставки (%d чорне, %d червоне, %d ZERO)\n\nВи заробили %d рейтингових балів'),
('allstatm', 'uk', E'Ваша статистика за весь час\nЗроблено %d ставок (%d чорне, %d червоне, %d ZERO)\nз них\nВиграно %d ставки (%d чорне, %d червоне, %d ZERO)\nПрограно %d ставки (%d чорне, %d червоне, %d ZERO)\n\nВи заробили %d рейтингових балів'),
('statistics next', 'uk', 'Виберіть інший часовий період для отримання статистики або поверніться в головне меню');

-- Добавление новых локализаций для игры
INSERT INTO localizations (key, language, value) VALUES
    ('playstart1', 'ru', E'Суть игры\nВаша задача угадать цвет поля, которое выпадет на виртуальной рулетке.\n\nЗа каждое правильное угадывание вы получаете 1 зачетный бал...'),
    ('playstart1', 'en', E'Game essence\nYour task is to guess the color of the field that will appear on the virtual roulette.\n\nFor each correct guess you get 1 credit point...'),
    ('playstart1', 'uk', E'Суть гри\nВаше завдання вгадати колір поля, яке випаде на віртуальній рулетці.\n\nЗа кожне правильне вгадування ви отримуєте 1 заліковий бал...'),
    
    ('rulesstart', 'ru', 'Детальные правила'),
    ('rulesstart', 'en', 'Detailed Rules'),
    ('rulesstart', 'uk', 'Детальні правила'),
    
    ('availablebets', 'ru', 'Доступно ставок'),
    ('availablebets', 'en', 'Available bets'),
    ('availablebets', 'uk', 'Доступно ставок'),
    
    ('stop', 'ru', 'Стоп игра'),
    ('stop', 'en', 'Stop game'),
    ('stop', 'uk', 'Стоп гра'),
    
    ('betsbalancelow', 'ru', 'У вас закончились ставки на сегодня. Приходите завтра!'),
    ('betsbalancelow', 'en', 'You have run out of bets for today. Come back tomorrow!'),
    ('betsbalancelow', 'uk', 'У вас закінчилися ставки на сьогодні. Приходьте завтра!'),
    
    ('betsbalanceok', 'ru', 'У вас есть еще %d ставок на сегодня.'),
    ('betsbalanceok', 'en', 'You have %d more bets available today.'),
    ('betsbalanceok', 'uk', 'У вас є ще %d ставок на сьогодні.'),
    
    ('round_info_countdown', 'ru', E'Раунд #%s\nХеш: %s\n\nДо следующего определения ставки осталось %d секунд.\n\nСделайте свой выбор'),
    ('round_info_countdown', 'en', E'Round #%s\nHash: %s\n\n%d seconds left until the next bet determination.\n\nMake your choice'),
    ('round_info_countdown', 'uk', E'Раунд #%s\nХеш: %s\n\nДо наступного визначення ставки залишилось %d секунд.\n\nЗробіть свій вибір'),
    ('waiting_for_round', 'ru', 'Ожидаем начала нового раунда. Пожалуйста, подождите...'),
    ('waiting_for_round', 'en', 'Waiting for a new round to start. Please wait...'),
    ('waiting_for_round', 'uk', 'Очікуємо початку нового раунду. Будь ласка, зачекайте...');