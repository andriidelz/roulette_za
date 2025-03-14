package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Сторінка налаштувань
func (a *AdminPanel) settingsPage(c *gin.Context) {
	// Отримуємо налаштування
	settings, err := a.service.GetSettings()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "settings", gin.H{
		"title":     "Admin-panel - Налаштування",
		"settings":  settings,
		"activeTab": "settings",
	})
}

// Збереження налаштувань
func (a *AdminPanel) saveSettings(c *gin.Context) {
	// Отримуємо налаштування з форми
	// Зберігаємо налаштування
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			if err := a.service.UpdateSetting(key, values[0]); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
