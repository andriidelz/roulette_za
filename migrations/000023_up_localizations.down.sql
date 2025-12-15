ALTER TABLE weekly_ratings DROP COLUMN won_bets;

DELETE FROM localizations
WHERE key IN ('bet_adjust_no_points_msg', 'bet_boost_motivation_msg')
    AND language IN ('ru', 'uk', 'en');
