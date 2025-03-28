package admin

import (
	"net/http"
	"roulette/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// getUserJSON обработчик для получения информации о пользователе в формате JSON
func (a *AdminPanel) getUserJSON(c *gin.Context) {
	// Получаем ID пользователя
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Получаем информацию о пользователе
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем статистику пользователя
	totalBets, err := a.repo.GetUserTotalBets(user.ID)
	if err != nil {
		totalBets = 0
	}

	wonBets, err := a.repo.GetUserWonBets(user.ID)
	if err != nil {
		wonBets = 0
	}

	totalPoints, err := a.repo.GetUserTotalPoints(user.ID)
	if err != nil {
		totalPoints = 0
	}

	// Вычисляем эффективность
	var efficiency float64
	if totalBets > 0 {
		efficiency = float64(wonBets) / float64(totalBets) * 100
	}

	// Получаем текущую позицию пользователя в рейтинге
	year, week := a.service.GetCurrentYearWeek()
	rating, err := a.repo.GetUserWeeklyRating(user.ID, year, week)

	// Получаем детальную статистику для детализированного отображения
	detailedStats, err := a.service.GetDetailedUserStats(user.TelegramID, "all")
	if err != nil {
		detailedStats = make(map[string]int)
	}

	// Получаем последние ставки пользователя
	bets, err := a.repo.GetUserBets(user.ID, 10)
	if err != nil {
		bets = []models.Bet{}
	}

	// Получаем выводы пользователя
	withdrawals, err := a.repo.GetUserWithdrawals(user.ID, 5)
	if err != nil {
		withdrawals = []models.Withdrawal{}
	}

	// Возвращаем результат в формате JSON
	c.JSON(http.StatusOK, gin.H{
		"user": user,
		"stats": gin.H{
			"TotalBets":   totalBets,
			"WonBets":     wonBets,
			"TotalPoints": totalPoints,
			"Efficiency":  efficiency,
		},
		"rating":        rating,
		"detailedStats": detailedStats,
		"bets":          bets,
		"withdrawals":   withdrawals,
	})
}

// getUserDetailedStats получает детальную статистику пользователя в формате JSON
func (a *AdminPanel) getUserDetailedStats(c *gin.Context) {
	// Получаем ID пользователя
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Получаем пользователя
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем различные периоды статистики
	dailyStats, err := a.service.GetDetailedUserStats(user.TelegramID, "day")
	if err != nil {
		dailyStats = make(map[string]int)
	}

	weeklyStats, err := a.service.GetDetailedUserStats(user.TelegramID, "week")
	if err != nil {
		weeklyStats = make(map[string]int)
	}

	monthlyStats, err := a.service.GetDetailedUserStats(user.TelegramID, "month")
	if err != nil {
		monthlyStats = make(map[string]int)
	}

	allTimeStats, err := a.service.GetDetailedUserStats(user.TelegramID, "all")
	if err != nil {
		allTimeStats = make(map[string]int)
	}

	// Возвращаем результат в формате JSON
	c.JSON(http.StatusOK, gin.H{
		"daily":   dailyStats,
		"weekly":  weeklyStats,
		"monthly": monthlyStats,
		"allTime": allTimeStats,
	})
}
