-- Добавляем столбец image в таблицу localizations
ALTER TABLE localizations ADD COLUMN IF NOT EXISTS image TEXT;
ALTER TABLE localizations ADD COLUMN IF NOT EXISTS video TEXT;

INSERT INTO localizations (key, language, value) VALUES
    ('invalid_name', 'ru', 'Имя должно содержать от 1 до 100 символов'),
    ('update_error', 'ru', 'Ошибка при обновлении данных. Попробуйте еще раз.');

INSERT INTO localizations (key, language, value) VALUES
    ('invalid_name', 'uk', 'Ім`я має містити від 1 до 100 символів'),
    ('update_error', 'uk', 'Ошибка при оновленні даних. Спробуйте ще раз.');

INSERT INTO localizations (key, language, value) VALUES
    ('invalid_name', 'en', 'The name must be between 1 and 100 characters long.'),
    ('update_error', 'en', 'Error updating data. Try again.');
