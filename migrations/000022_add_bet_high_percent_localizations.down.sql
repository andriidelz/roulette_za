-- Roll back: remove new keys and restore ow_rating_balance_msg
DELETE FROM localizations
WHERE key IN ('bet_high_percent', 'low_rating_balance_msg')
    AND language IN ('ru', 'uk', 'en');
INSERT INTO localizations (key, language, value)
VALUES (
        'ow_rating_balance_msg',
        'ru',
        'Недостаточно рейтинговых баллов для этой ставки 😶'
    ),
    (
        'ow_rating_balance_msg',
        'uk',
        'Недостатньо рейтингових балів для цієї ставки 😶'
    ),
    (
        'ow_rating_balance_msg',
        'en',
        'Not enough rating points for this bet 😶'
    );
