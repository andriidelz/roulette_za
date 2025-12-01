package rotator

import (
	"context"
	"log"
	"sync"
	"time"

	"roulette/internal/logger"
	"roulette/internal/messaging"
	"roulette/internal/models"
	"roulette/internal/service"
)

// Rotator отвечает за периодическую генерацию хешей и смену раундов
type Rotator struct {
	service    service.Service
	interval   time.Duration
	ctx        context.Context
	cancelFunc context.CancelFunc
	rabbitmq   *messaging.RabbitMQ
	wg         sync.WaitGroup
}

// NewRotator создает новый экземпляр ротатора
func NewRotator(service service.Service, interval time.Duration, rabbitmqURL string) (*Rotator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем клиент RabbitMQ
	rmq, err := messaging.NewRabbitMQ(rabbitmqURL, "roulette_events", "rotator")
	if err != nil {
		cancel() // Освобождаем ресурсы в случае ошибки
		return nil, err
	}

	// Создаем экземпляр ротатора
	rotator := &Rotator{
		service:    service,
		interval:   interval,
		ctx:        ctx,
		cancelFunc: cancel,
		rabbitmq:   rmq,
	}

	return rotator, nil
}

// Start запускает процесс периодической смены раундов
func (r *Rotator) Start() {
	logger.Info.Printf("Starting hash rotator with interval: %s", r.interval)

	// Генерируем первый хеш сразу
	newRound, err := r.service.StartNewRoundFromRotator()
	if err != nil {
		logger.Error.Printf("Error starting initial round: %v", err)
	} else if newRound != nil {
		logger.Info.Printf("Created initial round #%d", newRound.ID)

		// Отправляем сообщение о новом раунде через RabbitMQ с высоким приоритетом
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
			logger.Error.Printf("Error publishing round started message: %v", err)
		}
	}

	// Параметры временных интервалов
	roundDuration := 15 * time.Second // Длительность раунда (15 секунд)

	// Точное время для отправки сообщений о результатах:
	// - 2 секунды до первого сообщения (стикер с результатом) на 17-й секунде
	// - 1 секунда до второго сообщения (текст результата) на 18-й секунде
	// - 1 секунда до третьего сообщения (стикер выигрыша/проигрыша) на 19-й секунде
	// - 1 секунда до четвертого сообщения (полное сообщение) на 20-й секунде
	messagingPeriod := 5 * time.Second

	// Переменная для хранения ID предыдущего раунда для отслеживания обработки
	var lastCompletedRoundID uint

	for {
		select {
		case <-r.ctx.Done():
			logger.Info.Println("Hash rotator stopped")
			return
		default:
			// Получаем текущий активный раунд
			currentRound, err := r.service.GetCurrentRound()
			if err != nil {
				logger.Error.Printf("Error getting current round: %v", err)
				time.Sleep(1 * time.Second) // Короткая пауза перед повторной попыткой
				continue
			}

			if currentRound == nil {
				// Если активного раунда нет, создаем новый
				newRound, err := r.service.StartNewRoundFromRotator()
				if err != nil {
					logger.Error.Printf("Error starting new round: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				logger.Info.Printf("Created new round #%d (no active round found)", newRound.ID)

				// Отправляем сообщение о новом раунде через RabbitMQ
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
					logger.Error.Printf("Error publishing round started message: %v", err)
				}
				cancel()

				// Ждем окончания раунда (15 секунд)
				time.Sleep(roundDuration)
				continue
			}

			// Запоминаем ID текущего раунда для завершения
			currentRoundID := currentRound.ID

			// Если мы уже обработали этот раунд, пропускаем его
			if currentRoundID == lastCompletedRoundID {
				time.Sleep(500 * time.Millisecond) // Небольшая пауза перед повторной проверкой
				continue
			}

			// Проверяем, пора ли завершать раунд
			elapsedTime := time.Since(currentRound.CreatedAt)
			if elapsedTime < roundDuration {
				// Раунд еще не завершен, ждем оставшееся время
				remainingTime := roundDuration - elapsedTime
				if remainingTime > 0 {
					time.Sleep(remainingTime)
				}
			}

			// Точное время окончания раунда - 15-я секунда
			roundEndTime := time.Now()
			logger.Info.Printf("Round #%d ended at exactly: %s", currentRoundID,
				roundEndTime.Format("15:04:05.000"))

			// Завершаем текущий раунд (на 15 секунде)
			if err := r.service.CompleteRound(currentRoundID); err != nil {
				logger.Error.Printf("Error completing round #%d: %v", currentRoundID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Запоминаем ID завершенного раунда
			lastCompletedRoundID = currentRoundID

			// Получаем обновленные данные о завершенном раунде
			completedRound, err := r.service.GetHashEntryByID(currentRoundID)
			if err != nil {
				logger.Error.Printf("Error getting completed round #%d: %v", currentRoundID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Обрабатываем ставки
			bets, err := r.service.ProcessAndGetBets(currentRoundID, completedRound.Number)
			if err != nil {
				logger.Error.Printf("Error processing bets for round #%d: %v", currentRoundID, err)
				// Продолжаем, даже если произошла ошибка
				bets = []models.Bet{} // Пустой список
			}

			// Обновляем рейтинг
			if len(bets) > 0 {
				year, week := time.Now().ISOWeek()

				// Capture variables for goroutine closure
				betsList := bets
				y := year
				w := week
				roundID := currentRoundID

				r.wg.Go(func() { // ← Без параметров!
					// Виконання йде в рутині щоб не зупиняти логіку ротатора. 
					// Через це перерахунок рейтингу відбувається з затримкою
					startTime := time.Now()

					// Собираем уникальные ID игроков
					userIDsMap := make(map[uint]bool)
					for _, bet := range betsList {
						userIDsMap[bet.UserID] = true
					}

					userIDs := make([]uint, 0, len(userIDsMap))
					for userID := range userIDsMap {
						userIDs = append(userIDs, userID)
					}

					// Один запрос вместо N запросов
					if err := r.service.UpdateWeeklyRatingForUsers(userIDs, y, w); err != nil {
						logger.Error.Printf("Error batch updating ratings: %v", err)
						return // Exit goroutine on error
					}

					// После обновления всех игроков - пересчитываем позиции
					if err := r.service.GetRepo().RefreshWeeklyRatingsPosition(y, w); err != nil {
						logger.Error.Printf("Error refreshing ratings positions: %v", err)
					} else {
						duration := time.Since(startTime)
						logger.Info.Printf("Rating updated for %d players in round #%d, positions refreshed (took %v)",
							len(userIDs), roundID, duration)
					}
				})
			}

			// Создаем структуру с результатами раунда для отправки
			option, err := r.service.GetRoundResult(completedRound.Number)
			if err != nil {
				logger.Error.Printf("Error getting round result for #%d: %v", currentRoundID, err)
			}

			roundData := map[string]interface{}{
				"id":           completedRound.ID,
				"number":       completedRound.Number,
				"salt_hex":     completedRound.SaltHEX,
				"hash":         completedRound.Hash,
				"result":       string(option),
				"created_at":   completedRound.CreatedAt,
				"completed_at": roundEndTime,
			}

			// Отправляем сообщение о завершении раунда через RabbitMQ с САМЫМ высоким приоритетом
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.rabbitmq.PublishRoundCompleted(ctx, currentRoundID, roundData); err != nil {
				logger.Error.Printf("Error publishing round completed message: %v", err)
			}
			cancel()

			// Ждем ровно 5 секунд - точное время для отправки всех уведомлений
			// После этого сразу начинаем новый раунд (на 20-й секунде)
			time.Sleep(messagingPeriod)

			// Получаем текущий активный раунд перед созданием нового
			// чтобы убедиться, что другой поток не создал его в промежутке
			currentCheck, err := r.service.GetCurrentRound()
			if err != nil {
				log.Println(err)
			}
			if currentCheck != nil && currentCheck.ID != currentRoundID {
				logger.Warning.Printf("New round #%d already created by another process. Skipping creation.", currentCheck.ID)
				continue
			}

			// Генерируем новый хеш и начинаем новый раунд
			// Это происходит сразу после 20-й секунды
			newRound, err := r.service.StartNewRoundFromRotator()
			if err != nil {
				logger.Error.Printf("Error starting new round: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Точное время начала нового раунда - сразу после 20-й секунды
			newRoundTime := time.Now()
			logger.Info.Printf("Created new round #%d at: %s",
				newRound.ID, newRoundTime.Format("15:04:05.000"))

			// Отправляем сообщение о новом раунде через RabbitMQ
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
				logger.Error.Printf("Error publishing round started message: %v", err)
			}
			cancel()
		}
	}
}

// Stop останавливает процесс генерации хешов
func (r *Rotator) Stop() {
	logger.Info.Println("Stopping hash rotator...")
	r.cancelFunc() // Stop round generation loop

	// Wait for active rating updates with timeout
	logger.Info.Println("Waiting for rating updates to complete...")
	done := make(chan struct{})
	go func() {
		r.wg.Wait() // Wait for all goroutines to finish
		close(done)
	}()

	select {
	case <-done:
		logger.Info.Println("All rating updates completed successfully")
	case <-time.After(30 * time.Second):
		logger.Warning.Println("Rating updates timeout exceeded - forcing shutdown")
	}

	// Закрываем соединение с RabbitMQ
	if r.rabbitmq != nil {
		if err := r.rabbitmq.Close(); err != nil {
			logger.Error.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}
}
