package rotator

import (
	"context"
	"fmt"
	"log"
	"roulette/internal/conv"
	"roulette/internal/service"
	"roulette/internal/utils"
	"time"
)

// Settings містить налаштування ротатора
type Settings struct {
	Interval time.Duration
}

// Rotator відповідає за періодичну генерацію хешів
type Rotator struct {
	service    service.Service
	interval   time.Duration
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewRotator створює новий екземпляр ротатора
func NewRotator(service service.Service, interval time.Duration) *Rotator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Rotator{
		service:    service,
		interval:   interval,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Start запускає процес періодичної генерації хешів
func (r *Rotator) Start() {
	log.Printf("Starting hash rotator with interval: %s", r.interval)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Генеруємо перший хеш одразу
	r.generateAndLogHash()

	for {
		select {
		case <-r.ctx.Done():
			log.Println("Hash rotator stopped")
			return
		case <-ticker.C:
			r.generateAndLogHash()
		}
	}
}

// Stop зупиняє процес генерації хешів
func (r *Rotator) Stop() {
	log.Println("Stopping hash rotator...")
	r.cancelFunc()
}

// generateAndLogHash генерує новий хеш та виводить інформацію в лог
func (r *Rotator) generateAndLogHash() {
	entry, err := r.service.GenerateHashEntry()
	if err != nil {
		log.Printf("Error generating hash: %v", err)
		return
	}

	// Підготовка даних для рамки
	orderedData := []utils.KeyValue{
		{Key: "ID/Base62", Value: conv.String(entry.ID) + "/" + utils.ToBase62(entry.ID)},
		{Key: "Hash", Value: entry.Hash},
		{Key: "Color", Value: utils.GetColorForNumber(entry.Number)},
		{Key: "Number", Value: fmt.Sprintf("%d", entry.Number)},
		{Key: "Salt (HEX)", Value: entry.SaltHEX},
	}

	// Виведення рамки з даними в заданому порядку
	utils.PrintOrderedTextInFrame(orderedData, utils.DoubleBorderFrameStyle())
}
