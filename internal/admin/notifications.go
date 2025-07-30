package admin

import (
	"net/http"
	"roulette/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// setupNotificationsRoutes настраивает маршруты для раздела уведомлений
func (a *AdminPanel) setupNotificationsRoutes() {
	admin := a.router.Group("/admin")
	admin.Use(a.ipFilterMiddleware(), a.authRequired())

	// Страницы уведомлений
	admin.GET("/notifications", a.notificationsPage)
	admin.GET("/notifications/manual", a.notificationsManualPage)
	admin.GET("/notifications/automatic", a.notificationsAutomaticPage)
	admin.GET("/notifications/history", a.notificationsHistoryPage)

	// API для работы с шаблонами уведомлений
	admin.GET("/api/notification-templates", a.getNotificationTemplates)
	admin.GET("/api/notification-templates/:id", a.getNotificationTemplate)
	admin.POST("/api/notification-templates", a.createNotificationTemplate)
	admin.PUT("/api/notification-templates/:id", a.updateNotificationTemplate)
	admin.DELETE("/api/notification-templates/:id", a.deleteNotificationTemplate)

	// API для работы с задачами уведомлений
	admin.GET("/api/notification-tasks", a.getNotificationTasks)
	admin.GET("/api/notification-tasks/:id", a.getNotificationTask)
	admin.POST("/api/notification-tasks", a.createNotificationTask)
	admin.POST("/api/notification-tasks/:id/cancel", a.cancelNotificationTask)
	admin.GET("/api/notification-recipients", a.getNotificationRecipients)

	// API для получения списка стран
	admin.GET("/api/countries-with-users", a.getCountriesWithUsers)
}

// notificationsPage - основная страница уведомлений
func (a *AdminPanel) notificationsPage(c *gin.Context) {
	c.Redirect(http.StatusFound, "/admin/notifications/automatic")
}

// notificationsManualPage - страница ручных уведомлений
func (a *AdminPanel) notificationsManualPage(c *gin.Context) {
	// Получаем список шаблонов
	templates, _, err := a.service.GetNotificationTemplates("", 1, 100)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем список активных задач
	tasks, _, err := a.service.GetNotificationTasks("", 1, 100)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "notifications", gin.H{
		"title":        "Ручные уведомления",
		"activeTab":    "notifications",
		"activeSubTab": "manual",
		"templates":    templates,
		"tasks":        tasks,
	})
}

// notificationsAutomaticPage - страница автоматических уведомлений
func (a *AdminPanel) notificationsAutomaticPage(c *gin.Context) {
	// Получаем список автоматических шаблонов
	templates, _, err := a.service.GetNotificationTemplates("automatic", 1, 100)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "notifications", gin.H{
		"title":        "Автоматические уведомления",
		"activeTab":    "notifications",
		"activeSubTab": "automatic",
		"templates":    templates,
	})
}

// notificationsHistoryPage - страница истории отправок
func (a *AdminPanel) notificationsHistoryPage(c *gin.Context) {
	// Получаем параметры пагинации
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20

	// Получаем историю завершенных задач
	tasks, total, err := a.service.GetNotificationTasks("completed,failed,canceled", page, perPage)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Вычисляем общее количество страниц
	totalPages := (int(total) + perPage - 1) / perPage

	c.HTML(http.StatusOK, "notifications", gin.H{
		"title":        "История отправок",
		"activeTab":    "notifications",
		"activeSubTab": "history",
		"tasks":        tasks,
		"pagination": gin.H{
			"current":    page,
			"total":      totalPages,
			"totalItems": total,
			"perPage":    perPage,
		},
	})
}

// getNotificationTemplates - API метод для получения списка шаблонов уведомлений
func (a *AdminPanel) getNotificationTemplates(c *gin.Context) {
	templateType := c.DefaultQuery("type", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	templates, total, err := a.service.GetNotificationTemplates(templateType, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"meta": gin.H{
			"total":   total,
			"page":    page,
			"perPage": perPage,
		},
	})
}

// getNotificationTemplate - API метод для получения шаблона уведомления по ID
func (a *AdminPanel) getNotificationTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	// Получаем шаблон с локализациями
	template, err := a.service.GetTemplateWithLocalizations(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Формируем ответ для фронтенда
	response := gin.H{
		"id":            template.ID,
		"name":          template.Name,
		"type":          template.Type,
		"trigger_event": template.TriggerEvent,
		"image":         template.GetImage(),

		// Добавляем ключи локализации
		"title_key":       template.TitleKey,
		"message_key":     template.MessageKey,
		"button_text_key": template.ButtonTextKey,

		// Существующие поля для совместимости
		"title":   template.TitleLocalizations,
		"message": template.MessageLocalizations,
		"button": gin.H{
			"text":     template.ButtonTextLocalizations,
			"url":      template.ButtonURL,
			"callback": template.ButtonCallback,
			"text_key": template.ButtonTextKey, // Добавляем ключ для текста кнопки
		},
		"active": template.Active,
	}

	c.JSON(http.StatusOK, response)
}

