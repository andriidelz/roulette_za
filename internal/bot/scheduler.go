package bot

import (
	"log"
	"time"
)

// StartRatingScheduler запускает планировщик заданий для обновления рейтингов
func (b *Bot) StartRatingScheduler() {
	go func() {
		// Периодическое обновление рейтингов
		ratingTicker := time.NewTicker(15 * time.Minute) // Обновляем каждые 15 минут
		defer ratingTicker.Stop()

		// Запуск обновления рейтингов по понедельникам
		weeklyResetTicker := time.NewTicker(1 * time.Hour) // Проверяем каждый час
		defer weeklyResetTicker.Stop()

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
				// Проверяем, нужно ли инициировать еженедельное обновление рейтинга
				now := time.Now()
				if now.Weekday() == time.Monday && now.Hour() == 0 {
					// Если сейчас понедельник и время от 00:00 до 01:00
					log.Println("Running weekly rating reset...")

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
				}
			}
		}
	}()
}
