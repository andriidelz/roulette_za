package admin

import (
	"net/http"
	"roulette/internal/logger"
	"roulette/internal/utils"

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

// Статистика по источникам
func (a *AdminPanel) sourcesPage(c *gin.Context) {
	c.HTML(http.StatusOK, "sources_statistics", gin.H{
		"title":        "Admin-panel - Statistics by source",
		"activeTab":    "sources",
		"activeSubTab": "",
	})
}

// Статистика по источникам
func (a *AdminPanel) sourcesGetAll(c *gin.Context) {

	params := struct {
		Period   string `form:"period" json:"period"`
		DateFrom string `form:"dateFrom" json:"dateFrom"`
		DateTo   string `form:"dateTo" json:"dateTo"`
	}{}

	if err := c.Bind(&params); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	logger.Error.Println(params.Period, params.DateFrom, params.DateTo)

	_, err := utils.PeriodControl(&params.DateFrom, &params.DateTo, &params.Period)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	logger.Error.Println(params.DateFrom, params.DateTo)

	// Получаем статистику
	result, err := a.service.GetSource(params.DateFrom, params.DateTo)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result, "count": len(result)})
}
