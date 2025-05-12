-- Создаем файл миграции 008_nickname_localizations.sql

-- Удаление существующих локализаций при их наличии (чтобы избежать дублирования)
DELETE FROM localizations WHERE key IN (
    'name_mes',
    'name_changeyes',
    'name_changeno',
    'name_changeok',
    'name_changesave',
    'invalid_nickname'
);

-- English
INSERT INTO localizations (key, language, value) VALUES
('name_mes', 'en', 'Your name that will be displayed in the general rating is {profile_name} and will be publicly visible to all players. Do you want to change it?'),
('name_changeyes', 'en', 'Yes, I want to change'),
('name_changeno', 'en', 'No, I don''t want to change'),
('name_changeno_msg', 'en', 'Your game name has been fixed'),
('name_changeok', 'en', 'Enter the desired name for public display in the rating. Only Latin alphabet, numbers and underscore are allowed. The name must be 3-20 characters long.'),
('name_changesave', 'en', 'New game name for rating has been saved.'),
('invalid_nickname', 'en', 'Invalid nickname. The nickname should contain only Latin letters, numbers, and underscores, and be 3-20 characters long. Please try again.');

-- Russian
INSERT INTO localizations (key, language, value) VALUES
('name_mes', 'ru', 'Ваше имя, которое будет отображаться в общем рейтинге {profile_name} и будет публично видно всем игрокам. Хотите его изменить?'),
('name_changeyes', 'ru', 'Да, хочу изменить'),
('name_changeno', 'ru', 'Нет, не хочу изменять'),
('name_changeno_msg', 'ru', 'Ваше игровое имя зафиксировано'),
('name_changeok', 'ru', 'Отправьте желаемое имя для публичного отображения в рейтинге. Допускается использование только латинского алфавита, цифр и знака подчеркивания. Имя должно быть длиной 3-20 символов.'),
('name_changesave', 'ru', 'Новое игровое имя для рейтинга сохранено.'),
('invalid_nickname', 'ru', 'Некорректный никнейм. Никнейм должен содержать только латинские буквы, цифры и знак подчеркивания, а также быть длиной от 3 до 20 символов. Пожалуйста, попробуйте еще раз.');

-- Ukrainian
INSERT INTO localizations (key, language, value) VALUES
('name_mes', 'uk', 'Ваше ім''я, яке буде відображатися в загальному рейтингу {profile_name} і буде публічно видно всім гравцям. Бажаєте його змінити?'),
('name_changeyes', 'uk', 'Так, хочу змінити'),
('name_changeno', 'uk', 'Ні, не хочу змінювати'),
('name_changeno_msg', 'uk', 'Ваше ігрове ім''я зафіксовано'),
('name_changeok', 'uk', 'Надішліть бажане ім''я для публічного відображення в рейтингу. Допускається використання лише латинського алфавіту, цифр та знаку підкреслення. Ім''я повинно бути довжиною 3-20 символів.'),
('name_changesave', 'uk', 'Нове ігрове ім''я для рейтингу збережено.'),
('invalid_nickname', 'uk', 'Некоректний нікнейм. Нікнейм повинен містити лише латинські літери, цифри та знак підкреслення, а також бути довжиною від 3 до 20 символів. Будь ласка, спробуйте ще раз.');
