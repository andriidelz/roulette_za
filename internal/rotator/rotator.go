package rotator

import (
	"context"
	"fmt"
	"log"
	"roulette/internal/models"
	"roulette/internal/service"
	"roulette/internal/utils"
	"time"
)

// Интерфейс для уведомления о новом хеше
type HashChangeNotifier interface {
	HandleNewHash(hashEntry *models.HashEntry)
}

// Settings содержит настройки ротатора
type Settings struct {
	Interval time.Duration
}

// Rotator отвечает за периодическую генерацию хешей
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

// Start запускает процесс периодической генерации хешей
func (r *Rotator) Start() {
	log.Printf("Starting hash rotator with interval: %s", r.interval)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Генерируем первый хеш сразу
	entry := r.generateAndLogHash()
	if entry != nil {
		r.notifyAll(entry)
	}

	for {
		select {
		case <-r.ctx.Done():
			log.Println("Hash rotator stopped")
			return
		case <-ticker.C:
			entry := r.generateAndLogHash()
			if entry != nil {
				r.notifyAll(entry)
			}
		}
	}
}

// Stop останавливает процесс генерации хешей
func (r *Rotator) Stop() {
	log.Println("Stopping hash rotator...")
	r.cancelFunc()
}

// generateAndLogHash генерирует новый хеш и выводит информацию в лог
func (r *Rotator) generateAndLogHash() *models.HashEntry {
	entry, err := r.service.GenerateHashEntry()
	if err != nil {
		log.Printf("Error generating hash: %v", err)
		return nil
	}

	// Подготовка данных для рамки
	orderedData := []utils.KeyValue{
		{Key: "ID/Base62", Value: fmt.Sprintf("%d/%s", entry.ID, utils.ToBase62(entry.ID))},
		{Key: "Hash", Value: entry.Hash},
		{Key: "Color", Value: utils.GetColorForNumber(entry.Number)},
		{Key: "Number", Value: fmt.Sprintf("%d", entry.Number)},
		{Key: "Salt (HEX)", Value: entry.SaltHEX},
	}

	// Выведение рамки с данными в заданном порядке
	utils.PrintOrderedTextInFrame(orderedData, utils.DoubleBorderFrameStyle())

	return entry
}

// notifyAll уведомляет всех зарегистрированных обработчиков о новом хеше
func (r *Rotator) notifyAll(entry *models.HashEntry) {
	for _, notifier := range r.notifiers {
		go notifier.HandleNewHash(entry)
	}
}
