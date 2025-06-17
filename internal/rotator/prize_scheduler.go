package rotator

import (
	"log"
	"strconv"
	"time"

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
		lastRun:   time.Now().UTC().Add(-24 * time.Hour), // Инициализируем на 24 часа назад
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
	// Получаем настройки для определения дня раздачи призов
	settings, err := p.service.GetSettings()
	if err != nil {
		log.Printf("Error getting settings for prize distribution: %v", err)
		return
	}

	// Определяем день недели для раздачи призов из настроек
	prizeDay := 1 // По умолчанию - понедельник
	if dayStr, ok := settings["prize_distribution_day"]; ok && dayStr != "" {
		if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 7 {
			prizeDay = day
		}
	}

	// Получаем текущую дату
	now := time.Now()

	// Определяем дату для обработки предыдущей недели
	// Если сегодня день раздачи (например, четверг),
	// то мы хотим обработать предыдущую полную неделю (не текущую)
	var targetDate time.Time

	currentDay := int(now.Weekday())
	if currentDay == 0 {
		currentDay = 7 // Воскресенье в Go имеет индекс 0, преобразуем в 7
	}

	if currentDay == prizeDay {
		// Если сегодня день раздачи призов, обрабатываем предыдущую неделю
		// Находим последний день предыдущей недели (воскресенье предыдущей недели)
		daysToSubtract := currentDay + 7
		targetDate = now.AddDate(0, 0, -daysToSubtract)
	} else {
		// Для других дней, это может быть вызвано только если дата изменилась
		// в настройках или при ручном вызове - используем последний день предыдущей недели
		// Находим последнее воскресенье
		daysToLastSunday := currentDay
		if currentDay == 7 { // Если сегодня воскресенье
			daysToLastSunday = 7
		}
		targetDate = now.AddDate(0, 0, -daysToLastSunday)
	}

	// Получаем год и неделю для этой даты
	year, week := targetDate.ISOWeek()

	// Распределяем призы для найденной недели
	err = p.service.DistributePrizes(year, week)
	if err != nil {
		log.Printf("Error distributing prizes for week %d/%d: %v", year, week, err)
		return
	}
	log.Printf("Prize distribution for week %d/%d completed successfully", year, week)
}
