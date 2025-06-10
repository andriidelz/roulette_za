package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"roulette/internal/data"
	"roulette/internal/messaging"
	"roulette/internal/models"
	"roulette/internal/utils"
)

// Constants for RabbitMQ messaging
const (
	// Event type for notifications
	EventUserNotification = "user_notification"

	// Routing key for notifications
	RoutingUserNotification = "user.notification"
)

// NotificationData структура для передачи данных уведомления через RabbitMQ
type NotificationData struct {
	UserID         uint   `json:"user_id"`
	TelegramID     int64  `json:"telegram_id"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	ImageURL       string `json:"image_url,omitempty"`
	ButtonText     string `json:"button_text,omitempty"`
	ButtonURL      string `json:"button_url,omitempty"`
	ButtonCallback string `json:"button_callback,omitempty"`
	NotificationID uint   `json:"notification_id,omitempty"`
}

// GetNotificationTemplates получает список шаблонов уведомлений
func (s *ServiceImpl) GetNotificationTemplates(templateType string, page, perPage int) ([]models.NotificationTemplate, int64, error) {
	return s.repo.GetNotificationTemplates(templateType, page, perPage)
}

// GetNotificationTemplateByID получает шаблон уведомления по ID
func (s *ServiceImpl) GetNotificationTemplateByID(id uint) (*models.NotificationTemplate, error) {
	return s.repo.GetNotificationTemplateByID(id)
}

// CreateNotificationTemplate создает новый шаблон уведомления
func (s *ServiceImpl) CreateNotificationTemplate(template *models.NotificationTemplate) error {
	// Проверяем наличие ключей локализации
	if template.TitleKey == "" || template.MessageKey == "" {
		return errors.New("ключи локализации для заголовка и сообщения обязательны")
	}

	// Если кнопка имеет текст, но не указан ключ локализации, генерируем его
	if template.ButtonTextKey == "" && (template.ButtonURL != "" || template.ButtonCallback != "") {
		template.ButtonTextKey = "notification_button_" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	// Проверяем поле ImageURLs
	if template.ImageURLs == "" {
		template.ImageURLs = "{}"
	}

	return s.repo.CreateNotificationTemplate(template)
}

// UpdateNotificationTemplate обновляет шаблон уведомления
func (s *ServiceImpl) UpdateNotificationTemplate(template *models.NotificationTemplate) error {
	// Проверяем наличие ключей локализации
	if template.TitleKey == "" || template.MessageKey == "" {
		return errors.New("ключи локализации для заголовка и сообщения обязательны")
	}

	// Если кнопка имеет текст, но не указан ключ локализации, генерируем его
	if template.ButtonTextKey == "" && (template.ButtonURL != "" || template.ButtonCallback != "") {
		template.ButtonTextKey = "notification_button_" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	return s.repo.UpdateNotificationTemplate(template)
}

// DeleteNotificationTemplate удаляет шаблон уведомления
func (s *ServiceImpl) DeleteNotificationTemplate(id uint) error {
	return s.repo.DeleteNotificationTemplate(id)
}

// GetNotificationTasks получает список задач уведомлений
func (s *ServiceImpl) GetNotificationTasks(status string, page, perPage int) ([]models.NotificationTask, int64, error) {
	return s.repo.GetNotificationTasks(status, page, perPage)
}

// GetNotificationTaskByID получает задачу уведомления по ID
func (s *ServiceImpl) GetNotificationTaskByID(id uint) (*models.NotificationTask, error) {
	return s.repo.GetNotificationTaskByID(id)
}

// GetEnhancedNotificationTask получает задачу уведомления с дополнительной информацией
func (s *ServiceImpl) GetEnhancedNotificationTask(id uint) (*models.EnhancedNotificationTask, error) {
	// Получаем базовую задачу
	task, err := s.repo.GetNotificationTaskByID(id)
	if err != nil {
		return nil, err
	}

	// Получаем шаблон с локализациями
	templateWithLocalizations, err := s.repo.GetTemplateWithLocalizations(task.TemplateID)
	if err != nil {
		return nil, err
	}

	// Расчет прогресса выполнения
	var progress float64
	if task.TotalUsers > 0 {
		progress = float64(task.SentCount) / float64(task.TotalUsers) * 100
	}

	// Расчет оставшегося времени
	var estimatedTimeRemaining string
	if task.Status == "processing" && task.StartedAt != nil && task.SentCount > 0 && task.TotalUsers > task.SentCount {
		// Расчет на основе времени начала и текущего прогресса
		elapsedTime := time.Since(*task.StartedAt)
		avgTimePerMessage := elapsedTime / time.Duration(task.SentCount)
		remainingMessages := task.TotalUsers - task.SentCount
		estimatedTime := avgTimePerMessage * time.Duration(remainingMessages)

		// Форматирование времени
		hours := int(estimatedTime.Hours())
		minutes := int(estimatedTime.Minutes()) % 60
		if hours > 0 {
			estimatedTimeRemaining = fmt.Sprintf("%dч %dмин", hours, minutes)
		} else {
			estimatedTimeRemaining = fmt.Sprintf("%dмин", minutes)
		}
	}

	// Создаем расширенный объект задачи
	enhancedTask := &models.EnhancedNotificationTask{
		NotificationTask:          *task,
		TemplateWithLocalizations: *templateWithLocalizations,
		TaskProgress:              progress,
		EstimatedTimeRemaining:    estimatedTimeRemaining,
	}

	return enhancedTask, nil
}

// CreateNotificationTask создает задачу на отправку уведомлений
func (s *ServiceImpl) CreateNotificationTask(templateID uint, targetType string, targetParams models.NotificationTargetParams, scheduledAt *time.Time) (*models.NotificationTask, error) {
	// Проверяем существование шаблона
	_, err := s.repo.GetNotificationTemplateByID(templateID)
	if err != nil {
		return nil, err
	}

	// Создаем объект задачи
	task := &models.NotificationTask{
		TemplateID:   templateID,
		Status:       "pending",
		TargetType:   targetType,
		TargetParams: targetParams,
		ScheduledAt:  scheduledAt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Находим пользователей, соответствующих критериям таргетинга
	users, err := s.repo.GetUsersForNotificationTask(task)
	if err != nil {
		return nil, err
	}

	// Если пользователи не найдены, возвращаем ошибку
	if len(users) == 0 {
		return nil, fmt.Errorf("пользователи для отправки не найдены")
	}

	// Устанавливаем общее количество пользователей
	task.TotalUsers = len(users)

	// Сохраняем задачу
	if err := s.repo.CreateNotificationTask(task); err != nil {
		return nil, err
	}

	log.Printf("Создана задача уведомлений %d, найдено %d получателей", task.ID, len(users))

	// Создаем получателей
	recipients := make([]models.NotificationRecipient, 0, len(users))
	for _, user := range users {
		// Определяем время отправки с учетом часового пояса
		var recipientScheduledAt *time.Time
		if scheduledAt != nil {
			// Определяем часовой пояс пользователя на основе страны
			userTimeZone := data.GetTimezone(user.Country)
			localTime := adjustTimeToUserTimeZone(*scheduledAt, userTimeZone, targetParams.SendTimeStart, targetParams.SendTimeEnd)
			recipientScheduledAt = &localTime
		}

		recipient := models.NotificationRecipient{
			TaskID:      task.ID,
			UserID:      user.ID,
			Status:      "pending",
			ScheduledAt: recipientScheduledAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		recipients = append(recipients, recipient)
	}

	// Сохраняем получателей пакетно
	if err := s.repo.CreateNotificationRecipientsBatch(recipients); err != nil {
		log.Printf("Ошибка при создании получателей для задачи %d: %v", task.ID, err)
		return nil, err
	}

	log.Printf("Создано %d получателей для задачи %d", len(recipients), task.ID)

	// Проверяем, созданы ли получатели
	recipients, total, err := s.repo.GetNotificationRecipients(task.ID, "", 1, 1)
	if err != nil {
		log.Printf("Ошибка при проверке получателей для задачи %d: %v", task.ID, err)
	} else if total == 0 {
		log.Printf("ВНИМАНИЕ: Для задачи %d не создано ни одного получателя!", task.ID)
	} else {
		log.Printf("Задача %d: создано %d получателей", task.ID, total)
	}

	return task, nil
}

// SendNotifications отправляет уведомления для задачи
func (s *ServiceImpl) SendNotifications(taskID uint) error {
	// Получаем задачу
	task, err := s.repo.GetNotificationTaskByID(taskID)
	if err != nil {
		log.Printf("Ошибка при получении задачи %d: %v", taskID, err)
		return err
	}

	// Проверяем, что задача в статусе pending
	if task.Status != "pending" {
		log.Printf("Задача %d не может быть запущена, статус: %s", taskID, task.Status)
		return fmt.Errorf("задача уже выполняется или завершена")
	}

	// Получаем шаблон
	template, err := s.repo.GetNotificationTemplateByID(task.TemplateID)
	if err != nil {
		log.Printf("Ошибка при получении шаблона %d для задачи %d: %v", task.TemplateID, taskID, err)
		return err
	}

	// Обновляем статус и время начала
	now := time.Now()
	task.Status = "processing"
	task.StartedAt = &now
	task.UpdatedAt = now
	if err := s.repo.UpdateNotificationTask(task); err != nil {
		log.Printf("Ошибка при обновлении статуса задачи %d: %v", taskID, err)
		return err
	}

	log.Printf("Начинаем обработку задачи уведомлений %d (шаблон: %s)", taskID, template.Name)

	// Запускаем отправку в фоновом режиме
	go func() {
		// Получаем получателей для задачи
		page := 1
		pageSize := 100
		sentCount := 0
		deliveredCount := 0
		readCount := 0
		hasDelayedRecipients := false

		// Получаем общее количество получателей
		_, total, err := s.repo.GetNotificationRecipients(taskID, "", 1, 1)
		if err != nil {
			log.Printf("Ошибка при получении общего количества получателей для задачи %d: %v", taskID, err)
		} else {
			log.Printf("Задача %d: найдено %d получателей", taskID, total)
		}

		// Основной цикл обработки получателей
		for {
			recipients, _, err := s.repo.GetNotificationRecipients(taskID, "pending", page, pageSize)
			if err != nil {
				log.Printf("Ошибка при получении получателей для задачи %d: %v", taskID, err)
				break
			}

			log.Printf("Задача %d: получено %d получателей на странице %d", taskID, len(recipients), page)

			if len(recipients) == 0 {
				log.Printf("Задача %d: больше нет получателей для обработки", taskID)
				break
			}

			for _, recipient := range recipients {
				// Проверяем время отправки
				if recipient.ScheduledAt != nil && recipient.ScheduledAt.After(time.Now()) {
					log.Printf("Задача %d: пропускаем получателя %d, время отправки еще не наступило: %s",
						taskID, recipient.ID, recipient.ScheduledAt.Format("2006-01-02 15:04:05"))
					hasDelayedRecipients = true
					continue
				}

				// Получаем пользователя для более информативного логирования
				user, err := s.repo.GetUserByID(recipient.UserID)
				if err != nil {
					log.Printf("Задача %d: ошибка при получении пользователя %d: %v",
						taskID, recipient.UserID, err)
					continue
				}

				log.Printf("Задача %d: отправка уведомления пользователю %d (TelegramID: %d)",
					taskID, recipient.UserID, user.TelegramID)

				// Отправляем уведомление пользователю
				err = s.sendNotificationToUser(recipient.UserID, template, task)
				nowSent := time.Now()
				recipient.SentAt = &nowSent
				recipient.UpdatedAt = nowSent

				if err != nil {
					// Обновляем статус и сообщение об ошибке
					recipient.Status = "failed"
					recipient.ErrorMessage = err.Error()
					log.Printf("Задача %d: ошибка при отправке уведомления пользователю %d: %v",
						taskID, recipient.UserID, err)
				} else {
					// Обновляем статус
					recipient.Status = "sent"
					sentCount++
					log.Printf("Задача %d: успешно отправлено уведомление пользователю %d (TelegramID: %d)",
						taskID, recipient.UserID, user.TelegramID)
				}

				// Сохраняем обновления получателя
				if err := s.repo.UpdateNotificationRecipient(&recipient); err != nil {
					log.Printf("Задача %d: ошибка при обновлении получателя %d: %v",
						taskID, recipient.ID, err)
				}
			}

			// Обновляем прогресс задачи
			if err := s.repo.UpdateTaskProgress(taskID, sentCount, deliveredCount, readCount); err != nil {
				log.Printf("Задача %d: ошибка при обновлении прогресса: %v", taskID, err)
			} else {
				log.Printf("Задача %d: прогресс обновлен: отправлено %d, доставлено %d, прочитано %d",
					taskID, sentCount, deliveredCount, readCount)
			}

			page++
		}

		// Проверяем, есть ли отложенные получатели
		if hasDelayedRecipients {
			// Если есть отложенные получатели, возвращаем задачу в статус "pending"
			pendingTask, err := s.repo.GetNotificationTaskByID(taskID)
			if err != nil {
				log.Printf("Задача %d: ошибка при получении задачи для возврата в статус pending: %v", taskID, err)
				return
			}

			pendingTask.Status = "pending"
			pendingTask.SentCount = sentCount
			pendingTask.DeliveredCount = deliveredCount
			pendingTask.ReadCount = readCount
			pendingTask.UpdatedAt = time.Now()

			if err := s.repo.UpdateNotificationTask(pendingTask); err != nil {
				log.Printf("Задача %d: ошибка при возврате задачи в статус pending: %v", taskID, err)
			} else {
				log.Printf("Задача %d: возвращена в статус pending из-за наличия отложенных получателей", taskID)
			}
		} else {
			// Если отложенных получателей нет, помечаем задачу как завершенную
			completedTask, err := s.repo.GetNotificationTaskByID(taskID)
			if err != nil {
				log.Printf("Задача %d: ошибка при получении задачи для завершения: %v", taskID, err)
				return
			}

			completedNow := time.Now()
			completedTask.Status = "completed"
			completedTask.CompletedAt = &completedNow
			completedTask.SentCount = sentCount
			completedTask.DeliveredCount = deliveredCount
			completedTask.ReadCount = readCount
			completedTask.UpdatedAt = completedNow

			if err := s.repo.UpdateNotificationTask(completedTask); err != nil {
				log.Printf("Задача %d: ошибка при завершении задачи: %v", taskID, err)
			} else {
				log.Printf("Задача %d завершена. Отправлено уведомлений: %d", taskID, sentCount)
			}
		}
	}()

	return nil
}

// DeleteNotificationTask удаляет задачу вместе с получателями
func (s *ServiceImpl) DeleteNotificationTask(id uint) error {
	if id == 0 {
		return fmt.Errorf("невалидный ID задачи")
	}

	// Получаем задачу для проверки
	task, err := s.repo.GetNotificationTaskByID(id)
	if err != nil {
		return fmt.Errorf("задача с ID %d не найдена: %w", id, err)
	}

	// Проверяем статус
	if task.Status == "processing" || task.Status == "sending" {
		return fmt.Errorf("нельзя удалить задачу в статусе '%s'", task.Status)
	}

	// Проверка на существование активных получателей
	recipients, _, err := s.repo.GetNotificationRecipients(id, "pending", 1, 1)
	if err == nil && len(recipients) > 0 {
		return fmt.Errorf("нельзя удалить задачу с активными получателями")
	}

	// Логируем операцию
	log.Printf("Удаление задачи уведомлений %d (шаблон: %s)", id, task.Template.Name)

	// Удаляем через репозиторий
	if err := s.repo.DeleteNotificationTask(id); err != nil {
		return fmt.Errorf("ошибка при удалении задачи: %w", err)
	}

	log.Printf("Задача уведомлений %d успешно удалена", id)
	return nil
}

// UpdateNotificationTask обновляет задачу
func (s *ServiceImpl) UpdateNotificationTask(task *models.NotificationTask) error {
	return s.repo.UpdateNotificationTask(task)
}

// GetNotificationTasksStats получает статистику задач уведомлений
func (s *ServiceImpl) GetNotificationTasksStats(period string) (*models.NotificationStatistics, error) {
	return s.repo.GetNotificationTasksStats(period)
}

// GetCountriesWithUserCounts получает список стран с количеством пользователей
func (s *ServiceImpl) GetCountriesWithUserCounts() ([]models.CountryOption, error) {
	return s.repo.GetCountriesWithUserCounts()
}

// GetActivityFiltersWithUserCounts получает список фильтров активности с количеством пользователей
func (s *ServiceImpl) GetActivityFiltersWithUserCounts() ([]models.ActivityFilterOption, error) {
	return s.repo.GetActivityFiltersWithUserCounts()
}

// CancelNotificationTask отменяет задачу
func (s *ServiceImpl) CancelNotificationTask(id uint) error {
	task, err := s.repo.GetNotificationTaskByID(id)
	if err != nil {
		return err
	}

	// Проверяем, что задача может быть отменена
	if task.Status != "pending" && task.Status != "processing" {
		return errors.New("задача не может быть отменена")
	}

	// Обновляем статус
	task.Status = "canceled"
	task.UpdatedAt = time.Now()
	return s.repo.UpdateNotificationTask(task)
}

// adjustTimeToUserTimeZone корректирует время с учетом часового пояса пользователя
func adjustTimeToUserTimeZone(scheduledTime time.Time, userTimeZone, sendTimeStart, sendTimeEnd string) time.Time {
	// Получаем часовой пояс пользователя
	loc, err := time.LoadLocation(userTimeZone)
	if err != nil {
		// В случае ошибки используем UTC
		loc = time.UTC
	}

	// Преобразуем время в часовой пояс пользователя
	localTime := scheduledTime.In(loc)

	// Проверяем, попадает ли время в разрешенный диапазон
	if sendTimeStart != "" && sendTimeEnd != "" {
		// Парсим время начала и конца
		startTime, startErr := time.Parse("15:04", sendTimeStart)
		endTime, endErr := time.Parse("15:04", sendTimeEnd)

		if startErr == nil && endErr == nil {
			// Получаем часы и минуты из localTime
			hour, min, _ := localTime.Clock()
			currentTimeOfDay := time.Date(1, 1, 1, hour, min, 0, 0, time.UTC)

			// Проверяем, находится ли currentTimeOfDay между startTime и endTime
			if currentTimeOfDay.Before(startTime) || currentTimeOfDay.After(endTime) {
				// Если время не в разрешенном диапазоне, перемещаем его на время начала следующего дня
				nextDay := localTime.AddDate(0, 0, 1)
				return time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(),
					startTime.Hour(), startTime.Minute(), 0, 0, loc)
			}
		}
	}

	return localTime
}

// GetTemplateWithLocalizations получает шаблон с локализациями
func (s *ServiceImpl) GetTemplateWithLocalizations(templateID uint) (*models.NotificationTemplateWithLocalizations, error) {
	return s.repo.GetTemplateWithLocalizations(templateID)
}

// HandleBalanceUpdate обрабатывает изменение баланса и отправляет уведомление при необходимости
func (s *ServiceImpl) HandleBalanceUpdate(userID uint, amount float64) error {
	// Создаем сервис автоматических уведомлений
	autoNotif := NewAutoNotificationService(s)

	// Отправляем уведомление о пополнении баланса
	return autoNotif.HandleBalanceUpdated(userID, amount)
}

// HandleTopRatingEntry обрабатывает вход пользователя в топ рейтинга
func (s *ServiceImpl) HandleTopRatingEntry(userID uint, position int) error {
	// Создаем сервис автоматических уведомлений
	autoNotif := NewAutoNotificationService(s)

	// Отправляем уведомление о входе в топ рейтинга
	return autoNotif.HandleTopRatingEntry(userID, position)
}

func (s *ServiceImpl) GetPendingNotificationTasks() ([]models.NotificationTask, error) {
	return s.repo.GetPendingNotificationTasks()
}

func (s *ServiceImpl) GetPendingNotifications() ([]models.Notification, error) {
	return s.repo.GetPendingNotifications()
}

// SendNotification отправляет уведомление немедленно
func (s *ServiceImpl) SendNotification(notification *models.Notification) error {
	// Получаем пользователя
	user, err := s.repo.GetUserByID(notification.UserID)
	if err != nil {
		return err
	}

	// Создаем карту данных для отправки через RabbitMQ
	// Это наиболее надежный способ передачи структурированных данных
	notificationDataMap := map[string]interface{}{
		"user_id":         user.ID,
		"telegram_id":     user.TelegramID,
		"title":           notification.Title,
		"message":         notification.Message,
		"image_url":       notification.ImageURL,
		"button_text":     notification.ButtonText,
		"button_url":      notification.ButtonURL,
		"button_callback": notification.ButtonCallback,
		"notification_id": notification.ID,
	}

	// Подключаемся к RabbitMQ
	rmq, err := messaging.NewRabbitMQ(s.getRabbitMQURL(), "roulette_events", "notification_service")
	if err != nil {
		log.Printf("Error connecting to RabbitMQ: %v", err)
		return err
	}
	defer rmq.Close()

	// Устанавливаем таймаут для отправки сообщения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Публикуем карту данных напрямую через RabbitMQ
	// Это позволит избежать проблем с сериализацией/десериализацией
	if err := rmq.Publish(ctx, RoutingUserNotification, EventUserNotification, 0, notificationDataMap, messaging.PriorityNormal); err != nil {
		log.Printf("Error publishing notification to RabbitMQ: %v", err)
		return err
	}

	log.Printf("Notification queued for user %d (%d): %s - %s",
		user.ID, user.TelegramID, notification.Title, notification.Message)

	// Помечаем уведомление как отправленное в базе данных только после успешной публикации
	return s.repo.MarkNotificationAsSent(notification.ID)
}

// sendNotificationToUser отправляет уведомление из задач notification_tasks
func (s *ServiceImpl) sendNotificationToUser(userID uint, template *models.NotificationTemplate, task *models.NotificationTask) error {
	// Получаем пользователя
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Получаем локализованные тексты
	language := user.LanguageCode
	if language == "" {
		language = "en" // По умолчанию английский
	}

	title, err := s.repo.GetLocalization(template.TitleKey, language)
	if err != nil {
		// Если локализация не найдена, пробуем английскую
		title, err = s.repo.GetLocalization(template.TitleKey, "en")
		if err != nil {
			title = template.TitleKey // Используем ключ как текст
		}
	}

	message, err := s.repo.GetLocalization(template.MessageKey, language)
	if err != nil {
		// Если локализация не найдена, пробуем английскую
		message, err = s.repo.GetLocalization(template.MessageKey, "en")
		if err != nil {
			message = template.MessageKey // Используем ключ как текст
		}
	}

	var buttonText string
	var buttonURL string
	var buttonCallback string

	if template.ButtonTextKey != "" {
		buttonText, err = s.repo.GetLocalization(template.ButtonTextKey, language)
		if err != nil {
			// Если локализация не найдена, пробуем английскую
			buttonText, err = s.repo.GetLocalization(template.ButtonTextKey, "en")
			if err != nil {
				buttonText = template.ButtonTextKey // Используем ключ как текст
			}
		}
		buttonURL = template.ButtonURL
		buttonCallback = template.ButtonCallback
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

	// Собираем параметры для замены макросов в зависимости от типа уведомления
	params := make(map[string]interface{})

	if template.TriggerEvent == "top_rating_entered" {
		// Получаем текущий год и неделю
		year, week := time.Now().ISOWeek()

		// Получаем рейтинг пользователя
		rating, err := s.repo.GetUserWeeklyRating(user.ID, year, week)
		if err == nil && rating != nil {
			// Добавляем позицию и очки пользователя
			params["position"] = rating.Position
			params["points"] = rating.Points
		}

		// Получаем призовой фонд
		prizeFund, err := s.repo.GetPrizeFund(year, week)
		if err == nil && prizeFund != nil {
			// Добавляем сумму призового фонда
			params["prize_fund"] = prizeFund.Amount
		}
	}

	// Заменяем макросы в текстах с помощью общей функции
	title, message, buttonText = utils.ReplaceMacrosInTexts(title, message, buttonText, params)

	// Создаем запись уведомления для истории пользователя
	notification := &models.Notification{
		UserID:         user.ID,
		Type:           template.Type,
		Title:          title,
		Message:        message,
		ImageURL:       imageURL,
		ButtonText:     buttonText,
		ButtonURL:      buttonURL,
		ButtonCallback: buttonCallback,
		Delivered:      false, // Изначально отмечаем как не доставленное
		CreatedAt:      time.Now(),
	}

	// Сохраняем уведомление в историю
	if err := s.repo.CreateNotification(notification); err != nil {
		log.Printf("Error saving notification to history: %v", err)
		// Продолжаем, так как это не критическая ошибка
	}

	// Создаем карту данных для отправки через RabbitMQ
	notificationDataMap := map[string]interface{}{
		"user_id":         user.ID,
		"telegram_id":     user.TelegramID,
		"title":           title,
		"message":         message,
		"image_url":       imageURL,
		"button_text":     buttonText,
		"button_url":      buttonURL,
		"button_callback": buttonCallback,
		"notification_id": notification.ID,
	}

	// Подключаемся к RabbitMQ
	rmq, err := messaging.NewRabbitMQ(s.getRabbitMQURL(), "roulette_events", "notification_service")
	if err != nil {
		log.Printf("Error connecting to RabbitMQ: %v", err)
		return err
	}
	defer rmq.Close()

	// Устанавливаем таймаут для отправки сообщения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Публикуем карту данных напрямую через RabbitMQ
	if err := rmq.Publish(ctx, RoutingUserNotification, EventUserNotification, 0, notificationDataMap, messaging.PriorityLow); err != nil {
		log.Printf("Error publishing notification to RabbitMQ: %v", err)
		return err
	}

	// Если все прошло успешно, логируем информацию
	log.Printf("Notification queued for user %d (%d): %s - %s",
		user.ID, user.TelegramID, title, message)

	return nil
}

// getRabbitMQURL возвращает URL для подключения к RabbitMQ
func (s *ServiceImpl) getRabbitMQURL() string {
	// Получаем URL из настроек
	settings, err := s.GetSettings()
	if err != nil {
		return "amqp://guest:guest@rabbitmq:5672/" // Значение по умолчанию
	}

	if url, ok := settings["RABBITMQ_URL"]; ok && url != "" {
		return url
	}

	return "amqp://guest:guest@rabbitmq:5672/" // Значение по умолчанию
}
