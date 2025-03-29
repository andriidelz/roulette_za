package admin

import (
	"net/http"
	"roulette/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

func (a *AdminPanel) dashboard(c *gin.Context) {
	// Отримуємо загальну статистику
	userCount, err := a.repo.GetUserCount()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем количество новых пользователей за сегодня
	newUsersToday := 0 // Заглушка, нужно реализовать в репозитории

	// Расчет роста пользователей (в процентах)
	var userGrowthRate float64 = 0.0 // или любое другое начальное значение

	// Рік і тиждень для рейтингу
	year, week := time.Now().ISOWeek()

	// Получаем информацию о количестве активных игроков
	activePlayers := 0 // Заглушка, нужно реализовать в репозитории

	// Расчет дней до окончания текущей недели
	today := time.Now()
	daysUntilSunday := 7 - int(today.Weekday())
	if daysUntilSunday == 7 {
		daysUntilSunday = 0
	}

	// Отримуємо призовий фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		// Якщо запис не знайдено, створюємо значення за замовчуванням
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   1000,
			TopCount: 100,
		}

		// Можемо спробувати зберегти його в базу даних
		_ = a.repo.UpdatePrizeFund(prizeFund) // Ігноруємо помилку, якщо вона виникне
	}

	// Данные о выплатах
	pendingWithdrawalsCount := 0    // Заглушка, нужно реализовать в репозитории
	pendingWithdrawalsAmount := 0.0 // Заглушка, нужно реализовать в репозитории

	// Данные о ставках
	redBetsPercentage := 33.0   // Заглушка, нужно реализовать в репозитории
	blackBetsPercentage := 33.0 // Заглушка, нужно реализовать в репозитории
	zeroBetsPercentage := 34.0  // Заглушка, нужно реализовать в репозитории

	// Топ игроков (заглушка)
	topPlayers := []gin.H{
		{"ID": 1, "Username": "player1", "FirstName": "John", "LastName": "Doe", "Points": 1200, "BetsCount": 45},
		{"ID": 2, "Username": "player2", "FirstName": "Jane", "LastName": "Smith", "Points": 980, "BetsCount": 38},
		{"ID": 3, "Username": "", "FirstName": "Mike", "LastName": "Johnson", "Points": 870, "BetsCount": 32},
		{"ID": 4, "Username": "winner4", "FirstName": "Alex", "LastName": "Brown", "Points": 760, "BetsCount": 28},
		{"ID": 5, "Username": "lucky5", "FirstName": "Sam", "LastName": "Wilson", "Points": 650, "BetsCount": 25},
	}

	// Системная информация
	launchDate := "2025-01-15"
	uptime := "42 дней 8 часов"
	scheduledTasksCount := 3

	// Последние действия (заглушка)
	recentActions := []gin.H{
		{"Timestamp": "15:30:45", "Type": "registration", "Action": "Регистрация", "UserID": 123, "Username": "newuser", "Details": "Новый пользователь"},
		{"Timestamp": "15:25:12", "Type": "bet", "Action": "Ставка", "UserID": 115, "Username": "player1", "Details": "100 очков на красное"},
		{"Timestamp": "15:20:33", "Type": "withdrawal", "Action": "Вывод", "UserID": 87, "Username": "winner", "Details": "250$ запрошено"},
		{"Timestamp": "14:55:18", "Type": "login", "Action": "Вход", "UserID": 92, "Username": "regular", "Details": "Успешный вход"},
		{"Timestamp": "14:48:02", "Type": "bet", "Action": "Ставка", "UserID": 103, "Username": "gambler", "Details": "50 очков на черное"},
	}

	// Поточна дата
	currentDateTime := time.Now().Format("2006-01-02 15:04:05")

	c.HTML(http.StatusOK, "dashboard", gin.H{
		"title":                    "Admin-panel - Головна",
		"userCount":                userCount,
		"newUsersToday":            newUsersToday,
		"userGrowthRate":           userGrowthRate,
		"year":                     year,
		"week":                     week,
		"activePlayers":            activePlayers,
		"daysLeft":                 daysUntilSunday,
		"prizeFund":                prizeFund,
		"prizePlacesCount":         prizeFund.TopCount,
		"pendingWithdrawalsCount":  pendingWithdrawalsCount,
		"pendingWithdrawalsAmount": pendingWithdrawalsAmount,
		"redBetsPercentage":        redBetsPercentage,
		"blackBetsPercentage":      blackBetsPercentage,
		"zeroBetsPercentage":       zeroBetsPercentage,
		"topPlayers":               topPlayers,
		"launchDate":               launchDate,
		"uptime":                   uptime,
		"scheduledTasksCount":      scheduledTasksCount,
		"recentActions":            recentActions,
		"activeTab":                "dashboard",
		"currentDateTime":          currentDateTime,
	})
}
