-- Добавляем поле age_verified в таблицу пользователей
ALTER TABLE users
ADD COLUMN IF NOT EXISTS age_verified BOOLEAN DEFAULT NULL;

-- Добавление новых локализаций
-- Английский
INSERT INTO localizations (key, language, value) VALUES
('agemes', 'en', 'Please confirm that you are over 18 years old'),
('yes18', 'en', 'Yes, I am over 18 years old'),
('no18', 'en', 'No, I am under 18 years old'),
('stopage', 'en', 'Sorry, this service is only available to users over 18 years old.');

-- Русский
INSERT INTO localizations (key, language, value) VALUES
('agemes', 'ru', 'Подтвердите, что ваш возраст старше 18 лет'),
('yes18', 'ru', 'Да, я старше 18 лет'),
('no18', 'ru', 'Нет, мне нет 18 лет'),
('stopage', 'ru', 'К сожалению, сервис доступный только для пользователей старше 18 лет.');

-- Украинский
INSERT INTO localizations (key, language, value) VALUES
('agemes', 'uk', 'Підтвердіть, що ваш вік старше 18 років'),
('yes18', 'uk', 'Так, я старше 18 років'),
('no18', 'uk', 'Ні, мені немає 18 років'),
('stopage', 'uk', 'На жаль, сервіс доступний тільки для користувачів старше 18 років.');
