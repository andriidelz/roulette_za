package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Статистика
func (a *AdminPanel) statistics(c *gin.Context) {
	// Отримуємо статистику за день, тиждень, місяць
	// Тут потрібно додати методи для отримання статистики

	c.HTML(http.StatusOK, "statistics", gin.H{
		"title":     "Admin-panel - Statistics",
		"activeTab": "stats",
	})
}
