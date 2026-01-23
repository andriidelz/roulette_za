DELETE FROM settings WHERE key ='captcha_ttl';
DELETE FROM settings WHERE key ='captcha_refresh_count';
DELETE FROM settings WHERE key ='captcha_need_count';
DELETE FROM settings WHERE key ='captcha_wrong_count';
DELETE FROM settings WHERE key ='captcha_ban_count';
DELETE FROM settings WHERE key ='captcha_ban_short_ttl';
DELETE FROM settings WHERE key ='captcha_ban_long_ttl';
DELETE FROM settings WHERE key ='captcha_user_activity_ttl';
DELETE FROM settings WHERE key ='captcha_bet_activity_ttl';
DELETE FROM settings WHERE key ='captcha_bet_duplicate_ttl';

DROP INDEX IF EXISTS idx_user_ban_logs_user_id;
-- Drop table
DROP TABLE IF EXISTS user_ban_logs;

DELETE FROM localizations
WHERE key IN ('captcha_next', 'captcha_stage_title', 'captcha_refresh', 'captcha_blocked_action')
    AND language IN ('ru', 'uk', 'en');