// createNotificationTemplate - API метод для создания шаблона уведомления
func (a *AdminPanel) createNotificationTemplate(c *gin.Context) {
	var request struct {
		Name         string            `json:"name"`
		Type         string            `json:"type"` // manual, automatic
		TriggerEvent string            `json:"trigger_event,omitempty"`
		TitleKey     string            `json:"title_key"`       // Ключ для заголовка, введенный пользователем
		MessageKey   string            `json:"message_key"`     // Ключ для сообщения, введенный пользователем
		Title        map[string]string `json:"title"`           // Локализации заголовка
		Message      map[string]string `json:"message"`         // Локализации сообщения
		Image        map[string]string `json:"image,omitempty"` // Локализованные изображения
		Button       struct {
			TextKey  string            `json:"text_key,omitempty"` // Ключ для кнопки, введенный пользователем
			Text     map[string]string `json:"text,omitempty"`     // Локализации текста кнопки
			URL      string            `json:"url,omitempty"`
			Callback string            `json:"callback,omitempty"`
		} `json:"button,omitempty"`
		Active bool `json:"active"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем введенные ключи локализации
	if request.TitleKey == "" || request.MessageKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ключи локализации для заголовка и сообщения обязательны",
		})
		return
	}

	// Сохраняем локализации для заголовка
	for lang, text := range request.Title {
		if err := a.clearAndSaveLocalization(request.TitleKey, lang, text, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to save title localization: " + err.Error(),
			})
			return
		}
	}

	// Сохраняем локализации для сообщения
	for lang, text := range request.Message {
		if err := a.clearAndSaveLocalization(request.MessageKey, lang, text, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to save message localization: " + err.Error(),
			})
			return
		}
	}

	// Сохраняем локализации для кнопки, если они есть
	buttonTextKey := request.Button.TextKey
	if buttonTextKey != "" && len(request.Button.Text) > 0 {
		for lang, text := range request.Button.Text {
			if err := a.clearAndSaveLocalization(buttonTextKey, lang, text, ""); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to save button text localization: " + err.Error(),
				})
				return
			}
		}
	}

	// Создаем шаблон уведомления
	template := &models.NotificationTemplate{
		Name:           request.Name,
		Type:           request.Type,
		TriggerEvent:   request.TriggerEvent,
		TitleKey:       request.TitleKey,   // Используем введенный пользователем ключ
		MessageKey:     request.MessageKey, // Используем введенный пользователем ключ
		ButtonTextKey:  buttonTextKey,      // Используем введенный пользователем ключ
		ButtonURL:      request.Button.URL,
		ButtonCallback: request.Button.Callback,
		Active:         request.Active,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Устанавливаем локализованные изображения
	if len(request.Image) > 0 {
		if err := template.SetImage(request.Image); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to set image: " + err.Error(),
			})
			return
		}
	} else {
		// Установка пустого объекта для избежания ошибки с JSON
		template.ImageURLs = "{}"
	}

	if err := a.service.CreateNotificationTemplate(template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create notification template: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      template.ID,
		"message": "Template created successfully",
	})
}

// updateNotificationTemplate - API метод для обновления шаблона уведомления
func (a *AdminPanel) updateNotificationTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	// Получаем существующий шаблон
	template, err := a.service.GetNotificationTemplateByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var request struct {
		Name         string            `json:"name"`
		Type         string            `json:"type"`
		TriggerEvent string            `json:"trigger_event,omitempty"`
		TitleKey     string            `json:"title_key"`
		MessageKey   string            `json:"message_key"`
		Title        map[string]string `json:"title"`
		Message      map[string]string `json:"message"`
		Image        map[string]string `json:"image,omitempty"`
		Button       struct {
			TextKey  string            `json:"text_key,omitempty"`
			Text     map[string]string `json:"text,omitempty"`
			URL      string            `json:"url,omitempty"`
			Callback string            `json:"callback,omitempty"`
		} `json:"button,omitempty"`
		Active bool `json:"active"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что ключи локализации указаны
	if request.TitleKey == "" || request.MessageKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ключи локализации для заголовка и сообщения обязательны",
		})
		return
	}

	newTitleKey := request.TitleKey
	newMessageKey := request.MessageKey
	newButtonTextKey := request.Button.TextKey

	// Обновляем локализации заголовка
	for lang, text := range request.Title {
		if err := a.clearAndSaveLocalization(newTitleKey, lang, text, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update title localization: " + err.Error(),
			})
			return
		}
	}

	// Обновляем локализации сообщения
	for lang, text := range request.Message {
		if err := a.clearAndSaveLocalization(newMessageKey, lang, text, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update message localization: " + err.Error(),
			})
			return
		}
	}

	// Обновляем локализации кнопки
	if newButtonTextKey != "" && len(request.Button.Text) > 0 {
		for lang, text := range request.Button.Text {
			if err := a.clearAndSaveLocalization(newButtonTextKey, lang, text, ""); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to update button text localization: " + err.Error(),
				})
				return
			}
		}
	}

	// Обновляем основные поля шаблона
	template.Name = request.Name
	template.Type = request.Type
	template.TriggerEvent = request.TriggerEvent
	template.TitleKey = newTitleKey
	template.MessageKey = newMessageKey
	template.ButtonTextKey = newButtonTextKey
	template.ButtonURL = request.Button.URL
	template.ButtonCallback = request.Button.Callback
	template.Active = request.Active
	template.UpdatedAt = time.Now()

	// Устанавливаем локализованные изображения
	if len(request.Image) > 0 {
		if err := template.SetImage(request.Image); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to set image: " + err.Error(),
			})
			return
		}
	} else {
		// Установка пустого объекта для избежания ошибки с JSON
		template.ImageURLs = "{}"
	}

	// Сохраняем обновленный шаблон
	if err := a.service.UpdateNotificationTemplate(template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update notification template: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Template updated successfully",
	})
}

