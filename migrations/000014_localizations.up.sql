ALTER TABLE bets
ADD COLUMN IF NOT EXISTS get_result BOOLEAN DEFAULT TRUE;

INSERT INTO localizations (key, language, value) VALUES
    ('next_round', 'ru', 'Начать следующий раунд'),
    ('repeatresultalert', 'ru', 'Вы уже получили результат этого раунда раньше'),
    ('wait_result', 'ru', 'Результаты еще не рассчитываются. Осталось пару секунд - дождись сообщения системы.');

INSERT INTO localizations (key, language, value) VALUES
    ('next_round', 'uk', 'Почати наступний раунд'),
    ('repeatresultalert', 'uk', 'Ви вже отримали результат цього раунду раніше'),
    ('wait_result', 'uk', 'Результати ще розраховуються. Залишилось пару секунд - дочекайся повідомлення системи.');

INSERT INTO localizations (key, language, value) VALUES
    ('next_round', 'en', 'Start next round'),
    ('repeatresultalert', 'en', 'You have already received the result of this round before.'),
    ('wait_result', 'en', 'The results are still being calculated. There are a few seconds left - wait for the system message.');
