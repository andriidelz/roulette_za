package service

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"roulette/internal/models"
)

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
		return nil, errors.New("пользователи для отправки не найдены")
	}

	// Устанавливаем общее количество пользователей
	task.TotalUsers = len(users)

	// Сохраняем задачу
	if err := s.repo.CreateNotificationTask(task); err != nil {
		return nil, err
	}

	// Создаем получателей
	recipients := make([]models.NotificationRecipient, 0, len(users))
	for _, user := range users {
		// Определяем время отправки с учетом часового пояса
		var recipientScheduledAt *time.Time
		if scheduledAt != nil {
			// Определяем часовой пояс пользователя на основе страны
			userTimeZone := getUserTimeZone(user.Country)
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
		return nil, err
	}

	return task, nil
}

// SendNotifications отправляет уведомления для задачи
func (s *ServiceImpl) SendNotifications(taskID uint) error {
	// Получаем задачу
	task, err := s.repo.GetNotificationTaskByID(taskID)
	if err != nil {
		return err
	}

	// Проверяем, что задача в статусе pending
	if task.Status != "pending" {
		return errors.New("задача уже выполняется или завершена")
	}

	// Получаем шаблон
	template, err := s.repo.GetNotificationTemplateByID(task.TemplateID)
	if err != nil {
		return err
	}

	// Обновляем статус и время начала
	now := time.Now()
	task.Status = "processing"
	task.StartedAt = &now
	task.UpdatedAt = now
	if err := s.repo.UpdateNotificationTask(task); err != nil {
		return err
	}

	// Запускаем отправку в фоновом режиме
	go func() {
		// Получаем получателей для задачи
		page := 1
		pageSize := 100
		sentCount := 0
		deliveredCount := 0
		readCount := 0

		for {
			recipients, _, err := s.repo.GetNotificationRecipients(taskID, "pending", page, pageSize)
			if err != nil {
				log.Printf("Ошибка при получении получателей: %v", err)
				break
			}

			if len(recipients) == 0 {
				break
			}

			for _, recipient := range recipients {
				// Проверяем время отправки
				if recipient.ScheduledAt != nil && recipient.ScheduledAt.After(time.Now()) {
					// Пропускаем, если время отправки еще не наступило
					continue
				}

				// Отправляем уведомление пользователю
				err := s.sendNotificationToUser(recipient.UserID, template, task)
				nowSent := time.Now()
				recipient.SentAt = &nowSent
				recipient.UpdatedAt = nowSent

				if err != nil {
					// Обновляем статус и сообщение об ошибке
					recipient.Status = "failed"
					recipient.ErrorMessage = err.Error()
				} else {
					// Обновляем статус
					recipient.Status = "sent"
					sentCount++
				}

				// Сохраняем обновления
				if err := s.repo.UpdateNotificationRecipient(&recipient); err != nil {
					log.Printf("Ошибка при обновлении получателя %d: %v", recipient.ID, err)
				}
			}

			// Обновляем прогресс задачи
			if err := s.repo.UpdateTaskProgress(taskID, sentCount, deliveredCount, readCount); err != nil {
				log.Printf("Ошибка при обновлении прогресса задачи %d: %v", taskID, err)
			}

			page++
		}

		// Помечаем задачу как завершенную
		completedTask, err := s.repo.GetNotificationTaskByID(taskID)
		if err != nil {
			log.Printf("Ошибка при получении задачи %d: %v", taskID, err)
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
			log.Printf("Ошибка при обновлении задачи %d: %v", taskID, err)
		}
	}()

	return nil
}

// sendNotificationToUser отправляет уведомление пользователю
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

	// var buttonText string
	// if template.ButtonTextKey != "" {
	// 	buttonText, err = s.repo.GetLocalization(template.ButtonTextKey, language)
	// 	if err != nil {
	// 		// Если локализация не найдена, пробуем английскую
	// 		buttonText, err = s.repo.GetLocalization(template.ButtonTextKey, "en")
	// 		if err != nil {
	// 			buttonText = template.ButtonTextKey // Используем ключ как текст
	// 		}
	// 	}
	// }

	// Здесь должна быть реализация отправки через Telegram Bot API
	// Поскольку это выходит за рамки текущей задачи, просто логируем действие
	log.Printf("Отправка уведомления пользователю %d (%s): %s - %s",
		user.ID, user.TelegramID, title, message)

	// TODO: Реализовать отправку через Telegram Bot API
	// Используя токен s.telegramToken

	return nil
}

// DeleteNotificationTask удаляет задачу вместе с получателями
func (s *ServiceImpl) DeleteNotificationTask(id uint) error {
	// Начинаем транзакцию
	// Поскольку мы не имеем прямого доступа к транзакциям через интерфейс репозитория,
	// здесь нужно будет реализовать метод в репозитории для удаления задачи вместе с получателями
	return s.repo.DeleteNotificationTask(id)
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

// getUserTimeZone получает часовой пояс для страны
func getUserTimeZone(countryCode string) string {
	// Для упрощения используем мапку страна -> часовой пояс
	timeZones := map[string]string{
		"US": "America/New_York",
		"GB": "Europe/London",
		"DE": "Europe/Berlin",
		"FR": "Europe/Paris",
		"RU": "Europe/Moscow",
		"JP": "Asia/Tokyo",
		"UA": "Europe/Kiev",
		// Остальные страны...
	}

	if tz, ok := timeZones[countryCode]; ok {
		return tz
	}
	return "UTC" // По умолчанию UTC
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
