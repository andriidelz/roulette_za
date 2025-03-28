package rotator

import (
	"context"
	"log"
	"strconv"
	"time"

	"roulette/internal/messaging"
	"roulette/internal/service"
)

// PrizeScheduler отвечает за планирование и исполнение раздачи призов
type PrizeScheduler struct {
	service   service.Service
	ticker    *time.Ticker
	stopChan  chan struct{}
	lastRun   time.Time
	isRunning bool
}

// NewPrizeScheduler создает новый планировщик раздачи призов
func NewPrizeScheduler(service service.Service) *PrizeScheduler {
	return &PrizeScheduler{
		service:   service,
		stopChan:  make(chan struct{}),
		isRunning: false,
	}
}

// Start запускает планировщик раздачи призов
func (p *PrizeScheduler) Start() {
	if p.isRunning {
		log.Println("Prize scheduler is already running")
		return
	}

	// Устанавливаем интервал проверки в 1 минуту
	p.ticker = time.NewTicker(1 * time.Minute)
	p.isRunning = true

	log.Println("Starting prize distribution scheduler")

	go func() {
		for {
			select {
			case <-p.ticker.C:
				p.checkAndDistributePrizes()
			case <-p.stopChan:
				p.ticker.Stop()
				log.Println("Prize scheduler stopped")
				return
			}
		}
	}()
}

// Stop останавливает планировщик
func (p *PrizeScheduler) Stop() {
	if !p.isRunning {
		return
	}

	p.stopChan <- struct{}{}
	p.isRunning = false
}

// checkAndDistributePrizes проверяет, нужно ли раздавать призы в данный момент
func (p *PrizeScheduler) checkAndDistributePrizes() {
	// Получаем текущее время в UTC
	now := time.Now().UTC()

	// Получаем из настроек день недели для раздачи призов (1-7, где 1 - Понедельник)
	settings, err := p.service.GetSettings()
	if err != nil {
		log.Printf("Error getting settings for prize distribution: %v", err)
		return
	}

	// Определяем день недели для раздачи призов
	prizeDay := 1 // По умолчанию - понедельник
	if dayStr, ok := settings["prize_distribution_day"]; ok && dayStr != "" {
		if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 7 {
			prizeDay = day
		}
	}

	// Проверяем, совпадает ли текущий день недели с днем раздачи
	currentDay := int(now.Weekday())
	if currentDay == 0 {
		currentDay = 7 // Воскресенье в Go имеет индекс 0, преобразуем в 7
	}

	if currentDay != prizeDay {
		return // Не тот день недели
	}

	// Получаем время для раздачи призов
	prizeTimeStr := "00:00" // По умолчанию - полночь
	if timeStr, ok := settings["prize_distribution_time"]; ok && timeStr != "" {
		prizeTimeStr = timeStr
	}

	// Парсим время
	prizeTime, err := time.Parse("15:04", prizeTimeStr)
	if err != nil {
		log.Printf("Error parsing prize distribution time: %v", err)
		return
	}

	// Проверяем, прошло ли нужное время
	currentHour, currentMinute := now.Hour(), now.Minute()
	prizeHour, prizeMinute := prizeTime.Hour(), prizeTime.Minute()

	// Проверяем, текущий час и минута равны или уже прошли час и минуту раздачи призов,
	// но еще не прошло более 5 минут (чтобы не вызывать раздачу несколько раз подряд)
	if (currentHour > prizeHour || (currentHour == prizeHour && currentMinute >= prizeMinute)) &&
		(now.Sub(p.lastRun).Hours() >= 23) { // Не запускаем чаще раза в день
		// Проверяем, что прошло не более 5 минут с момента наступления времени раздачи
		targetTime := time.Date(now.Year(), now.Month(), now.Day(), prizeHour, prizeMinute, 0, 0, time.UTC)
		if now.Sub(targetTime).Minutes() <= 5 {
			// Запускаем процесс раздачи призов
			log.Println("Starting prize distribution process...")
			go p.distributePrizes()
			p.lastRun = now
		}
	}
}

// distributePrizes выполняет раздачу призов
func (p *PrizeScheduler) distributePrizes() {
	err := p.service.DistributePrizes()
	if err != nil {
		log.Printf("Error distributing prizes: %v", err)
		return
	}
	log.Println("Prize distribution completed successfully")
}

// Rotator отвечает за периодическую генерацию хешей и смену раундов
type Rotator struct {
	service        service.Service
	interval       time.Duration
	ctx            context.Context
	cancelFunc     context.CancelFunc
	rabbitmq       *messaging.RabbitMQ
	prizeScheduler *PrizeScheduler // Добавляем поле для планировщика
}

// Stop останавливает процесс генерации хешей
func (r *Rotator) Stop() {
	log.Println("Stopping hash rotator...")
	r.cancelFunc()

	// Останавливаем планировщик призов
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
