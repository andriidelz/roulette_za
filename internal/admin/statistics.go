package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Статистика
func (a *AdminPanel) statistics(c *gin.Context) {
	// Получаем общую статистику
	totalStats, err := a.service.GetTotalStats()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем статистику успешных угадываний
	successRateStats, err := a.service.GetSuccessRateStats()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем топ игроков по проценту успешных ставок
	topSuccessRatePlayers, err := a.service.GetTopPlayersBySuccessRate(10)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем топ игроков по количеству ставок
	topBetsPlayers, err := a.service.GetTopPlayersByAttempts(10)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "statistics", gin.H{
		"title":                 "Admin-panel - Statistics",
		"activeTab":             "stats",
		"totalStats":            totalStats,
		"successRateStats":      successRateStats,
		"topSuccessRatePlayers": topSuccessRatePlayers,
		"topBetsPlayers":        topBetsPlayers,
	})
}
