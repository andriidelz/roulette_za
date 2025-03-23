package rotator

import (
	"context"
	"log"
	"roulette/internal/messaging"
	"roulette/internal/models"
	"roulette/internal/service"
	"time"
)

// HashChangeNotifier интерфейс для уведомления компонентов о новом раунде
type HashChangeNotifier interface {
	HandleNewRound(hashEntry *models.HashEntry)
}

// Rotator отвечает за периодическую генерацию хешей и смену раундов
type Rotator struct {
	service    service.Service
	interval   time.Duration
	ctx        context.Context
	cancelFunc context.CancelFunc
	notifiers  []HashChangeNotifier
	rabbitmq   *messaging.RabbitMQ // Добавлен клиент RabbitMQ
}

// NewRotator создает новый экземпляр ротатора
func NewRotator(service service.Service, interval time.Duration, rabbitmqURL string) (*Rotator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем клиент RabbitMQ
	rmq, err := messaging.NewRabbitMQ(rabbitmqURL, "roulette_events", "rotator")
	if err != nil {
		return nil, err
	}

	return &Rotator{
		service:    service,
		interval:   interval,
		ctx:        ctx,
		cancelFunc: cancel,
		notifiers:  make([]HashChangeNotifier, 0),
		rabbitmq:   rmq,
	}, nil
}

// RegisterNotifier регистрирует обработчик для уведомлений о новом хеше
func (r *Rotator) RegisterNotifier(notifier HashChangeNotifier) {
	r.notifiers = append(r.notifiers, notifier)
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
	} else {
		log.Printf("Created initial round #%d", newRound.ID)

		// Отправляем сообщение о новом раунде через RabbitMQ
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := r.rabbitmq.PublishRoundStarted(ctx, newRound.ID, newRound); err != nil {
			log.Printf("Error publishing round started message: %v", err)
		}

		// Также продолжаем использовать существующий механизм уведомлений для совместимости
		r.notifyAll(newRound)
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
				log.Printf("No active round found to complete")
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

			// Отправляем сообщение о завершении раунда через RabbitMQ
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

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

			if err := r.rabbitmq.PublishRoundCompleted(ctx, currentRoundID, roundData); err != nil {
				log.Printf("Error publishing round completed message: %v", err)
			}
			cancel()

			// ВАЖНО: Добавляем задержку после публикации сообщения о завершении раунда
			// чтобы гарантировать, что оно будет обработано до сообщения о новом раунде
			time.Sleep(2 * time.Second)

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

			// Также продолжаем использовать существующий механизм уведомлений для совместимости
			r.notifyAll(newRound)
		}
	}
}

// Stop останавливает процесс генерации хешей
func (r *Rotator) Stop() {
	log.Println("Stopping hash rotator...")
	r.cancelFunc()

	// Закрываем соединение с RabbitMQ
	if r.rabbitmq != nil {
		if err := r.rabbitmq.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}
}

// notifyAll уведомляет всех зарегистрированных обработчиков о новом раунде
func (r *Rotator) notifyAll(entry *models.HashEntry) {
	for _, notifier := range r.notifiers {
		// Запускаем обработчики асинхронно, т.к. теперь у нас есть надежная система очередей
		// для обеспечения правильного порядка сообщений
		go notifier.HandleNewRound(entry)
	}
}
