DELETE FROM localizations WHERE key ='low_rating_balance_msg';
DELETE FROM localizations WHERE key ='bet_adjust_no_points_msg';
DELETE FROM localizations WHERE key ='losemessage_boost';

INSERT INTO localizations (key, language, value) VALUES
  -- ru
  ('losemessage_boost', 'ru', 'Твоя ставка не сыграла! 
<b>%d</b> рейтинговые баллы списаны с твоего баланса.'),
  ('bet_adjust_no_points_msg', 'ru', '🚫 <b>На твоем балансе недостаточно баллов, чтобы повторить предыдущую ставку</b>

🎰 Предыдущая ставка: <b>%d</b> %s
💰 Доступно на балансе: <b>%d</b> %s

Вы можете выбрать новый размер ставки, используя кнопки ниже:'),
  ('low_rating_balance_msg', 'ru', 'Недостаточно рейтинговых баллов для этой ставки 😶
Играй в раундах, зарабатывай баллы и возвращайся по повышенной ставке 💪🔥

Достаточно баллов: <b>%d</b>'),

  -- uk
  ('losemessage_boost', 'uk', 'Твоя ставка не зіграла! 
 <b>%d</b> рейтингових балів списано з твого балансу.'),
   ('bet_adjust_no_points_msg', 'uk', '🚫 <b>На твоєму балансі недостатньо балів, щоб повторити попередню ставку</b>

🎰 Попередня ставка: <b>%d</b> %s
💰 Доступно на балансі: <b>%d</b> %s

Ти можеш обрати новий розмір ставки використовуючи кнопки нижче:'),
  ('low_rating_balance_msg', 'uk', 'Недостатньо рейтингових балів для цієї ставки 😶
Грай у раундах, заробляй бали та повертайся за підвищеною ставкою 💪🔥

Достуно балів: <b>%d</b>'),

  -- en
   ('losemessage_boost', 'en', 'Your bet did not win!
<b>%d</b> rating points have been deducted from your balance.'),
  ('bet_adjust_no_points_msg', 'en', '🚫 <b>You do not have enough points on your balance to repeat your previous bet</b>

🎰 Previous bet: <b>%d</b> %s
💰 Available on your balance: <b>%d</b> %s

You can choose a new bet size using the buttons below:'),
  ('low_rating_balance_msg', 'en', 'Not enough rating points for this bet 😶
Play rounds, earn points and come back for a higher bet 💪🔥

Points available: <b>%d</b>');
