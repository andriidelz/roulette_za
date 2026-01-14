package bot

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"roulette/internal/logger"
	"roulette/internal/models"

	"github.com/redis/go-redis/v9"
)

// StartUpdateCaptcha запускает планировщик заданий для обновления капчи
func (b *Bot) StartUpdateCaptcha() {
	// Запускаем планировщик для обновления капч

	cont, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Список всіх користувачів що можуть отримати оновлення капчі через бездіяльність
	allUsers, err := b.redisDB.LRange(cont, userCaptchaUpdateKey, 0, -1).Result()
	if err != nil {
		logger.Error.Printf("Failed to LRANGE all elements: %v", err)
	}

	if len(allUsers) == 0 {
		return
	}

	b.settingsMutex.Lock()
	countRefreshCaptcha, _ := b.settings["captcha_refresh_count"]
	b.settingsMutex.Unlock()

	for i := range allUsers {
		userID, err := strconv.ParseInt(allUsers[i], 10, 64)
		if err != nil {
			logger.Error.Printf("Could not parse '%s': %v\n", allUsers[i], err)
			continue
		}

		// Якщо користувач в списку на приймання капчі і немає відповіді протягом 30 сек оновлюєм капчу (3 рази)
		userUpdateKey := fmt.Sprintf(userCaptchaUpdateCountPrefix, userID)
		val, err := b.redisDB.Get(cont, userUpdateKey).Int64()
		if err == redis.Nil {
			// Ще не було відправки, створюємо запис
			err = b.redisDB.Set(cont, userUpdateKey, 1, 0).Err()
			if err != nil {
				logger.Error.Printf("Error Set %d: %v", userID, err)
			}
			continue
		} else if err != nil {
			logger.Error.Printf("Error Get %d: %v", userID, err)
			continue
		}
		val++

		if val > countRefreshCaptcha {

			// Было отправлено 3 одновления капчи. Прекращаем отправку.
			// Удаляем из список пользователей которые ожидают капчу
			_, err := b.redisDB.LRem(cont, userCaptchaUpdateKey, 0, userID).Result()
			if err != nil {
				logger.Error.Printf("Failed to LREM: %d: %v", userID, err)
			}
			// Удаляем ключ
			_, err = b.redisDB.Del(cont, userUpdateKey).Result()
			if err != nil {
				logger.Error.Printf("Error Del %d: %v", userID, err)
			}

		} else {

			// Пользователь не превышает активность
			// Обновляем кол-во
			err = b.redisDB.Set(cont, userUpdateKey, val, redis.KeepTTL).Err()
			if err != nil {
				logger.Error.Printf("Error Set %d: %v", userID, err)
			}

			language, err := b.getUserLang(userID, "")
			if err != nil {
				logger.Error.Printf("Error getting user %d: %v", userID, err)
				continue
			}

			// Присилаємо нову капчу
			b.SendMessage(userID, b.captchaMessage(userID, language, "refresh"))
		}
	}
}

// StartUpdateCache запускает планировщик кеша
func (b *Bot) StartUpdateCache() {
	go func() {
		fiveMinuteTicker := time.NewTicker(5 * time.Minute)
		minuteTicker := time.NewTicker(1 * time.Minute)
		thirtySecTicker := time.NewTicker(30 * time.Second)
		defer thirtySecTicker.Stop()
		defer fiveMinuteTicker.Stop()
		defer minuteTicker.Stop()

		b.refreshMinuteCache()
		b.refreshUserCache()
		b.refreshLocalizationCache()

		for {
			select {
			case <-thirtySecTicker.C:
				b.StartUpdateCaptcha()
			case <-minuteTicker.C:
				b.refreshMinuteCache()
				b.refreshUserCache()
				b.refreshActiveUsers()
			case <-fiveMinuteTicker.C:
				b.refreshLocalizationCache()
			}
		}
	}()
}

