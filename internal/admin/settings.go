package admin

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// settingsPage страница настроек
func (a *AdminPanel) settingsPage(c *gin.Context) {
	// Получаем настройки с дополнительной информацией
	settingsWithInfo, err := a.service.GetSettingsWithInfo()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем список дней недели для выбора
	daysOfWeek := []gin.H{
		{"Value": "1", "Name": "Понедельник"},
		{"Value": "2", "Name": "Вторник"},
		{"Value": "3", "Name": "Среда"},
		{"Value": "4", "Name": "Четверг"},
		{"Value": "5", "Name": "Пятница"},
		{"Value": "6", "Name": "Суббота"},
		{"Value": "7", "Name": "Воскресенье"},
	}

	c.HTML(http.StatusOK, "settings", gin.H{
		"title":       "Admin-panel - Настройки",
		"settings":    settingsWithInfo,
		"daysOfWeek":  daysOfWeek,
		"activeTab":   "settings",
		"currentTime": time.Now().Format("15:04"),
	})
}

// saveSettings обработчик для сохранения настроек
func (a *AdminPanel) saveSettings(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		log.Printf("Ошибка разбора формы: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка разбора формы: " + err.Error()})
		return
	}

	formData := make(map[string]string)
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			formData[key] = values[0]
		}
	}

	if len(formData) == 0 {
		// Проверим сырые данные запроса
		bodyBytes, _ := c.GetRawData()
		log.Printf("Сырые данные запроса: %s", string(bodyBytes))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Форма не содержит данных"})
		return
	}

	// Сохраняем настройки
	if err := a.service.SaveSettings(formData); err != nil {
		log.Printf("Ошибка сохранения настроек: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
