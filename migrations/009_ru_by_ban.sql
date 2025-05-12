-- Добавление локализаций для сообщения о блокировке стран RU и BY

-- Сначала удаляем существующие локализации, если они есть
DELETE FROM localizations WHERE key = 'stopcountry';

-- Добавляем локализации для разных языков
INSERT INTO localizations (key, language, value) VALUES
('stopcountry', 'en', 'Service is not available for residents of Russia or Belarus.'),
('stopcountry', 'ru', 'Сервис не доступен для жителей россии или белларуси.'),
('stopcountry', 'uk', 'Сервіс недоступний для мешканців росії або білорусі.');
