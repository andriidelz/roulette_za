package rotator

import (
	"context"
	"log"
	"roulette/internal/messaging"
	"roulette/internal/service"
	"time"
)

// Rotator отвечает за периодическую генерацию хешей и смену раундов
type Rotator struct {
	service        service.Service
	interval       time.Duration
	ctx            context.Context
	cancelFunc     context.CancelFunc
	rabbitmq       *messaging.RabbitMQ
	prizeScheduler *PrizeScheduler // Добавляем поле для планировщика
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

	// Создаем и инициализируем планировщик призов
	rotator.prizeScheduler = NewPrizeScheduler(service)
	rotator.prizeScheduler.Start()

	return rotator, nil
}

// Start запускает процесс периодической смены раундов
func (r *Rotator) Start() {
	log.Printf("Starting hash rotator with interval: %s", r.interval)

	// Генерируем первый хеш сразу
	newRound, err := r.service.StartNewRoundFromRotator()
	if err != nil {
		log.Printf("Error starting initial round: %v", err)
	} else if newRound != nil {
		log.Printf("Created initial round #%d", newRound.ID)

		// Отправляем сообщение о новом раунде через RabbitMQ с высоким приоритетом
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
			log.Printf("Error publishing round started message: %v", err)
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
			log.Println("Hash rotator stopped")
			return
		default:
			// Получаем текущий активный раунд
			currentRound, err := r.service.GetCurrentRound()
			if err != nil {
				log.Printf("Error getting current round: %v", err)
				time.Sleep(1 * time.Second) // Короткая пауза перед повторной попыткой
				continue
			}

			if currentRound == nil {
				// Если активного раунда нет, создаем новый
				newRound, err := r.service.StartNewRoundFromRotator()
				if err != nil {
					log.Printf("Error starting new round: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				log.Printf("Created new round #%d (no active round found)", newRound.ID)

				// Отправляем сообщение о новом раунде через RabbitMQ
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
					log.Printf("Error publishing round started message: %v", err)
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
			log.Printf("Round #%d ended at exactly: %s", currentRoundID,
				roundEndTime.Format("15:04:05.000"))

			// Завершаем текущий раунд (на 15 секунде)
			if err := r.service.CompleteRound(currentRoundID); err != nil {
				log.Printf("Error completing round #%d: %v", currentRoundID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Запоминаем ID завершенного раунда
			lastCompletedRoundID = currentRoundID

			// Получаем обновленные данные о завершенном раунде
			completedRound, err := r.service.GetHashEntryByID(currentRoundID)
			if err != nil {
				log.Printf("Error getting completed round #%d: %v", currentRoundID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Создаем структуру с результатами раунда для отправки
			roundResult, err := r.service.GetRoundResult(currentRoundID)
			if err != nil {
				log.Printf("Error getting round result for #%d: %v", currentRoundID, err)
			}

			roundData := map[string]interface{}{
				"id":           completedRound.ID,
				"number":       completedRound.Number,
				"salt_hex":     completedRound.SaltHEX,
				"hash":         completedRound.Hash,
				"result":       string(roundResult),
				"created_at":   completedRound.CreatedAt,
				"completed_at": roundEndTime,
			}

			// Отправляем сообщение о завершении раунда через RabbitMQ с САМЫМ высоким приоритетом
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.rabbitmq.PublishRoundCompleted(ctx, currentRoundID, roundData); err != nil {
				log.Printf("Error publishing round completed message: %v", err)
			}
			cancel()

			// Добавляем дополнительное логирование
			log.Printf("Round #%d completed. Waiting exactly %v seconds for messaging to complete",
				currentRoundID, messagingPeriod.Seconds())

			// Ждем ровно 5 секунд - точное время для отправки всех уведомлений
			// После этого сразу начинаем новый раунд (на 20-й секунде)
			time.Sleep(messagingPeriod)

			// Точное время окончания отправки сообщений - 20-я секунда
			messageEndTime := time.Now()
			log.Printf("Messaging period for round #%d ended at: %s (after exactly 5 seconds)",
				currentRoundID, messageEndTime.Format("15:04:05.000"))

			// Получаем текущий активный раунд перед созданием нового
			// чтобы убедиться, что другой поток не создал его в промежутке
			currentCheck, _ := r.service.GetCurrentRound()
			if currentCheck != nil && currentCheck.ID != currentRoundID {
				log.Printf("New round #%d already created by another process. Skipping creation.", currentCheck.ID)
				continue
			}

			// Генерируем новый хеш и начинаем новый раунд
			// Это происходит сразу после 20-й секунды
			newRound, err := r.service.StartNewRoundFromRotator()
			if err != nil {
				log.Printf("Error starting new round: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Точное время начала нового раунда - сразу после 20-й секунды
			newRoundTime := time.Now()
			log.Printf("Created new round #%d at: %s",
				newRound.ID, newRoundTime.Format("15:04:05.000"))

			// Рассчитываем и выводим точное время цикла
			cycleDuration := newRoundTime.Sub(roundEndTime)
			log.Printf("Total cycle time from end of previous round to start of new round: %v",
				cycleDuration)

			// Отправляем сообщение о новом раунде через RabbitMQ
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
				log.Printf("Error publishing round started message: %v", err)
			}
			cancel()
		}
	}
}

// Stop останавливает процесс генерации хешей
func (r *Rotator) Stop() {
	log.Println("Stopping hash rotator...")
	r.cancelFunc()

	// Останавливаем планировщик призов, если он был инициализирован
	if r.prizeScheduler != nil {
		r.prizeScheduler.Stop()
	}

	// Закрываем соединение с RabbitMQ
	if r.rabbitmq != nil {
		if err := r.rabbitmq.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}
}
