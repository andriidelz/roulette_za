-- Добавляем столбец registered в таблицу users
ALTER TABLE users ADD COLUMN IF NOT EXISTS bet BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bet_boost BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bet_at TIMESTAMP;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bet_boost_at TIMESTAMP;

UPDATE public.users AS u
SET
    bet = TRUE,
    bet_at = sq.created_at
FROM
    (
        SELECT DISTINCT ON (user_id) 
            user_id, 
            created_at
        FROM 
            public.bets
        ORDER BY 
            user_id, 
            created_at ASC
    ) AS sq 
WHERE u.id = sq.user_id;

UPDATE public.users AS u
SET
    bet_boost = TRUE,
    bet_boost_at = sq.created_at
FROM
    (
        SELECT DISTINCT ON (user_id) 
            user_id, 
            created_at
        FROM 
            public.bets
        WHERE bet_point > 0
        ORDER BY 
            user_id, 
            created_at ASC
    ) AS sq 
WHERE u.id = sq.user_id;