// Зберігаємо налаштування призового фонду
// Зберігаємо налаштування капчі
func (b *Bot) refreshMinuteCache() {
	year, week := time.Now().ISOWeek()

	// По умолчанию
	var prizeFundAmount float64 = 1000.0
	var topCount int = 100
	var totalPoints int = 0

	// Получаем призовой фонд через репозиторий
	prizeFund, err := b.service.GetPrizeFund(year, week)
	if err != nil {
		logger.Error.Printf("Error getting prize fund: %v", err)
	} else {
		// Устанавливаем данные о призовом фонде из БД
		prizeFundAmount = prizeFund.Amount
		topCount = prizeFund.TopCount
	}

	// Отримуємо рейтинг
	ratings, err := b.service.GetWeeklyRating(topCount)
	if err != nil {
		logger.Error.Printf("Error getting GetWeeklyRating: %v", err)
	}

	// Рахуємо загальну кількість балів у топі
	for _, rating := range ratings {
		totalPoints += rating.Points
	}

	b.gameHandler.prizeFundMutex.Lock()
	b.gameHandler.prizeFundAmount = prizeFundAmount
	// b.gameHandler.topCount = topCount
	b.gameHandler.totalPoints = totalPoints
	b.gameHandler.prizeFundMutex.Unlock()

	// Налаштування капчі
	settings, err := b.service.GetSettings()
	if err != nil {
		logger.Error.Printf("Error GetSettings: %v", err)
	}

	settingsMap := map[string]int64{}
	params := []string{
		"captcha_bet_activity",      // Лимит ставок за период betActivityExpiration
		"captcha_bet_activity_ttl",  // Период ставок для лимита (сек)
		"captcha_user_activity",     // Лимит действий за период userActivityExpiration
		"captcha_user_activity_ttl", // Период действий для лимита (сек)
		"captcha_bet_points",        // Лимит баллов для запуска капчи
		"captcha_bet_duplicate_ttl", // Период дубликатов ставок (сек)

		"captcha_ttl",           // Время ожидания капчи (мин)
		"captcha_refresh_count", // Кол-во обновлений
		"captcha_need_count",    // Кол-во этапов
		"captcha_wrong_count",   // Кол-во неправильнх ответов
		"captcha_ban_count",     // Кол-во банов
		"captcha_ban_short_ttl", // Время бана short (мин)
		"captcha_ban_long_ttl",  // Время бана long (мин)
	}

	for i := range params {
		key := params[i]
		if settKey, ok := settings[key]; ok {
			if limit, err := strconv.Atoi(settKey); err == nil {
				settingsMap[key] = int64(limit)
			}
		}
	}

	b.settingsMutex.Lock()
	b.settings = settingsMap
	b.settingsMutex.Unlock()
}

// Зберігаємо локалізацію мов в кеш
func (b *Bot) refreshLocalizationCache() {
	languages := []string{"en", "ru", "uk"}

	for _, lang := range languages {
		localizations, err := b.service.GetRepo().GetAllLocalizationsForLanguage(lang)
		if err != nil {
			logger.Error.Printf("Error GetAllLocalizationsForLanguage: %v", err)
		}

		localizationMap := make(map[string]models.Localization)
		for _, loc := range localizations {
			localizationMap[loc.Key] = loc
		}

		b.localMutex.Lock()
		b.localizations[lang] = localizationMap
		b.localMutex.Unlock()
	}
}

// TODO: при кількості користувачів більше 1000000 переписати на отримання порціями
func (b *Bot) refreshUserCache() {
	// Зберігаємо локалізацію користувачів в кеш
	usersInfo := map[int64]models.User{}
	var count int64
	users, totalUsers, err := b.service.GetRepo().GetUsers(1, 1000000)
	if err != nil {
		logger.Error.Printf("Error GetUsers: %v", err)
		return
	}
	count = totalUsers

	logger.Error.Println(count)

	for i := range users {
		usersInfo[users[i].TelegramID] = users[i]
	}

	b.usersInfoMutex.Lock()
	b.usersInfo = usersInfo
	b.usersInfoMutex.Unlock()
}

// getUser - отримання користувача з кешу
func (b *Bot) getUser(telegramID int64) (*models.User, error) {
	b.usersInfoMutex.Lock()
	v, ok := b.usersInfo[telegramID]
	b.usersInfoMutex.Unlock()

	if ok {
		return &v, nil
	}

	logger.Error.Println("Cannot find: ", telegramID)

	dbUser, err := b.service.GetUser(telegramID)
	if err == nil {
		b.usersInfoMutex.Lock()
		b.usersInfo[telegramID] = *dbUser
		b.usersInfoMutex.Unlock()
	}
	return dbUser, err
}

// getUser - оновлення користувача в кеші
func (b *Bot) updateUserCache(telegramID int64) error {
	dbUser, err := b.service.GetUser(telegramID)
	if err == nil {
		b.usersInfoMutex.Lock()
		b.usersInfo[telegramID] = *dbUser
		b.usersInfoMutex.Unlock()
	}
	return err
}

// getUserLang - обгортка getUser, отримання лише мови користувача
func (b *Bot) getUserLang(telegramID int64, appLanguage string) (string, error) {
	// Отримуємо користувача з кешу або бази
	dbUser, err := b.getUser(telegramID)

	// Визначаємо мову
	return getLanguage(dbUser.LanguageCode, appLanguage), err
}

// updateUserActivity - перевірка і за відсутності додавання користувача в список активних протягом останньої 1 хв
func (b *Bot) updateUserActivity(userID int64) {
	b.activeUsersMutex.Lock()
	if _, ok := b.activeUsers[userID]; !ok {
		b.activeUsers[userID] = true
	}
	b.activeUsersMutex.Unlock()
}

func (b *Bot) refreshActiveUsers() {
	b.activeUsersMutex.Lock()
	data := b.activeUsers
	b.activeUsersMutex.Unlock()

	if metrics := b.getMetrics(); metrics != nil && metrics.Bot != nil {
		metrics.Bot.SetActiveUsers(float64(len(data)))
	}

	for userID := range data {

		// logger.Error.Println("UpdateUserActivity", userID)
		if err := b.service.UpdateUserActivity(userID); err != nil {
			logger.Error.Printf("Error UpdateUserActivity: %d, %v", userID, err)
		}
	}

	// очищуєм map
	b.activeUsersMutex.Lock()
	b.activeUsers = map[int64]bool{}
	b.activeUsersMutex.Unlock()
}
