package repository

import (
	"fmt"
	"strings"
	"time"

	"roulette/internal/data"
	"roulette/internal/models"
)

// CreateNotificationTemplate создает новый шаблон уведомления
func (r *PostgresRepository) CreateNotificationTemplate(template *models.NotificationTemplate) error {
	return r.db.Create(template).Error
}

// GetNotificationTemplates получает список шаблонов уведомлений с фильтрацией и пагинацией
func (r *PostgresRepository) GetNotificationTemplates(templateType string, page, pageSize int) ([]models.NotificationTemplate, int64, error) {
	var templates []models.NotificationTemplate
	var total int64

	query := r.db
	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}

	// Получаем общее количество записей
	err := query.Model(&models.NotificationTemplate{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Применяем пагинацию
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&templates).Error

	return templates, total, err
}

// GetNotificationTemplateByID получает шаблон уведомления по ID
func (r *PostgresRepository) GetNotificationTemplateByID(id uint) (*models.NotificationTemplate, error) {
	var template models.NotificationTemplate
	err := r.db.First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// UpdateNotificationTemplate обновляет шаблон уведомления
func (r *PostgresRepository) UpdateNotificationTemplate(template *models.NotificationTemplate) error {
	// Получаем текущее значение из базы данных
	var existingTemplate models.NotificationTemplate
	if err := r.db.First(&existingTemplate, template.ID).Error; err != nil {
		return err
	}

	// Проверяем, не пустая ли строка ImageURLs
	if template.ImageURLs == "" {
		// Получаем изображения из шаблона
		image := template.GetImage()

		// Если есть хотя бы одно изображение, сохраняем их
		if len(image) > 0 {
			if err := template.SetImage(image); err != nil {
				return err
			}
		} else {
			// Если изображений нет, используем пустую карту
			template.ImageURLs = "{}"
		}
	}

	// Сохраняем шаблон
	return r.db.Save(template).Error
}

// DeleteNotificationTemplate удаляет шаблон уведомления
func (r *PostgresRepository) DeleteNotificationTemplate(id uint) error {
	return r.db.Delete(&models.NotificationTemplate{}, id).Error
}

// CreateNotificationTask создает новую задачу на отправку уведомлений
func (r *PostgresRepository) CreateNotificationTask(task *models.NotificationTask) error {
	return r.db.Create(task).Error
}

// GetNotificationTasks получает список задач с фильтрацией и пагинацией
func (r *PostgresRepository) GetNotificationTasks(status string, page, pageSize int) ([]models.NotificationTask, int64, error) {
	var tasks []models.NotificationTask
	var total int64

	query := r.db.Preload("Template")

	// Применяем фильтр по статусу (поддерживаем несколько статусов через запятую)
	if status != "" {
		// Проверяем, содержит ли строка запятую (несколько статусов)
		if strings.Contains(status, ",") {
			statuses := strings.Split(status, ",")
			query = query.Where("status IN ?", statuses)
		} else {
			query = query.Where("status = ?", status)
		}
	}

	// Получаем общее количество записей
	err := query.Model(&models.NotificationTask{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Применяем пагинацию
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tasks).Error

	return tasks, total, err
}

// GetNotificationTaskByID получает задачу по ID с загрузкой связанных данных
func (r *PostgresRepository) GetNotificationTaskByID(id uint) (*models.NotificationTask, error) {
	var task models.NotificationTask
	err := r.db.Preload("Template").First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateNotificationTask обновляет задачу
func (r *PostgresRepository) UpdateNotificationTask(task *models.NotificationTask) error {
	return r.db.Save(task).Error
}

// DeleteNotificationTask удаляет задачу
func (r *PostgresRepository) DeleteNotificationTask(id uint) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Удаляем получателей
	if err := tx.Where("task_id = ?", id).Delete(&models.NotificationRecipient{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Удаляем задачу
	if err := tx.Delete(&models.NotificationTask{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// CreateNotificationRecipient создает получателя уведомления
func (r *PostgresRepository) CreateNotificationRecipient(recipient *models.NotificationRecipient) error {
	return r.db.Create(recipient).Error
}

// CreateNotificationRecipientsBatch создает несколько получателей уведомления за один запрос
func (r *PostgresRepository) CreateNotificationRecipientsBatch(recipients []models.NotificationRecipient) error {
	return r.db.CreateInBatches(recipients, 100).Error
}

// GetNotificationRecipients получает получателей уведомления для задачи
func (r *PostgresRepository) GetNotificationRecipients(taskID uint, status string, page, pageSize int) ([]models.NotificationRecipient, int64, error) {
	var recipients []models.NotificationRecipient
	var total int64

	query := r.db.Where("task_id = ?", taskID).Preload("User")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Получаем общее количество получателей
	err := query.Model(&models.NotificationRecipient{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Применяем пагинацию
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&recipients).Error

	return recipients, total, err
}

// UpdateNotificationRecipient обновляет получателя уведомления
func (r *PostgresRepository) UpdateNotificationRecipient(recipient *models.NotificationRecipient) error {
	return r.db.Save(recipient).Error
}

// GetPendingNotificationTasks получает список задач, которые должны быть выполнены сейчас
func (r *PostgresRepository) GetPendingNotificationTasks() ([]models.NotificationTask, error) {
	var tasks []models.NotificationTask
	now := time.Now()

	err := r.db.Preload("Template").
		Where("status = ? AND scheduled_at <= ?", "pending", now).
		Find(&tasks).Error

	return tasks, err
}

// MarkNotificationAsSent помечает уведомление как отправленное
func (r *PostgresRepository) MarkNotificationAsSent(id uint) error {
	return r.db.Model(&models.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"delivered": true,
			"read":      false,
		}).Error
}

// GetUsersForNotificationTask находит пользователей, соответствующих параметрам таргетинга
func (r *PostgresRepository) GetUsersForNotificationTask(task *models.NotificationTask) ([]models.User, error) {
	var users []models.User

	// Базовый запрос: пользователи не заблокированы
	query := r.db.Model(&models.User{}).Where("banned = ?", false)

	// Применяем фильтры в зависимости от типа таргетинга
	switch task.TargetType {
	case "all":
		// Все пользователи, не нужны дополнительные фильтры

	case "unregistered":
		// Таргетинг незареєстрованих
		query = query.Where("registered = ?", false)

	case "country":
		// Таргетинг по странам
		if len(task.TargetParams.Countries) > 0 {
			query = query.Where("country IN ?", task.TargetParams.Countries)
		}

	case "activity":
		// Таргетинг по активности
		if len(task.TargetParams.ActivityFilters) > 0 {
			// Временные границы для разных фильтров
			now := time.Now()
			threeDay := now.Add(-3 * 24 * time.Hour)
			twelveHour := now.Add(-12 * time.Hour)
			sevenDay := now.Add(-7 * 24 * time.Hour)
			fourteenDay := now.Add(-14 * 24 * time.Hour)

			// Множественные условия WHERE
			var conditions []string
			var values []interface{}

			for _, filter := range task.TargetParams.ActivityFilters {
				switch filter {
				case "inactive_3days":
					// Пользователи, активные от 12 часов до 3 дней назад
					conditions = append(conditions, "(last_activity_at >= ? AND last_activity_at <= ?)")
					values = append(values, threeDay, twelveHour)

				case "inactive_7days":
					// Пользователи, активные от 3 до 7 дней назад
					conditions = append(conditions, "(last_activity_at >= ? AND last_activity_at <= ?)")
					values = append(values, sevenDay, threeDay)

				case "inactive_14days":
					// Пользователи, активные от 7 до 14 дней назад
					conditions = append(conditions, "(last_activity_at >= ? AND last_activity_at <= ?)")
					values = append(values, fourteenDay, sevenDay)

				case "inactive_more_14days":
					// Пользователи, активные более 14 дней назад
					conditions = append(conditions, "last_activity_at <= ?")
					values = append(values, fourteenDay)
				}
			}

			if len(conditions) > 0 {
				// Объединяем условия через OR
				whereClause := "(" + strings.Join(conditions, " OR ") + ")"

				// Применяем условие к запросу
				query = query.Where(whereClause, values...)
			}
		}

	case "custom":
		// Таргетинг по конкретным пользователям
		if len(task.TargetParams.UserIDs) > 0 {
			query = query.Where("id IN ?", task.TargetParams.UserIDs)
		}
	}

	// Выполняем запрос
	err := query.Find(&users).Error

	return users, err
}

// GetNotificationTasksStats получает статистику задач уведомлений по периоду
func (r *PostgresRepository) GetNotificationTasksStats(period string) (*models.NotificationStatistics, error) {
	stats := &models.NotificationStatistics{
		Period: period,
	}

	// Определяем начальную дату периода
	var startDate time.Time
	now := time.Now()
	switch period {
	case "day":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		daysSinceMonday := (int(now.Weekday()) - 1 + 7) % 7
		startDate = time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		return nil, fmt.Errorf("неверный период: %s", period)
	}

	// Получаем общую статистику за период
	var tasks []models.NotificationTask
	err := r.db.Where("created_at >= ?", startDate).Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	// Считаем общую статистику
	for _, task := range tasks {
		stats.TotalSent += task.SentCount
		stats.TotalDelivered += task.DeliveredCount
		stats.TotalRead += task.ReadCount
	}

	// Получаем статистику по странам
	var countryStats []struct {
		Country   string
		Sent      int
		Delivered int
		Read      int
	}

	// SQL-запрос для статистики по странам
	query := `
		SELECT 
			u.country, 
			COUNT(CASE WHEN nr.status IN ('sent', 'delivered', 'read') THEN 1 ELSE NULL END) as sent,
			COUNT(CASE WHEN nr.status IN ('delivered', 'read') THEN 1 ELSE NULL END) as delivered,
			COUNT(CASE WHEN nr.status = 'read' THEN 1 ELSE NULL END) as read
		FROM notification_recipients nr
		JOIN users u ON nr.user_id = u.id
		JOIN notification_tasks nt ON nr.task_id = nt.id
		WHERE nt.created_at >= ?
		GROUP BY u.country
	`

	err = r.db.Raw(query, startDate).Scan(&countryStats).Error
	if err != nil {
		return nil, err
	}

	// Преобразуем результаты и добавляем названия стран
	stats.CountryStats = make([]models.CountryStats, len(countryStats))
	for i, cs := range countryStats {
		stats.CountryStats[i] = models.CountryStats{
			Country:     cs.Country,
			CountryName: data.GetCountryByCode(cs.Country).Name,
			Sent:        cs.Sent,
			Delivered:   cs.Delivered,
			Read:        cs.Read,
		}
	}

	return stats, nil
}

// GetCountriesWithUserCounts получает список стран с количеством пользователей
func (r *PostgresRepository) GetCountriesWithUserCounts() ([]models.CountryOption, error) {
	var results []struct {
		Country string
		Count   int
	}

	// Получаем количество пользователей по странам
	err := r.db.Model(&models.User{}).
		Select("country, COUNT(*) as count").
		Where("country <> ''").
		Group("country").
		Order("count DESC").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	// Если нет результатов, возвращаем пустой список
	if len(results) == 0 {
		return []models.CountryOption{}, nil
	}

	// Формируем список опций стран
	options := make([]models.CountryOption, len(results))

	// Получаем информацию о странах из пакета data
	countries := data.Countries
	countriesMap := make(map[string]data.Country)
	for _, country := range countries {
		countriesMap[country.Code] = country
	}

	for i, result := range results {
		// Базовая информация
		options[i] = models.CountryOption{
			Code:  result.Country,
			Count: result.Count,
		}

		// Дополняем информацией из справочника стран
		if countryInfo, ok := countriesMap[result.Country]; ok {
			options[i].Emoji = countryInfo.Emoji
			options[i].Name = countryInfo.Name
		} else {
			// Если страна не найдена в справочнике, используем код как название
			options[i].Name = result.Country
		}
	}

	return options, nil
}

// GetActivityFiltersWithUserCounts получает список фильтров активности с количеством пользователей
func (r *PostgresRepository) GetActivityFiltersWithUserCounts() ([]models.ActivityFilterOption, error) {
	now := time.Now()

	// Определяем временные границы для фильтров
	threeDay := now.Add(-3 * 24 * time.Hour)
	twelveHour := now.Add(-12 * time.Hour)
	sevenDay := now.Add(-7 * 24 * time.Hour)
	fourteenDay := now.Add(-14 * 24 * time.Hour)

	// Создаем структуру для результатов
	var filters []models.ActivityFilterOption
	var count int64

	// Не играл от 3 дней до 12 часов
	if err := r.db.Model(&models.User{}).
		Where("last_activity_at < ? AND last_activity_at > ?", threeDay, twelveHour).
		Count(&count).Error; err != nil {
		return nil, err
	}
	filters = append(filters, models.ActivityFilterOption{
		ID:    "inactive_3days",
		Label: "Не играл от 3 дней до 12 часов",
		Count: int(count),
	})

	// Не играл более 3 дней и менее 7 дней
	if err := r.db.Model(&models.User{}).
		Where("last_activity_at < ? AND last_activity_at > ?", sevenDay, threeDay).
		Count(&count).Error; err != nil {
		return nil, err
	}
	filters = append(filters, models.ActivityFilterOption{
		ID:    "inactive_7days",
		Label: "Не играл более 3 дней и менее 7 дней",
		Count: int(count),
	})

	// Не играл более 7 дней и менее 14 дней
	if err := r.db.Model(&models.User{}).
		Where("last_activity_at < ? AND last_activity_at > ?", fourteenDay, sevenDay).
		Count(&count).Error; err != nil {
		return nil, err
	}
	filters = append(filters, models.ActivityFilterOption{
		ID:    "inactive_14days",
		Label: "Не играл более 7 дней и менее 14 дней",
		Count: int(count),
	})

	// Не играл более 14 дней
	if err := r.db.Model(&models.User{}).
		Where("last_activity_at < ?", fourteenDay).
		Count(&count).Error; err != nil {
		return nil, err
	}
	filters = append(filters, models.ActivityFilterOption{
		ID:    "inactive_more_14days",
		Label: "Не играл более 14 дней",
		Count: int(count),
	})

	return filters, nil
}

// GetTemplateWithLocalizations получает шаблон уведомления с локализациями
func (r *PostgresRepository) GetTemplateWithLocalizations(templateID uint) (*models.NotificationTemplateWithLocalizations, error) {
	// Получаем базовый шаблон
	template, err := r.GetNotificationTemplateByID(templateID)
	if err != nil {
		return nil, err
	}

	result := &models.NotificationTemplateWithLocalizations{
		NotificationTemplate:    *template,
		TitleLocalizations:      make(map[string]string),
		MessageLocalizations:    make(map[string]string),
		ButtonTextLocalizations: make(map[string]string),
	}

	// Получаем локализации заголовка
	if template.TitleKey != "" {
		titleLocalizations, err := r.GetAllLocalizationsByKey(template.TitleKey)
		if err != nil {
			return nil, err
		}
		for _, loc := range titleLocalizations {
			result.TitleLocalizations[loc.Language] = loc.Value
		}
	}

	// Получаем локализации сообщения
	if template.MessageKey != "" {
		messageLocalizations, err := r.GetAllLocalizationsByKey(template.MessageKey)
		if err != nil {
			return nil, err
		}
		for _, loc := range messageLocalizations {
			result.MessageLocalizations[loc.Language] = loc.Value
		}
	}

	// Получаем локализации текста кнопки
	if template.ButtonTextKey != "" {
		buttonTextLocalizations, err := r.GetAllLocalizationsByKey(template.ButtonTextKey)
		if err != nil {
			return nil, err
		}
		for _, loc := range buttonTextLocalizations {
			result.ButtonTextLocalizations[loc.Language] = loc.Value
		}
	}

	return result, nil
}

// UpdateTaskProgress обновляет прогресс выполнения задачи
func (r *PostgresRepository) UpdateTaskProgress(taskID uint, sentCount, deliveredCount, readCount int) error {
	return r.db.Model(&models.NotificationTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"sent_count":      sentCount,
			"delivered_count": deliveredCount,
			"read_count":      readCount,
			"updated_at":      time.Now(),
		}).Error
}

// GetScheduledRecipients получает получателей, которые запланированы на отправку
func (r *PostgresRepository) GetScheduledRecipients(limit int) ([]models.NotificationRecipient, error) {
	var recipients []models.NotificationRecipient
	now := time.Now()

	err := r.db.Preload("User").
		Where("status = ? AND scheduled_at <= ?", "pending", now).
		Order("scheduled_at").
		Limit(limit).
		Find(&recipients).Error

	return recipients, err
}

// CheckNotificationSent проверяет, было ли отправлено уведомление данного типа пользователю в указанную дату
func (r *PostgresRepository) CheckNotificationSent(userID uint, notificationType string, date string) (bool, error) {
	var count int64
	startOfDay, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false, err
	}

	endOfDay := startOfDay.Add(24 * time.Hour)

	err = r.db.Model(&models.Notification{}).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
			userID, notificationType, startOfDay, endOfDay).
		Count(&count).Error

	return count > 0, err
}

// SaveNotificationSent сохраняет запись о том, что уведомление было отправлено
func (r *PostgresRepository) SaveNotificationSent(userID uint, notificationType string, date string) error {
	// Получаем пользователя
	_, err := r.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Создаем уведомление как запись о том, что оно было отправлено
	notification := &models.Notification{
		UserID:    userID,
		Type:      notificationType,
		Message:   "Notification tracking record for " + date,
		Title:     "Rating notification",
		Delivered: true,
		Read:      false,
		CreatedAt: time.Now(),
	}

	return r.CreateNotification(notification)
}
