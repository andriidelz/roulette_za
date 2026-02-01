

DELETE FROM localizations
WHERE key IN ('captcha_status_title', 'captcha_status_active', 'captcha_status_banned')
    AND language IN ('ru', 'uk', 'en');