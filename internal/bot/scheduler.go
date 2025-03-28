package bot

import (
	"log"
	"time"
)

// StartRatingScheduler запускает планировщик заданий для обновления рейтингов
func (b *Bot) StartRatingScheduler() {
	go func() {
		// Периодическое обновление рейтингов (каждые 15 минут)
		ratingTicker := time.NewTicker(15 * time.Minute)
		defer ratingTicker.Stop()

		// Проверка на новую неделю (каждый час)
		weeklyResetTicker := time.NewTicker(1 * time.Hour)
		defer weeklyResetTicker.Stop()

		// Хранение информации о последней обработанной неделе
		var lastProcessedYear, lastProcessedWeek int

		// Сразу проверяем необходимость создания рейтинга для текущей недели
		currentYear, currentWeek := time.Now().ISOWeek()
		// При запуске проверим, существует ли рейтинг для текущей недели
		_, err := b.service.GetPrizeFund(currentYear, currentWeek)
		if err != nil {
			log.Printf("Error checking current week's prize fund: %v", err)
		}

		// Устанавливаем последнюю обработанную неделю
		lastProcessedYear, lastProcessedWeek = currentYear, currentWeek

		for {
			select {
			case <-ratingTicker.C:
				// Обновляем позиции в рейтинге
				if err := b.service.RefreshAllRatings(); err != nil {
					log.Printf("Error refreshing ratings: %v", err)
				} else {
					log.Println("Successfully refreshed ratings")
				}

			case <-weeklyResetTicker.C:
				// Проверяем, наступила ли новая неделя
				now := time.Now()
				year, week := now.ISOWeek()

				// Если текущая неделя отличается от последней обработанной
				if year != lastProcessedYear || week != lastProcessedWeek {
					log.Printf("New week detected: %d/%d (previous: %d/%d)", year, week, lastProcessedYear, lastProcessedWeek)

					// Обновляем еженедельные рейтинги
					if err := b.service.UpdateWeeklyRatings(); err != nil {
						log.Printf("Error updating weekly ratings: %v", err)
					} else {
						log.Println("Successfully updated weekly ratings")
					}

					// Распределяем призы для предыдущей недели
					if err := b.service.DistributePrizes(); err != nil {
						log.Printf("Error distributing prizes: %v", err)
					} else {
						log.Println("Successfully distributed prizes")
					}

					// Обновляем последнюю обработанную неделю
					lastProcessedYear, lastProcessedWeek = year, week
				}

				// Проверяем отдельно, если сейчас понедельник, 00:00-01:00
				// (дополнительная проверка для надежности)
				if now.Weekday() == time.Monday && now.Hour() == 0 {
					// Раз в неделю проверяем актуальность рейтинга для текущей недели
					_, err := b.service.GetPrizeFund(year, week)
					if err != nil {
						log.Printf("Error checking prize fund for current week on Monday: %v", err)
					}
				}
			}
		}
	}()
}
