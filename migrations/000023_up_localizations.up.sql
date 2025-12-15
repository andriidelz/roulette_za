ALTER TABLE weekly_ratings
ADD COLUMN IF NOT EXISTS won_bets INT DEFAULT 0;

INSERT INTO localizations (key, language, value) VALUES
  -- ru
  ('bet_adjust_no_points_msg', 'ru', '🚫 <b>На твоем балансе недостаточно баллов, чтобы повторить предыдущую ставку</b>

🎰 Предыдущая ставка: <b>%d</b> баллов
💰 Доступно на балансе: <b>%d</b> баллов

Вы можете выбрать новый размер ставки, используя кнопки ниже:'),
  ('bet_boost_motivation_msg', 'ru', '💡 Помни, что ты можешь повышать ставку за счет рейтинговых баллов – так проще вырваться вперед в топе.'),

  -- uk
  ('bet_adjust_no_points_msg', 'uk', '🚫 <b>На твоєму балансі недостатньо балів, щоб повторити попередню ставку</b>

🎰 Попередня ставка: <b>%d</b> балів
💰 Доступно на балансі: <b>%d</b> балів

Ти можеш обрати новий розмір ставки використовуючи кнопки нижче:'),
  ('bet_boost_motivation_msg', 'uk', '💡 Памʼятай, що ти можеш підвищувати ставку за рахунок рейтингових балів — так простіше вирватися вперед у топі.'),

  -- en
  ('bet_adjust_no_points_msg', 'en', '🚫 <b>You do not have enough points on your balance to repeat your previous bet</b>

🎰 Previous bet: <b>%d</b> points
💰 Available on your balance: <b>%d</b> points

You can choose a new bet size using the buttons below:'),
  ('bet_boost_motivation_msg', 'en', '💡 Remember that you can increase your bet using rating points - this way it is easier to break ahead in the top.');