// deleteNotificationTemplate - API метод для удаления шаблона уведомления
func (a *AdminPanel) deleteNotificationTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	if err := a.service.DeleteNotificationTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Template deleted successfully",
	})
}

// getNotificationTasks - API метод для получения списка задач уведомлений
func (a *AdminPanel) getNotificationTasks(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	tasks, total, err := a.service.GetNotificationTasks(status, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"meta": gin.H{
			"total":      total,
			"page":       page,
			"perPage":    perPage,
			"totalPages": (total + int64(perPage) - 1) / int64(perPage),
		},
	})
}

// getNotificationTask - API метод для получения задачи уведомления по ID
func (a *AdminPanel) getNotificationTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	task, err := a.service.GetEnhancedNotificationTask(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// getNotificationRecipients - API метод для получения получателей уведомления
func (a *AdminPanel) getNotificationRecipients(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указан ID задачи"})
		return
	}

	taskIDUint, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID задачи"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	// Получаем получателей для задачи
	recipients, total, err := a.service.GetNotificationRecipients(uint(taskIDUint), "", 1, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recipients": recipients,
		"total":      total,
	})
}

// createNotificationTask - API метод для создания задачи уведомления
func (a *AdminPanel) createNotificationTask(c *gin.Context) {
	var request struct {
		TemplateID    uint                            `json:"templateId"`
		TargetType    string                          `json:"targetType"` // all, country, activity, custom
		TargetParams  models.NotificationTargetParams `json:"targetParams"`
		ScheduledSend bool                            `json:"scheduledSend"`
		ScheduledAt   time.Time                       `json:"scheduledAt,omitempty"`
		Macros        map[string]interface{}          `json:"macros,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Определяем scheduledAt
	var scheduledAt *time.Time
	if request.ScheduledSend && !request.ScheduledAt.IsZero() {
		scheduledAt = &request.ScheduledAt
	}

	// Создаем пустую карту макросов для пользователей
	macrosForUsers := make(map[uint]map[string]interface{})

	// Если есть макросы в параметрах таргетинга, добавляем их для каждого пользователя
	if len(request.TargetParams.UserIDs) > 0 {
		for _, userID := range request.TargetParams.UserIDs {
			// Если в запросе были указаны макросы, копируем их для каждого пользователя
			if request.Macros != nil {
				macrosForUsers[userID] = request.Macros
			}
		}
	}

	// Создаем задачу через сервис
	task, err := a.service.CreateNotificationTask(
		request.TemplateID,
		request.TargetType,
		request.TargetParams,
		scheduledAt,
		macrosForUsers,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         task.ID,
		"message":    "Notification task created successfully",
		"totalUsers": task.TotalUsers,
	})
}

// cancelNotificationTask - API метод для отмены задачи уведомления
func (a *AdminPanel) cancelNotificationTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	if err := a.service.CancelNotificationTask(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task canceled successfully",
	})
}

// getCountriesWithUsers - API метод для получения списка стран с количеством пользователей
func (a *AdminPanel) getCountriesWithUsers(c *gin.Context) {
	countries, err := a.service.GetCountriesWithUserCounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, countries)
}
