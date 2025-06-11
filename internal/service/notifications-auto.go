package service

import (
	"log"
	"roulette/internal/data"
	"roulette/internal/models"
	"roulette/internal/utils"
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

	// Получаем пользователя для определения языка
	user, err := s.service.GetRepo().GetUserByID(userID)
	if err != nil {
		return err
	}

	// Подготавливаем параметры для шаблона
	params := map[string]interface{}{
		"amount":  amount,
		"balance": user.Balance,
	}

	// Создаем и отправляем уведомление
	return s.sendTemplateNotification(balanceTemplate.ID, user, params)
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

	// Проверяем, созданы ли получатели
	_, total, err := s.service.GetRepo().GetNotificationRecipients(task.ID, "", 1, 10)
	if err != nil {
		log.Printf("Ошибка при проверке получателей для задачи %d: %v", task.ID, err)
	} else {
		log.Printf("Проверка получателей для задачи %d: найдено %d", task.ID, total)
	}

	return nil
}

// sendTemplateNotification отправляет уведомление на основе шаблона
func (s *AutoNotificationService) sendTemplateNotification(templateID uint, user *models.User, params map[string]interface{}) error {
	// Получаем шаблон с локализациями
	template, err := s.service.GetTemplateWithLocalizations(templateID)
	if err != nil {
		return err
	}

	// Определяем язык пользователя
	language := user.LanguageCode
	if language == "" {
		language = "en"
	}

	// Получаем локализованные тексты
	title := template.TitleLocalizations[language]
	message := template.MessageLocalizations[language]

	// Если локализация для языка пользователя не найдена, пробуем английскую версию
	if title == "" {
		title = template.TitleLocalizations["en"]
	}
	if message == "" {
		message = template.MessageLocalizations["en"]
	}

	// Если английской версии тоже нет, используем ключи
	if title == "" {
		title = template.TitleKey
	}
	if message == "" {
		message = template.MessageKey
	}

	// Получаем URL изображения для соответствующего языка
	imageURL := ""
	if images := template.GetImage(); len(images) > 0 {
		if url, ok := images[language]; ok {
			imageURL = url
		} else if url, ok := images["en"]; ok {
			imageURL = url
		}
	}

	// Определяем текст кнопки
	var buttonText string
	if template.ButtonTextKey != "" {
		buttonText = template.ButtonTextLocalizations[language]
		if buttonText == "" {
			buttonText = template.ButtonTextLocalizations["en"]
		}
		if buttonText == "" {
			buttonText = template.ButtonTextKey
		}
	}

	// Заменяем макросы в текстах с помощью общей функции
	title, message, buttonText = utils.ReplaceMacrosInTexts(title, message, buttonText, params)

	// Проверяем тип уведомления
	if template.TriggerEvent == "balance_updated" {
		// Для уведомлений о балансе создаем обычное уведомление и отправляем сразу
		notification := &models.Notification{
			UserID:    user.ID,
			Type:      template.Type,
			Title:     title,
			Message:   message,
			ImageURL:  imageURL,
			CreatedAt: time.Now(),
		}

		// Если есть кнопка, добавляем информацию о ней
		if template.ButtonTextKey != "" {
			notification.ButtonText = buttonText
			notification.ButtonURL = template.ButtonURL
			notification.ButtonCallback = template.ButtonCallback
		}

		// Сохраняем уведомление в базу данных для немедленной отправки
		return s.service.GetRepo().CreateNotification(notification)
	} else if template.TriggerEvent == "top_rating_entered" {
		// Для уведомлений о рейтинге создаем задачу, запланированную на 20:00 по локальному времени

		// Создаем параметры таргетирования для конкретного пользователя
		targetParams := models.NotificationTargetParams{
			UserIDs: []uint{user.ID},
		}

		// Учитываем часовой пояс пользователя, если указана страна
		if user.Country != "" {
			targetParams.TimeZone = data.GetTimezone(user.Country)
		}

		// Рассчитываем время отправки - 20:00 по локальному времени пользователя
		scheduledTime := s.calculateTimeFor8PM(user.Country)

		// Создаем задачу на отправку уведомления
		task := &models.NotificationTask{
			TemplateID:   template.ID,
			Status:       "pending",
			TargetType:   "custom",
			TargetParams: targetParams,
			ScheduledAt:  &scheduledTime,
			TotalUsers:   1,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Сохраняем задачу
		if err := s.service.GetRepo().CreateNotificationTask(task); err != nil {
			return err
		}

		// Задача будет выполнена планировщиком в запланированное время
		return nil
	}

	// Для всех остальных типов уведомлений - отправляем сразу
	notification := &models.Notification{
		UserID:    user.ID,
		Type:      template.Type,
		Title:     title,
		Message:   message,
		ImageURL:  imageURL,
		CreatedAt: time.Now(),
	}

	// Если есть кнопка, добавляем информацию о ней
	if template.ButtonTextKey != "" {
		notification.ButtonText = buttonText
		notification.ButtonURL = template.ButtonURL
		notification.ButtonCallback = template.ButtonCallback
	}

	// Сохраняем уведомление в базу данных
	return s.service.GetRepo().CreateNotification(notification)
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
