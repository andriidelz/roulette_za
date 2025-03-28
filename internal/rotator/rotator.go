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

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

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

	for {
		select {
		case <-r.ctx.Done():
			log.Println("Hash rotator stopped")
			return
		case <-ticker.C:
			// Получаем текущий активный раунд для завершения
			currentRound, err := r.service.GetCurrentRound()
			if err != nil {
				log.Printf("Error getting current round: %v", err)
				continue
			}

			if currentRound == nil {
				// Если активного раунда нет, создаем новый
				newRound, err := r.service.StartNewRoundFromRotator()
				if err != nil {
					log.Printf("Error starting new round: %v", err)
					continue
				}

				log.Printf("Created new round #%d (no active round found)", newRound.ID)

				// Отправляем сообщение о новом раунде через RabbitMQ
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
					log.Printf("Error publishing round started message: %v", err)
				}
				cancel()
				continue
			}

			// Запоминаем ID текущего раунда для завершения
			currentRoundID := currentRound.ID

			// Завершаем текущий раунд
			if err := r.service.CompleteRound(currentRoundID); err != nil {
				log.Printf("Error completing round #%d: %v", currentRoundID, err)
				continue
			}

			// Получаем обновленные данные о завершенном раунде
			completedRound, err := r.service.GetHashEntryByID(currentRoundID)
			if err != nil {
				log.Printf("Error getting completed round #%d: %v", currentRoundID, err)
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
				"completed_at": time.Now(),
			}

			// Отправляем сообщение о завершении раунда через RabbitMQ с САМЫМ высоким приоритетом
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.rabbitmq.PublishRoundCompleted(ctx, currentRoundID, roundData); err != nil {
				log.Printf("Error publishing round completed message: %v", err)
			}
			cancel()

			// Даем небольшую задержку для обработки сообщения о завершении раунда
			time.Sleep(500 * time.Millisecond)

			// Генерируем новый хеш и начинаем новый раунд
			newRound, err := r.service.StartNewRoundFromRotator()
			if err != nil {
				log.Printf("Error starting new round: %v", err)
				continue
			}

			log.Printf("Created new round #%d", newRound.ID)

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
