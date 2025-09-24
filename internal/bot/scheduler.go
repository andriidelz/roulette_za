package bot

import (
	"context"
	"fmt"
	"roulette/internal/logger"
	"roulette/internal/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// StartRatingScheduler запускает планировщик заданий для обновления рейтингов
func (b *Bot) StartRatingScheduler() {
	go func() {
		// Периодическое обновление рейтингов (каждые 15 минут)
		ratingTicker := time.NewTicker(15 * time.Minute)
		defer ratingTicker.Stop()

		// TODO перенести определение недели внутрь цикла и оттестить

		// // Проверка на новую неделю (каждый час)
		// weeklyResetTicker := time.NewTicker(1 * time.Hour)
		// defer weeklyResetTicker.Stop()

		// // Хранение информации о последней обработанной неделе
		// var lastProcessedYear, lastProcessedWeek int

		// // Сразу проверяем необходимость создания рейтинга для текущей недели
		// currentYear, currentWeek := time.Now().ISOWeek()
		// // При запуске проверим, существует ли рейтинг для текущей недели
		// _, err := b.service.GetPrizeFund(currentYear, currentWeek)
		// if err != nil {
		// 	logger.Error.Printf("Error checking current week's prize fund: %v", err)
		// }

		// // Устанавливаем последнюю обработанную неделю
		// lastProcessedYear, lastProcessedWeek = currentYear, currentWeek

		for {
			select {

			case <-ratingTicker.C:
				// Обновляем позиции в рейтинге
				if err := b.service.RefreshAllRatings(); err != nil {
					logger.Error.Printf("Error refreshing ratings: %v", err)
				} else {
					logger.Info.Println("Successfully refreshed ratings")
				}

				// case <-weeklyResetTicker.C:
				// 	// Проверяем, наступила ли новая неделя
				// 	now := time.Now()
				// 	year, week := now.ISOWeek()

				// 	// Если текущая неделя отличается от последней обработанной
				// 	if year != lastProcessedYear || week != lastProcessedWeek {
				// 		logger.Info.Printf("New week detected: %d/%d (previous: %d/%d)", year, week, lastProcessedYear, lastProcessedWeek)

				// 		// Обновляем еженедельные рейтинги
				// 		if err := b.service.UpdateWeeklyRatings(); err != nil {
				// 			logger.Error.Printf("Error updating weekly ratings: %v", err)
				// 		} else {
				// 			logger.Info.Println("Successfully updated weekly ratings")
				// 		}

				// 		// Распределяем призы для предыдущей недели
				// 		now := time.Now()
				// 		yesterday := now.AddDate(0, 0, -1)
				// 		year, week := yesterday.ISOWeek()

				// 		if err := b.service.DistributePrizes(year, week); err != nil {
				// 			logger.Error.Printf("Error distributing prizes: %v", err)
				// 		} else {
				// 			logger.Info.Println("Successfully distributed prizes")
				// 		}

				// 		// Обновляем последнюю обработанную неделю
				// 		lastProcessedYear, lastProcessedWeek = year, week
				// 	}

				// 	// Проверяем отдельно, если сейчас понедельник, 00:00-01:00
				// 	// (дополнительная проверка для надежности)
				// 	if now.Weekday() == time.Monday && now.Hour() == 0 {
				// 		// Раз в неделю проверяем актуальность рейтинга для текущей недели
				// 		_, err := b.service.GetPrizeFund(year, week)
				// 		if err != nil {
				// 			logger.Error.Printf("Error checking prize fund for current week on Monday: %v", err)
				// 		}
				// 	}
			}
		}
	}()
}

// StartUpdateCaptcha запускает планировщик заданий для обновления капчи
func (b *Bot) StartUpdateCaptcha() {
	go func() {
		ratingTicker := time.NewTicker(30 * time.Second)
		defer ratingTicker.Stop()

		for range ratingTicker.C {
			cont, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			// Список всіх користувачів що можуть отримати оновлення капчі через бездіяльність
			allUsers, err := b.redisDB.LRange(cont, userCaptchaUpdateKey, 0, -1).Result()
			if err != nil {
				logger.Error.Printf("Failed to LRANGE all elements: %v", err)
			}

			if len(allUsers) == 0 {
				continue
			}

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

				if val > 3 {

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
					dbUser, err := b.service.GetUser(userID)
					if err != nil {
						logger.Error.Printf("Error GetUser %d: %v", userID, err)
						continue
					}

					// Всегда используем язык из базы данных, т.к. он может быть обновлен
					language := getLanguage(dbUser.LanguageCode, "")

					// Присилаємо нову капчу
					b.SendMessage(userID, b.captchaMessage(userID, language))
				}
			}
		}
	}()
}

// StartUpdateCache запускает планировщик кеша
func (b *Bot) StartUpdateCache() {
	go func() {

		localizationTicker := time.NewTicker(5 * time.Minute)
		prizeTicker := time.NewTicker(1 * time.Minute)
		defer localizationTicker.Stop()
		defer prizeTicker.Stop()

		b.refreshPrizeCache()
		b.refreshLocalizationCache()

		for {
			select {
			case <-prizeTicker.C:
				b.refreshPrizeCache()
				b.refreshActiveUsers()
			case <-prizeTicker.C:
				b.refreshLocalizationCache()
			}
		}
	}()
}

// Зберігаємо налаштування призового фонду в кеш
func (b *Bot) refreshPrizeCache() {
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

	b.gameHandler.prizeFundAmount = prizeFundAmount
	b.gameHandler.topCount = topCount
	b.gameHandler.totalPoints = totalPoints
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

	for userID := range data {

		logger.Error.Println("UpdateUserActivity", userID)
		if err := b.service.UpdateUserActivity(userID); err != nil {
			logger.Error.Printf("Error UpdateUserActivity: %d, %v", userID, err)
		}
	}

	// очищуєм map
	b.activeUsersMutex.Lock()
	b.activeUsers = map[int64]bool{}
	b.activeUsersMutex.Unlock()
}
