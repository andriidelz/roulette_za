DELETE FROM localizations WHERE key ='captcha_blocked_action';
DELETE FROM localizations WHERE key ='captcha_nex';

INSERT INTO localizations (key, language, value) VALUES
  -- ru
  ('captcha_status_title', 'ru', 'Изменение статуса'), 
  ('captcha_status_active', 'ru', 'Вы были разблокированы'), 
  ('captcha_status_banned', 'ru', 'Вы были заблокированы'), 
  ('captcha_nex', 'ru', 'Продолжаем проверку – еще немного и ты в игре 🎮'), 
  ('captcha_blocked_action', 'ru', 'Проверка не пройдена. Ты несколько раз подряд неправильно выполнил капчу, поэтому мы временно ограничили доступ к игре.'),

  -- uk
  ('captcha_status_title', 'uk', 'Зміна статусу'),
  ('captcha_status_active', 'uk', 'Ви були розблоковані'),
  ('captcha_status_banned', 'uk', 'Ви були заблоковані'),
  ('captcha_nex', 'uk', 'Продовжуємо перевірку — ще трохи і ти в грі 🎮'),
  ('captcha_blocked_action', 'uk', 'Перевірку не пройдено. Ти кілька разів поспіль неправильно виконав капчу, тому ми тимчасово обмежили доступ до гри.'),

  -- en
  ('captcha_status_title', 'en', 'Status change'),
  ('captcha_status_active', 'en', 'You have been unblocked'),
  ('captcha_status_banned', 'en', 'You have been blocked'),
  ('captcha_nex', 'en', 'We continue the verification - just a little more and you will be in the game 🎮'),
  ('captcha_blocked_action', 'en', 'The verification failed. You have incorrectly completed the captcha several times in a row, so we have temporarily restricted access to the game.');