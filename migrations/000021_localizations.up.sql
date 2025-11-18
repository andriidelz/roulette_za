ALTER TABLE bets
ADD COLUMN IF NOT EXISTS bet_point INT DEFAULT 0;

INSERT INTO localizations (key, language, value) VALUES
    ('bet_boost_prompt_msg', 'ru', '🎰 Твоя ставка в этом раунде: <b>0</b> рейтинговых баллов

Если ставка сыграет – получишь <b>1</b> рейтинговый балл 💎

📈 Можешь повысить ставку, используя заработанные рейтинговые баллы.

Как это работает:
• 🔴 / ⚫️ — <b>х2</b> от поставленных баллов
• 🟢 Zero — <b>х35</b>
• ❌ Промах — <b>теряешь всю ставку</b>

💬 <b>Сколько баллов хочешь поставить этот раунд?</b>'),
    ('bet_adjust_prompt_msg', 'ru', '🎰 Твоя текущая ставка: <b>%d</b> баллов

Если угадаешь — умножаешь свою ставку:
• 🔴 / ⚫️ — <b>х2</b>,
• 🟢 Zero — <b>х35</b>,
• ❌ Промах — <b>0 баллов</b>

🔧 Хочешь подкорректировать ставку?
Выбери, как именно:
• X2 – удвоить ставку
• +5 / +10 / +15 / +20 - добавить баллы
• ✍️ Указать кол-во баллов — установить вручную'),
    ('btn_betboost_1', 'ru', '+1'),
    ('btn_betboost_2', 'ru', '+2'),
    ('btn_betboost_5', 'ru', '+5'),
    ('btn_betboost_10', 'ru', '+10'),
    ('btn_betboost_15', 'ru', '+15'),
    ('btn_betboost_20', 'ru', '+20'),
    ('btn_betboost_manual', 'ru', '✍️ Указать кол-во баллов'),
    ('btn_betboost_skip', 'ru', '🚫 Не повышать'),
    ('ow_rating_balance_msg', 'ru', 'Недостаточно рейтинговых баллов для этой ставки 😶
Играй в раундах, зарабатывай баллы и возвращайся по повышенной ставке 💪🔥'),
    ('enter_bet_amount', 'ru', 'Введи размер ставки цифрой. 
❗️Ставка не может быть больше количества твоих заработанных баллов.
⚠️ Минимум - 1 балл'),
    ('enter_bet_mult', 'ru', 'Укажи кол-во баллов цифрой.
❗️Ставка не может быть больше количества твоих заработанных баллов.
⚠️ Минимум - 1 балл'),
    ('btn_mult_x2', 'ru', 'X2'),
    ('btn_reset_bet', 'ru', '❌ Скинути ставку до 0'),
    ('btn_keep_bet', 'ru', '🚫 Не менять'),
    ('input_invalid_msg', 'ru', '🚫Похоже, ты ввел некорректные данные.
👉 Попробуй еще раз'),
    ('input_invalid2_msg', 'ru', '🚫Похоже, ты опять ввел некорректные данные.
👇 Нажмите кнопку ниже, чтобы начать новый раунд'),
    ('input_invalid_zero_msg', 'ru', '⚠️ Неверное значение. Минимум - 1 балл
👉 Попробуй еще раз'),
    ('btn_confirm_bet', 'ru', '✅ Подтвердить ставку'),
    ('nextbid15_boost', 'ru', '🎰 Раунд #%s
🔒 Хеш:%s
⏳ Сделай свой выбор, до определения ставки осталось <b>%d</b> секунд.
Размер ставки в баллах: <b>%d</b>'),
    ('nomorebids_boost', 'ru', 'Твоя ставка %d баллов принята ✅Ставок больше нет!'),
    ('btn_bet_boost_info', 'ru', '📝 Как работает система повышения ставок'),
    ('bet_boost_info_msg', 'ru', '📌 На старте каждый игрок имеет 0 рейтинговых баллов.
📌 За выигрыши в раундах ты получаешь рейтинговые баллы.
📌 Когда накопишь достаточно – сможешь менять свою ставку:
• либо добавлять фиксированное количество баллов (+5, +10, +15, +20);
• или умножать текущую ставку (×2);
• или указать точное количество баллов вручную.

🎯 Чем выше ставка — тем больше потенциальный выигрыш.
⚠️ Но помни: проигрыш забирает все поставленные баллы.

💪 Играй активнее, поднимай рейтинг и открывай возможность ставить больше!'),
    ('bet_boost_lock_msg', 'ru', '🔓 Повышение ставок станет доступным, когда ты наберешь больше рейтинговых баллов.
💪 Просто играй активнее, зарабатывай баллы – и сможешь ставить больше, увеличивая возможный выигрыш!'),
    ('bet_boost_unlock_msg', 'ru', '🎉 <b>Привет!</b> Ты открыл доступ к повышению ставок.
Используй свои рейтинговые баллы, увеличивай ставку и повышай свои шансы на большой выигрыш 🎯'),
 ('btn_mult_x2_locked', 'ru', '🔒 X2'),
 ('btn_betboost_2_locked', 'ru', '🔒+2'),
 ('btn_betboost_5_locked', 'ru', '🔒+5'),
 ('btn_betboost_10_locked', 'ru', '🔒+10'),
 ('btn_betboost_15_locked', 'ru', '🔒+15'),
 ('btn_betboost_20_locked', 'ru', '🔒+20'),
 ('boost_limit', 'ru', 'Недостаточно баллов для повышения ставки');

INSERT INTO localizations (key, language, value) VALUES
    ('bet_boost_prompt_msg', 'uk', '🎰 Твоя ставка у цьому раунді: <b>0</b> рейтингових балів

Якщо ставка зіграє — отримаєш <b>1</b> рейтинговий бал 💎

📈 Можеш підвищити ставку, використавши зароблені рейтингові бали.

Як це працює:
• 🔴 / ⚫️ — <b>х2</b> від поставлених балів
• 🟢 Zero — <b>х35</b>
• ❌ Промах — <b>втрачаєш всю ставку</b>

💬 <b>Скільки балів хочеш поставити цього раунду?</b>'),
    ('bet_adjust_prompt_msg', 'uk', '🎰 Твоя поточна ставка: <b>%d</b> балів

Якщо вгадаєш — множиш свою ставку:
• 🔴 / ⚫️ — <b>х2</b>,
• 🟢 Zero — <b>х35</b>,
• ❌ Промах — <b>0 балів</b>

🔧 Хочеш підкоригувати ставку?
Обери, як саме:
• X2 — подвоїти ставку
• +5 / +10 / +15 / +20 — додати бали
• ✍️ Вказати к-сть балів — встановити вручну'),
    ('btn_betboost_1', 'uk', '+1'),
    ('btn_betboost_2', 'uk', '+2'),
    ('btn_betboost_5', 'uk', '+5'),
    ('btn_betboost_10', 'uk', '+10'),
    ('btn_betboost_15', 'uk', '+15'),
    ('btn_betboost_20', 'uk', '+20'),
    ('btn_betboost_manual', 'uk', '✍️ Вказати к-сть балів'),
    ('btn_betboost_skip', 'uk', '🚫 Не підвищувати'),
    ('ow_rating_balance_msg', 'uk', 'Недостатньо рейтингових балів для цієї ставки 😶
Грай у раундах, заробляй бали та повертайся за підвищеною ставкою 💪🔥'),
    ('enter_bet_amount', 'uk', 'Введи розмір ставки цифрою. 
    ❗️Ставка не може бути більшою за кількість твоїх зароблених балів.
⚠️ Мінімум — 1 бал'),
    ('enter_bet_mult', 'uk', 'Вкажи к-сть балів цифрою.
❗️Ставка не може бути більшою за кількість твоїх зароблених балів.
⚠️ Мінімум — 1 бал'),
    ('btn_mult_x2', 'uk', 'X2'),
    ('btn_reset_bet', 'uk', '❌ Скинути ставку до 0'),
    ('btn_keep_bet', 'uk', '🚫 Не змінювати'),
    ('input_invalid_msg', 'uk', '🚫Схоже ти ввів некоректні дані. 
👉 Спробуй ще раз'),
    ('input_invalid2_msg', 'uk', '🚫Схоже ти знову ввів некоректні дані. 
👇 Натисни кнопку нижче, щоб почати новий раунд'),
    ('input_invalid_zero_msg', 'uk', '⚠️ Невірне значення. Мінімум — 1 бал
👉 Спробуй ще раз'),
    ('btn_confirm_bet', 'uk', '✅ Підтвердити ставку'),
    ('nextbid15_boost', 'uk', '🎰 Раунд #%s
🔒 Хеш:%s
⏳ Зроби свій вибір, до визначення ставки залишилось <b>%d</b> секунд.
Розмір ставки в балах: <b>%d</b>'),
    ('nomorebids_boost', 'uk', 'Твоя ставка %d рейтингових балів прийнята ✅Ставок більше немає!'),
    ('btn_bet_boost_info', 'uk', '📝 Як працює система підвищення ставок'),
    ('bet_boost_info_msg', 'uk', '📌 На старті кожен гравець має 0 рейтингових балів.
📌 За виграші у раундах ти отримуєш рейтингові бали.
📌 Коли накопичиш достатньо — зможеш змінювати свою ставку:
• або додавати фіксовану кількість балів (+5, +10, +15, +20);
• або множити поточну ставку (×2);
• або вказати точну кількість балів вручну.

🎯 Чим вища ставка — тим більший потенційний виграш.
⚠️ Але пам’ятай: програш забирає всі поставлені бали.

💪 Грай активніше, піднімай рейтинг і відкривай можливість ставити більше!'),
    ('bet_boost_lock_msg', 'uk', '🔓 Підвищення ставок стане доступним, коли ти набереш більше рейтингових балів.
💪 Просто грай активніше, заробляй бали — і зможеш ставити більше, збільшуючи можливий виграш!'),
    ('bet_boost_unlock_msg', 'uk', '🎉 <b>Вітаю!</b> Ти відкрив доступ до підвищення ставок.
Використовуй свої рейтингові бали, збільшуй ставку й підвищуй свої шанси на великий виграш 🎯'),
 ('btn_mult_x2_locked', 'uk', '🔒 X2'),
 ('btn_betboost_2_locked', 'uk', '🔒+2'),
 ('btn_betboost_5_locked', 'uk', '🔒+5'),
 ('btn_betboost_10_locked', 'uk', '🔒+10'),
 ('btn_betboost_15_locked', 'uk', '🔒+15'),
 ('btn_betboost_20_locked', 'uk', '🔒+20'),
 ('boost_limit', 'uk', 'Недостатньо балів для підвищення ставки');

INSERT INTO localizations (key, language, value) VALUES
    ('bet_boost_prompt_msg', 'en', '🎰 Your bet in this round: <b>0</b> rating points

If the bet is successful, you will receive <b>1</b> rating point 💎

📈 You can increase your bet using the earned rating points.

How it works:
• 🔴 / ⚫️ — <b>x2</b> of the points bet
• 🟢 Zero — <b>x35</b>
• ❌ Miss — <b>lose your entire bet</b>

💬 <b>How many points do you want to bet this round?</b>'),
    ('bet_adjust_prompt_msg', 'en', '🎰 Your current bet: <b>%d</b> points

If you guess — multiply your bet:
• 🔴 / ⚫️ — <b>x2</b>,
• 🟢 Zero — <b>x35</b>,
• ❌ Miss — <b>0 points</b>

🔧 Do you want to adjust your bet?
Choose how:
• X2 — double the bet
• +5 / +10 / +15 / +20 — add points
• ✍️ Specify the number of points — set manually'),
    ('btn_betboost_1', 'en', '+1'),
    ('btn_betboost_2', 'en', '+2'),
    ('btn_betboost_5', 'en', '+5'),
    ('btn_betboost_10', 'en', '+10'),
    ('btn_betboost_15', 'en', '+15'),
    ('btn_betboost_20', 'en', '+20'),
    ('btn_betboost_manual', 'en', '✍️ Specify the number of points'),
    ('btn_betboost_skip', 'en', '🚫 Do not raise'),
    ('ow_rating_balance_msg', 'en', 'Not enough rating points for this bet 😶
Play rounds, earn points and come back for a higher bet 💪🔥'),
    ('enter_bet_amount', 'en', 'Enter the bet amount in numbers.
❗️The bet cannot be greater than the number of points you have earned.
⚠️ Minimum — 1 point'),
    ('enter_bet_mult', 'en', 'Enter the number of points in numbers.
❗️The bet cannot be greater than the number of points you have earned.
⚠️ Minimum — 1 point'),
    ('btn_mult_x2', 'en', 'X2'),
    ('btn_reset_bet', 'en', '❌ Reset bid to 0'),
    ('btn_keep_bet', 'en', '🚫 Do not change'),
    ('input_invalid_msg', 'en', '🚫It seems you entered incorrect data.
👉 Try again'),
    ('input_invalid2_msg', 'en', '🚫 It looks like you entered incorrect information again.
👇 Click the button below to start a new round'),
    ('input_invalid_zero_msg', 'en', '⚠️ Invalid value. Minimum — 1 point
👉 Try again'),
    ('btn_confirm_bet', 'en', '✅ Confirm bid'),
    ('nextbid15_boost', 'en', '🎰 Round #%s
🔒 Hash:%s
⏳ Make your choice, <b>%d</b> seconds left to place your bet.
Bet amount in points: <b>%d</b>'),
    ('nomorebids_boost', 'en', 'Your bid of %d rating points has been accepted ✅No more bids!'),
    ('btn_bet_boost_info', 'en', '📝 How the rate increase system works'),
    ('bet_boost_info_msg', 'en', '📌 At the start, each player has 0 rating points.
📌 For winning rounds, you get rating points.
📌 When you accumulate enough, you can change your bet:
• or add a fixed number of points (+5, +10, +15, +20);
• or multiply the current bet (×2);
• or specify the exact number of points manually.

🎯 The higher the bet, the greater the potential win.
⚠️ But remember: losing takes away all the points you bet.

💪 Play more actively, raise your rating and open the opportunity to bet more!'),
    ('bet_boost_lock_msg', 'en', '🔓 Increasing your stakes will become available as you gain more rating points.
💪 Just play more actively, earn points — and you will be able to bet more, increasing your possible winnings!'),
    ('bet_boost_unlock_msg', 'en', '🎉 <b>Congratulations!</b> You have unlocked access to the bet increase.
Use your rating points, increase your bet and increase your chances of winning big 🎯'),
 ('btn_mult_x2_locked', 'en', '🔒 X2'),
 ('btn_betboost_2_locked', 'en', '🔒+2'),
 ('btn_betboost_5_locked', 'en', '🔒+5'),
 ('btn_betboost_10_locked', 'en', '🔒+10'),
 ('btn_betboost_15_locked', 'en', '🔒+15'),
 ('btn_betboost_20_locked', 'en', '🔒+20'),
 ('boost_limit', 'en', 'Not enough points to raise your bid');
