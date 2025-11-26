-- Add bet_high_percent + low_rating_balance_msg, remove old ow_rating_balance_msg

DELETE FROM localizations
WHERE key = 'ow_rating_balance_msg'
  AND language IN ('ru', 'uk', 'en');

INSERT INTO localizations (key, language, value) VALUES
  -- ru
  ('bet_high_percent', 'ru', 'Размер твоей ставки составляет: %d
Твой рейтинговый баланс: %d'),
  ('low_rating_balance_msg', 'ru', 'Недостаточно рейтинговых баллов для этой ставки 😶'),

  -- uk
  ('bet_high_percent', 'uk', 'Розмір твоєї ставки складає: %d
Твій рейтинговий баланс: %d'),
  ('low_rating_balance_msg', 'uk', 'Недостатньо рейтингових балів для цієї ставки 😶'),

  -- en
  ('bet_high_percent', 'en', 'Your bet size is: %d
Your rating balance: %d'),
  ('low_rating_balance_msg', 'en', 'Not enough rating points for this bet 😶');
