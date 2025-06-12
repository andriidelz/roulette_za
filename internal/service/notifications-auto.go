package service

import (
	"log"
	"roulette/internal/data"
	"roulette/internal/models"
	"time"
)

// AutoNotificationService отвечает за обработку и отправку автоматических уведомлений
type AutoNotificationService struct {
	service Service
}

// NewAutoNotificationService создает новый сервис автоматических уведомлений
func NewAutoNotificationService(service Service) *AutoNotificationService {
	return &AutoNotificationService{
		service: service,
	}
}

// HandleBalanceUpdated обрабатывает событие обновления баланса и отправляет уведомление
func (s *AutoNotificationService) HandleBalanceUpdated(userID uint, amount float64) error {
	// Создаем базовые параметры для макросов
	params := map[string]interface{}{
		"amount": amount,
	}

	// Получаем пользователя для определения языка и текущего баланса
	user, err := s.service.GetRepo().GetUserByID(userID)
	if err != nil {
		return err
	}

	// Добавляем текущий баланс в параметры
	params["balance"] = user.Balance

	// Находим соответствующий шаблон для события
	templates, _, err := s.service.GetNotificationTemplates("automatic", 1, 100)
	if err != nil {
		return err
	}

	var balanceTemplate *models.NotificationTemplate
	for _, tpl := range templates {
		if tpl.TriggerEvent == "balance_updated" && tpl.Active {
			balanceTemplate = &tpl
			break
		}
	}

	if balanceTemplate == nil {
		log.Printf("No active template found for 'balance_updated' event")
		return nil
	}

	// Создаем параметры таргетинга для конкретного пользователя
	targetParams := models.NotificationTargetParams{
		UserIDs: []uint{user.ID},
		Macros:  params, // Передаем параметры для макросов
	}

	log.Printf("Creating balance update notification task for user %d with params: %+v",
		userID, targetParams.Macros)

	// Создаем задачу на отправку уведомления (немедленно)
	task, err := s.service.CreateNotificationTask(balanceTemplate.ID, "custom", targetParams, nil)
	if err != nil {
		log.Printf("Error creating balance update notification task for user %d: %v", userID, err)
		return err
	}

	log.Printf("Created balance update notification task %d for user %d", task.ID, userID)
	return nil
}

// HandleTopRatingEntered обрабатывает событие входа в топ рейтинга и отправляет уведомление
func (s *AutoNotificationService) HandleTopRatingEntry(userID uint, position int) error {
	// Находим соответствующий шаблон для события
	templates, _, err := s.service.GetNotificationTemplates("automatic", 1, 100)
	if err != nil {
		return err
	}

	var ratingTemplate *models.NotificationTemplate
	for _, tpl := range templates {
		if tpl.TriggerEvent == "top_rating_entered" && tpl.Active {
			ratingTemplate = &tpl
			break
		}
	}

	if ratingTemplate == nil {
		log.Printf("No active template found for 'top_rating_entered' event")
		return nil
	}

	// Получаем пользователя для определения языка
	user, err := s.service.GetRepo().GetUserByID(userID)
	if err != nil {
		return err
	}

	log.Printf("Создаем уведомление о входе в топ рейтинга для пользователя %d, позиция: %d", userID, position)

	// Учитываем часовой пояс пользователя, если указана страна
	timeZone := "UTC"
	if user.Country != "" {
		timeZone = data.GetTimezone(user.Country)
		log.Printf("Определен часовой пояс для пользователя %d: %s", userID, timeZone)
	}

	// Рассчитываем время отправки - 20:00 по локальному времени пользователя
	scheduledTime := s.calculateTimeFor8PM(user.Country)
	log.Printf("Запланировано время отправки для пользователя %d: %s", userID, scheduledTime.Format("2006-01-02 15:04:05"))

	// Создаем параметры таргетирования для конкретного пользователя
	targetParams := models.NotificationTargetParams{
		UserIDs:  []uint{user.ID},
		TimeZone: timeZone,
	}

	// Используем метод CreateNotificationTask, который создаст получателей
	task, err := s.service.CreateNotificationTask(ratingTemplate.ID, "custom", targetParams, &scheduledTime)
	if err != nil {
		log.Printf("Ошибка при создании задачи уведомления: %v", err)
		return err
	}

	log.Printf("Создана задача уведомления %d для пользователя %d с %d получателями",
		task.ID, userID, task.TotalUsers)

	return nil
}

// calculateTimeFor8PM рассчитывает время для отправки в 20:00 по локальному времени пользователя
func (s *AutoNotificationService) calculateTimeFor8PM(country string) time.Time {
	now := time.Now()

	// Определяем часовой пояс пользователя
	loc := data.GetTimezoneLocation(country)

	// Получаем текущее локальное время пользователя
	userLocalTime := now.In(loc)

	// Устанавливаем время на 20:00 сегодня по локальному времени пользователя
	targetTime := time.Date(
		userLocalTime.Year(), userLocalTime.Month(), userLocalTime.Day(),
		20, 0, 0, 0, loc,
	)

	// Если 20:00 уже прошло сегодня, переносим на завтра
	if userLocalTime.After(targetTime) {
		targetTime = targetTime.AddDate(0, 0, 1)
	}

	// Преобразуем время обратно в UTC для сохранения в базе
	return targetTime.UTC()
}
