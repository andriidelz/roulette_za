package rotator

import (
	"context"
	"log"
	"roulette/internal/models"
	"roulette/internal/service"
	"time"
)

// Интерфейс для уведомления о новом хеше
type HashChangeNotifier interface {
	HandleNewRound(hashEntry *models.HashEntry)
}

// Settings содержит настройки ротатора
type Settings struct {
	Interval time.Duration
}

// Rotator отвечает за периодическую генерацию хешей и смену раундов
type Rotator struct {
	service    service.Service
	interval   time.Duration
	ctx        context.Context
	cancelFunc context.CancelFunc
	notifiers  []HashChangeNotifier
}

// NewRotator создает новый экземпляр ротатора
func NewRotator(service service.Service, interval time.Duration) *Rotator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Rotator{
		service:    service,
		interval:   interval,
		ctx:        ctx,
		cancelFunc: cancel,
		notifiers:  make([]HashChangeNotifier, 0),
	}
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
		r.notifyAll(newRound)
	}

	for {
		select {
		case <-r.ctx.Done():
			log.Println("Hash rotator stopped")
			return
		case <-ticker.C:
			// Генерируем новый хеш и завершаем текущий раунд
			newRound, err := r.service.StartNewRoundFromRotator()
			if err != nil {
				log.Printf("Error starting new round: %v", err)
				continue
			}

			log.Printf("Created new round #%d", newRound.ID)
			// Уведомляем всех обработчиков о новом раунде
			r.notifyAll(newRound)
		}
	}
}

// Stop останавливает процесс генерации хешей
func (r *Rotator) Stop() {
	log.Println("Stopping hash rotator...")
	r.cancelFunc()
}

// notifyAll уведомляет всех зарегистрированных обработчиков о новом раунде
func (r *Rotator) notifyAll(entry *models.HashEntry) {
	for _, notifier := range r.notifiers {
		go notifier.HandleNewRound(entry)
	}
}
