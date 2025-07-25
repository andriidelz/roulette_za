package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"roulette/internal/config"
	"roulette/internal/data"
	"roulette/internal/logger"
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

// GetNotificationRecipients получает список получателей для задачи
func (s *ServiceImpl) GetNotificationRecipients(taskID uint, status string, page, limit int) ([]models.NotificationRecipient, int64, error) {
	return s.repo.GetNotificationRecipients(taskID, status, page, limit)
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
func (s *ServiceImpl) CreateNotificationTask(templateID uint, targetType string, targetParams models.NotificationTargetParams, scheduledAt *time.Time, macrosForUsers map[uint]map[string]interface{}) (*models.NotificationTask, error) {
	// Проверяем существование шаблона
	template, err := s.repo.GetNotificationTemplateByID(templateID)
	if err != nil {
		return nil, err
	}
	// Если scheduledAt не указано, устанавливаем текущее время
	if scheduledAt == nil {
		now := time.Now()
		scheduledAt = &now
	}

	// Для автоматических уведомлений проверяем, существует ли уже активная задача этого типа
	if template.Type == "automatic" && template.TriggerEvent != "" {
		// Для уведомлений, которые могут быть объединены, пытаемся найти существующую задачу
		// Проверяем только для определенных типов событий
		if template.TriggerEvent == "balance_updated" || template.TriggerEvent == "top_rating_entered" {
			// Ищем существующую задачу с таким же шаблоном и статусом "pending" или "processing"
			existingTasks, _, err := s.repo.GetNotificationTasks("pending,processing", 1, 10)
			if err == nil {
				for _, task := range existingTasks {
					// Проверяем, что это задача с тем же шаблоном
					if task.TemplateID == templateID {
						// Нашли существующую задачу того же типа

						// Проверяем, если это шаблон для конкретного пользователя
						if targetType == "custom" && len(targetParams.UserIDs) == 1 {
							// Получаем ID пользователя
							userID := targetParams.UserIDs[0]

							// Проверяем, что пользователь еще не включен в эту задачу
							existingRecipients, _, err := s.repo.GetNotificationRecipients(task.ID, "", 1, 1000)
							if err == nil {
								// Проверяем, есть ли уже этот пользователь среди получателей
								userExists := false
								for _, recipient := range existingRecipients {
									if recipient.UserID == userID {
										userExists = true
										break
									}
								}

								// Если пользователя еще нет в задаче, добавляем его
								if !userExists {
									// Получаем данные о пользователе
									user, err := s.repo.GetUserByID(userID)
									if err != nil {
										logger.Error.Printf("Error getting user %d: %v", userID, err)
										continue
									}

									// Определяем время отправки с учетом часового пояса
									recipientScheduledAt := *scheduledAt // Копируем значение времени

									// Если указаны параметры адаптации времени
									if targetParams.SendTimeStart != "" && targetParams.SendTimeEnd != "" && user.Country != "" {
										// Определяем часовой пояс пользователя на основе страны
										userTimeZone := data.GetTimezone(user.Country)
										recipientScheduledAt = adjustTimeToUserTimeZone(recipientScheduledAt, userTimeZone, targetParams.SendTimeStart, targetParams.SendTimeEnd)
									}

									// Получаем макросы для этого пользователя
									userMacros := macrosForUsers[userID]

									// Сериализуем макросы в JSON
									var macrosJSON []byte
									if userMacros != nil {
										macrosJSON, err = json.Marshal(userMacros)
										if err != nil {
											logger.Error.Printf("Error marshaling macros: %v", err)
											macrosJSON = []byte("{}")
										}
									} else {
										macrosJSON = []byte("{}")
									}

									// Создаем нового получателя с сохранением индивидуальных макросов
									recipient := models.NotificationRecipient{
										TaskID:      task.ID,
										UserID:      userID,
										Status:      "pending",
										ScheduledAt: &recipientScheduledAt,
										Macros:      string(macrosJSON),
										CreatedAt:   time.Now(),
										UpdatedAt:   time.Now(),
									}

									// Добавляем получателя в задачу
									if err := s.repo.CreateNotificationRecipient(&recipient); err != nil {
										logger.Error.Printf("Error creating recipient for task %d: %v", task.ID, err)
										continue
									}

									// Увеличиваем счетчик получателей в задаче
									task.TotalUsers++
									if err := s.repo.UpdateNotificationTask(&task); err != nil {
										logger.Error.Printf("Error updating task %d: %v", task.ID, err)
									}

									// Возвращаем существующую задачу
									return &task, nil
								}
							}
						}
					}
				}
			}
		}
	}

	// Создаем объект задачи с установленным временем
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

	logger.Info.Printf("Создана задача уведомлений %d, найдено %d получателей", task.ID, len(users))

	// Создаем получателей
	recipients := make([]models.NotificationRecipient, 0, len(users))
	for _, user := range users {
		// Определяем время отправки с учетом часового пояса
		recipientScheduledAt := *scheduledAt // Копируем значение времени

		// Если указаны параметры адаптации времени
		if targetParams.SendTimeStart != "" && targetParams.SendTimeEnd != "" && user.Country != "" {
			// Определяем часовой пояс пользователя на основе страны
			userTimeZone := data.GetTimezone(user.Country)
			recipientScheduledAt = adjustTimeToUserTimeZone(recipientScheduledAt, userTimeZone, targetParams.SendTimeStart, targetParams.SendTimeEnd)
		}

		// Получаем макросы для этого пользователя
		userMacros := macrosForUsers[user.ID]

		// Определяем макросы для получателя
		var macrosJSON string
		if userMacros != nil {
			macrosData, err := json.Marshal(userMacros)
			if err != nil {
				logger.Error.Printf("Error marshaling macros: %v", err)
				macrosJSON = "{}"
			} else {
				macrosJSON = string(macrosData)
			}
		} else {
			macrosJSON = "{}"
		}

		recipient := models.NotificationRecipient{
			TaskID:      task.ID,
			UserID:      user.ID,
			Status:      "pending",
			ScheduledAt: &recipientScheduledAt,
			Macros:      macrosJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		recipients = append(recipients, recipient)
	}

	// Сохраняем получателей пакетно
	if err := s.repo.CreateNotificationRecipientsBatch(recipients); err != nil {
		logger.Error.Printf("Ошибка при создании получателей для задачи %d: %v", task.ID, err)
		return nil, err
	}

	logger.Info.Printf("Создано %d получателей для задачи %d", len(recipients), task.ID)

	// Проверяем, созданы ли получатели
	_, total, err := s.repo.GetNotificationRecipients(task.ID, "", 1, 1)
	if err != nil {
		logger.Error.Printf("Ошибка при проверке получателей для задачи %d: %v", task.ID, err)
	} else if total == 0 {
		logger.Warning.Printf("ВНИМАНИЕ: Для задачи %d не создано ни одного получателя!", task.ID)
	} else {
		logger.Info.Printf("Задача %d: создано %d получателей", task.ID, total)
	}

	return task, nil
}

// SendNotifications отправляет уведомления для задачи
func (s *ServiceImpl) SendNotifications(taskID uint) error {
	// Получаем задачу
	task, err := s.repo.GetNotificationTaskByID(taskID)
	if err != nil {
		logger.Error.Printf("Error getting task %d: %v", taskID, err)
		return err
	}

	// Проверяем, что задача в статусе pending
	if task.Status != "pending" {
		return fmt.Errorf("task already in progress or completed")
	}

	// Получаем шаблон
	template, err := s.repo.GetNotificationTemplateByID(task.TemplateID)
	if err != nil {
		logger.Error.Printf("Error getting template %d for task %d: %v", task.TemplateID, taskID, err)
		return err
	}

	// Обновляем статус и время начала
	now := time.Now()
	task.Status = "processing"
	task.StartedAt = &now
	task.UpdatedAt = now
	if err := s.repo.UpdateNotificationTask(task); err != nil {
		logger.Error.Printf("Error updating task %d status: %v", taskID, err)
		return err
	}

	logger.Info.Printf("Starting notification task %d (template: %s)", taskID, template.Name)

	// Запускаем отправку в фоновом режиме
	go func() {
		page := 1
		pageSize := 100
		sentCount := 0
		deliveredCount := 0
		readCount := 0
		hasDelayedRecipients := false

		// Получаем общее количество получателей
		_, total, err := s.repo.GetNotificationRecipients(taskID, "", 1, 1)
		if err != nil {
			logger.Error.Printf("Error getting total recipients for task %d: %v", taskID, err)
		} else {
			logger.Info.Printf("Task %d: found %d recipients", taskID, total)
		}

		// Основной цикл обработки получателей
		for {
			recipients, _, err := s.repo.GetNotificationRecipients(taskID, "pending", page, pageSize)
			if err != nil {
				logger.Error.Printf("Error getting recipients for task %d: %v", taskID, err)
				break
			}

			if len(recipients) == 0 {
				break
			}

			for _, recipient := range recipients {
				// Проверяем время отправки
				if recipient.ScheduledAt != nil && recipient.ScheduledAt.After(time.Now()) {
					hasDelayedRecipients = true
					continue
				}

				// Отправляем уведомление пользователю
				err = s.sendNotificationToUser(recipient.UserID, template, task)
				nowSent := time.Now()
				recipient.SentAt = &nowSent
				recipient.UpdatedAt = nowSent

				if err != nil {
					recipient.Status = "failed"
					recipient.ErrorMessage = err.Error()
					logger.Error.Printf("Task %d: failed to send to user %d: %v", taskID, recipient.UserID, err)
				} else {
					recipient.Status = "sent"
					sentCount++
				}

				// Сохраняем обновления получателя
				if err := s.repo.UpdateNotificationRecipient(&recipient); err != nil {
					logger.Error.Printf("Task %d: error updating recipient %d: %v", taskID, recipient.ID, err)
				}
			}

			// Обновляем прогресс задачи
			if err := s.repo.UpdateTaskProgress(taskID, sentCount, deliveredCount, readCount); err != nil {
				logger.Error.Printf("Task %d: error updating progress: %v", taskID, err)
			}

			page++
		}

		// Проверяем, есть ли отложенные получатели
		if hasDelayedRecipients {
			// Если есть отложенные получатели, возвращаем задачу в статус "pending"
			pendingTask, err := s.repo.GetNotificationTaskByID(taskID)
			if err != nil {
				logger.Error.Printf("Task %d: error getting task for pending status: %v", taskID, err)
				return
			}

			pendingTask.Status = "pending"
			pendingTask.SentCount = sentCount
			pendingTask.DeliveredCount = deliveredCount
			pendingTask.ReadCount = readCount
			pendingTask.UpdatedAt = time.Now()

			if err := s.repo.UpdateNotificationTask(pendingTask); err != nil {
				logger.Error.Printf("Task %d: error returning task to pending: %v", taskID, err)
			} else {
				logger.Info.Printf("Task %d: returned to pending status (delayed recipients)", taskID)
			}
		} else {
			// Если отложенных получателей нет, помечаем задачу как завершенную
			completedTask, err := s.repo.GetNotificationTaskByID(taskID)
			if err != nil {
				logger.Error.Printf("Task %d: error getting task for completion: %v", taskID, err)
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
				logger.Error.Printf("Task %d: error completing task: %v", taskID, err)
			} else {
				logger.Info.Printf("Task %d completed: %d notifications sent", taskID, sentCount)
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

	// Удаляем через репозиторий
	if err := s.repo.DeleteNotificationTask(id); err != nil {
		return fmt.Errorf("ошибка при удалении задачи: %w", err)
	}

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

// sendNotificationToUser отправляет уведомление из задач notification_tasks
func (s *ServiceImpl) sendNotificationToUser(userID uint, template *models.NotificationTemplate, task *models.NotificationTask) error {
	// Получаем пользователя
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Получаем получателя для этого пользователя в рамках данной задачи
	recipients, _, err := s.repo.GetNotificationRecipients(task.ID, "", 1, 1000)
	if err != nil {
		logger.Error.Printf("Error getting recipients for task %d: %v", task.ID, err)
		// Продолжаем, используя пустые параметры
	}

	// Находим получателя для данного пользователя
	var recipient *models.NotificationRecipient
	for i, r := range recipients {
		if r.UserID == userID {
			recipient = &recipients[i]
			break
		}
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

	// Собираем параметры для замены макросов
	params := make(map[string]interface{})

	// Добавляем глобальные макросы
	for key, value := range s.GetGlobalMacros() {
		params[key] = value
	}

	// Проверяем, есть ли индивидуальные макросы у получателя
	if recipient != nil && recipient.Macros != "" {
		// Парсим макросы из JSON
		err := json.Unmarshal([]byte(recipient.Macros), &params)
		if err != nil {
			logger.Error.Printf("Error parsing recipient macros: %v", err)
		}
	}

	// В зависимости от типа уведомления добавляем дополнительные параметры
	switch template.TriggerEvent {
	case "balance_updated":
		// Если параметры не были переданы через макросы получателя,
		// попробуем найти транзакцию или использовать текущий баланс
		if _, exists := params["amount"]; !exists {
			// Тут можно добавить логику получения суммы из истории транзакций
			params["amount"] = "N/A" // Значение по умолчанию
			logger.Warning.Printf("Warning: 'amount' parameter missing for balance_updated notification for user %d", userID)
		}
		if _, exists := params["balance"]; !exists {
			params["balance"] = user.Balance
		}

	case "top_rating_entered":
		// Получаем текущий год и неделю
		year, week := time.Now().ISOWeek()

		// Получаем рейтинг пользователя, если информация отсутствует в макросах
		if _, exists := params["position"]; !exists {
			rating, err := s.repo.GetUserWeeklyRating(user.ID, year, week)
			if err == nil && rating != nil {
				// Добавляем позицию пользователя
				params["position"] = rating.Position
				params["points"] = rating.Points
			} else {
				logger.Warning.Printf("Warning: Could not get rating for user %d: %v", user.ID, err)
			}
		}
	}

	// Заменяем макросы в текстах с помощью общей функции
	title, message, buttonText = utils.ReplaceMacrosInTexts(title, message, buttonText, params)

	// Создаем запись уведомления для истории до отправки
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
		logger.Error.Printf("Error saving notification to history: %v", err)
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
		"notification_id": notification.ID, // Передаем ID для обновления статуса
	}

	// Подключаемся к RabbitMQ
	rmq, err := messaging.NewRabbitMQ(s.getRabbitMQURL(), "roulette_events", "notification_service")
	if err != nil {
		logger.Error.Printf("Error connecting to RabbitMQ: %v", err)
		return err
	}
	defer rmq.Close()

	// Устанавливаем таймаут для отправки сообщения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Публикуем карту данных напрямую через RabbitMQ
	if err := rmq.Publish(ctx, RoutingUserNotification, EventUserNotification, 0, notificationDataMap, messaging.PriorityLow); err != nil {
		logger.Error.Printf("Error publishing notification to RabbitMQ: %v", err)
		return err
	}

	// После успешной отправки обновляем статус уведомления
	if err := s.repo.MarkNotificationAsSent(notification.ID); err != nil {
		logger.Error.Printf("Error marking notification as sent: %v", err)
		// Не критическая ошибка, уведомление уже отправлено
	}

	// Если все прошло успешно, логируем информацию
	logger.Info.Printf("Notification sent for user %d (%d): %s - %s",
		user.ID, user.TelegramID, title, message)

	return nil
}

// getRabbitMQURL возвращает URL для подключения к RabbitMQ
func (s *ServiceImpl) getRabbitMQURL() string {
	// Получаем URL из конфигурации
	cfg := config.NewConfig()
	return cfg.RabbitMQURL
}

// CheckTopRatingEntries проверяет пользователей, вошедших в топ рейтинга и отправляет им уведомления
func (s *ServiceImpl) CheckTopRatingEntries() error {
	// Получаем текущий год и неделю
	year, week := time.Now().ISOWeek()

	// Получаем призовой фонд для текущей недели, чтобы узнать количество призовых мест
	fund, err := s.repo.GetPrizeFund(year, week)
	if err != nil {
		return fmt.Errorf("failed to get prize fund: %w", err)
	}

	// Получаем топ рейтинга
	topRating, err := s.repo.GetWeeklyRating(year, week, fund.TopCount)
	if err != nil {
		return fmt.Errorf("failed to get weekly rating: %w", err)
	}

	// Для каждого пользователя в топе проверяем, отправлялось ли ему уже уведомление
	for _, rating := range topRating {
		// Проверяем, было ли уже отправлено уведомление сегодня
		notificationSent, err := s.repo.CheckNotificationSent(rating.UserID, "top_rating_entered", time.Now().Format("2006-01-02"))
		if err != nil {
			logger.Error.Printf("Error checking notification status for user %d: %v", rating.UserID, err)
			continue
		}

		// Если уведомление уже было отправлено сегодня, пропускаем этого пользователя
		if notificationSent {
			continue
		}

		// Отправляем уведомление о вхождении в топ рейтинга
		if err := s.HandleTopRatingEntry(rating.UserID, rating.Position); err != nil {
			logger.Error.Printf("Error sending top rating notification to user %d: %v", rating.UserID, err)
			// Продолжаем обработку других пользователей
		} else {
			// Сохраняем информацию о том, что уведомление было отправлено
			if err := s.repo.SaveNotificationSent(rating.UserID, "top_rating_entered", time.Now().Format("2006-01-02")); err != nil {
				logger.Error.Printf("Error saving notification status for user %d: %v", rating.UserID, err)
			}
		}
	}

	return nil
}

// GetGlobalMacros получает глобальные макросы из настроек системы
func (s *ServiceImpl) GetGlobalMacros() map[string]interface{} {
	// Получаем настройки системы
	settings, err := s.GetSettings()
	if err != nil {
		logger.Error.Printf("Error getting settings for global macros: %v", err)
		return map[string]interface{}{}
	}

	// Создаем карту глобальных макросов
	globalMacros := make(map[string]interface{})

	// Добавляем лимит ставок за день
	if dailyBetsLimit, ok := settings["daily_bets_limit"]; ok {
		if limit, err := strconv.Atoi(dailyBetsLimit); err == nil {
			globalMacros["daily_bets_limit"] = limit
		}
	}

	// Добавляем лимит ставок для Zero
	if dailyBetsZeroLimit, ok := settings["daily_bets_zero_limit"]; ok {
		if limit, err := strconv.Atoi(dailyBetsZeroLimit); err == nil {
			globalMacros["daily_bets_zero_limit"] = limit
		}
	}

	// Добавляем минимальную сумму для вывода
	if minWithdrawal, ok := settings["minimum_withdrawal"]; ok {
		if amount, err := strconv.ParseFloat(minWithdrawal, 64); err == nil {
			globalMacros["minimum_withdrawal"] = amount
		}
	}

	// Добавляем сумму недельного призового фонда
	if prizeAmount, ok := settings["weekly_prize_amount"]; ok {
		if amount, err := strconv.ParseFloat(prizeAmount, 64); err == nil {
			globalMacros["prize_fund"] = amount
		}
	}

	// Добавляем количество призовых мест
	if prizeTop, ok := settings["weekly_prize_top"]; ok {
		if top, err := strconv.Atoi(prizeTop); err == nil {
			globalMacros["prize_top"] = top
		}
	}

	return globalMacros
}
