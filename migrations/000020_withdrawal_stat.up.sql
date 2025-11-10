
-- Статистика балансів та виводів
CREATE TABLE IF NOT EXISTS withdrawal_stats (
    id SERIAL PRIMARY KEY,
    day VARCHAR(12) UNIQUE NOT NULL,
    earn FLOAT DEFAULT 0,
    withdrawal FLOAT DEFAULT 0,
    payout FLOAT DEFAULT 0,
    balance FLOAT DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_withdrawal_stats_day ON withdrawal_stats (day);