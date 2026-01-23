package admin

import (
	"fmt"
	"net/http"
	"roulette/internal/data"
	"roulette/internal/logger"
	"roulette/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// usersList обработчик страницы списка пользователей с расширенными возможностями
func (a *AdminPanel) usersList(c *gin.Context) {
	// Получаем параметры пагинации
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20

	// Получаем параметр поиска
	query := c.Query("query")

	// Получаем пользователей
	var users []models.User
	var totalUsers int64
	var err error

	if query != "" {
		// Поиск пользователей с фильтрацией по запросу
		users, totalUsers, err = a.repo.SearchUsers(query, page, perPage)
	} else {
		// Получаем всех пользователей с пагинацией
		users, totalUsers, err = a.repo.GetUsers(page, perPage)
	}

	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Расширяем данные пользователей дополнительной информацией
	var enhancedUsers []gin.H
	for _, user := range users {
		// Получаем количество ставок за сегодня для каждого пользователя
		dailyBets, err := a.repo.GetUserDailyBets(user.ID)
		if err != nil {
			dailyBets = 0
		}

		// Добавляем информацию о стране
		var countryEmoji string
		for _, country := range data.Countries {
			if country.Code == user.Country {
				countryEmoji = country.Emoji
				break
			}
		}

		enhancedUsers = append(enhancedUsers, gin.H{
			"ID":           user.ID,
			"TelegramID":   user.TelegramID,
			"Username":     user.Username,
			"FirstName":    user.FirstName,
			"LastName":     user.LastName,
			"LanguageCode": user.LanguageCode,
			"Country":      user.Country,
			"CountryEmoji": countryEmoji,
			"Balance":      user.Balance,
			"Status":       user.Status,
			"Registered":   user.Registered,
			"CreatedAt":    user.CreatedAt,
			"UpdatedAt":    user.UpdatedAt,
			"DailyBets":    dailyBets,
		})
	}

	// Получаем список стран для фильтрации
	var countries []gin.H
	for _, country := range data.Countries {
		countries = append(countries, gin.H{
			"Code":  country.Code,
			"Name":  country.Name,
			"Emoji": country.Emoji,
		})
	}

	// Расчет параметров пагинации
	totalPages := (int(totalUsers) + perPage - 1) / perPage
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}

	c.HTML(http.StatusOK, "users", gin.H{
		"title":      "Admin-panel - Пользователи",
		"users":      enhancedUsers,
		"query":      query,
		"page":       page,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"totalUsers": totalUsers,
		"activeTab":  "users",
		"countries":  countries,
	})
}

// userDetails обработчик страницы с подробной информацией о пользователе
func (a *AdminPanel) userDetails(c *gin.Context) {
	// Получаем ID пользователя
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Wrong user ID",
		})
		return
	}

	// Получаем информацию о пользователе
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем статистику пользователя напрямую из таблицы bets
	totalBets, err := a.repo.GetUserTotalBets(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user total bets: %v", err)
		totalBets = 0
	}

	wonBets, err := a.repo.GetUserWonBets(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user won bets: %v", err)
		wonBets = 0
	}

	totalPoints, err := a.repo.GetUserTotalPoints(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user total points: %v", err)
		totalPoints = 0
	}

	// Получаем количество дневных ставок
	dailyBets, err := a.repo.GetUserDailyBets(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user daily bets: %v", err)
		dailyBets = 0
	}

	// Получаем количество ставок и очков за неделю
	weeklyBets, weeklyPoints, err := a.repo.GetUserWeeklyStats(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user weekly stats: %v", err)
		weeklyBets, weeklyPoints = 0, 0
	}

	// Получаем количество ставок и очков за месяц
	monthlyBets, monthlyPoints, err := a.repo.GetUserMonthlyStats(user.ID)
	if err != nil {
		logger.Error.Printf("Error getting user monthly stats: %v", err)
		monthlyBets, monthlyPoints = 0, 0
	}

	// Вычисляем эффективность
	var efficiency float64
	if totalBets > 0 {
		efficiency = float64(wonBets) / float64(totalBets) * 100
	}

	// Получаем текущую позицию пользователя в рейтинге
	year, week := a.service.GetCurrentYearWeek()
	rating, err := a.repo.GetUserWeeklyRating(user.ID, year, week)
	if err != nil {
		logger.Error.Printf("Error getting user rating: %v", err)
		rating = nil
	}

	// Создаем статистику для отображения в шаблоне
	stats := gin.H{
		"TotalBets":     totalBets,
		"WonBets":       wonBets,
		"TotalPoints":   totalPoints,
		"Efficiency":    efficiency,
		"DailyBets":     dailyBets,
		"WeeklyBets":    weeklyBets,
		"WeeklyPoints":  weeklyPoints,
		"MonthlyBets":   monthlyBets,
		"MonthlyPoints": monthlyPoints,
	}

	// Получаем список всех доступных стран для выбора
	var countries []gin.H
	for _, country := range data.Countries {
		countries = append(countries, gin.H{
			"Code":  country.Code,
			"Name":  country.Name,
			"Emoji": country.Emoji,
		})
	}

	// Находим флаг страны пользователя
	userCountryEmoji := ""
	userCountryName := ""
	for _, country := range data.Countries {
		if country.Code == user.Country {
			userCountryEmoji = country.Emoji
			userCountryName = country.Name
			break
		}
	}

	// Получаем список языков для выбора
	languages := []gin.H{
		{"Code": "en", "Name": "English"},
		{"Code": "ru", "Name": "Русский"},
		{"Code": "uk", "Name": "Українська"},
	}

	// Получаем последние ставки пользователя
	bets, err := a.repo.GetUserBets(user.ID, 20)
	if err != nil {
		logger.Error.Printf("Error getting user bets: %v", err)
		bets = []models.Bet{}
	}

	// Получаем историю выводов средств пользователя
	withdrawals, err := a.repo.GetUserWithdrawals(user.ID, 10)
	if err != nil {
		logger.Error.Printf("Error getting user withdrawals: %v", err)
		withdrawals = []models.Withdrawal{}
	}

	banLog, err := a.repo.GetActiveBanLog(user.ID)
	if err != nil {
		logger.Error.Printf("Failed to get ban log: %v", err)
		banLog = models.UserBanLog{}
	}

	uptime := time.Until(banLog.UntilTo)
	uptimeFormatted := formatDate(uptime)

	c.HTML(http.StatusOK, "user_details", gin.H{
		"title":            fmt.Sprintf("Admin-panel - Користувач %s", user.Username),
		"user":             user,
		"banLog":           banLog,
		"uptimeFormatted":  uptimeFormatted,
		"botName":          a.settings.BotName,
		"stats":            stats,
		"rating":           rating,
		"bets":             bets,
		"withdrawals":      withdrawals,
		"activeTab":        "users",
		"countries":        countries,
		"userCountryEmoji": userCountryEmoji,
		"userCountryName":  userCountryName,
		"languages":        languages,
	})
}

// updateUserProfile обновляет профиль пользователя
func (a *AdminPanel) updateUserProfile(c *gin.Context) {
	// Получаем ID пользователя
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Получаем пользователя из базы данных
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем данные формы
	username := c.PostForm("username")
	firstName := c.PostForm("firstName")
	lastName := c.PostForm("lastName")
	languageCode := c.PostForm("languageCode")
	country := c.PostForm("country")
	walletAddress := c.PostForm("walletAddress")

	// Обновляем данные пользователя
	user.Username = username
	user.FirstName = firstName
	user.LastName = lastName
	user.LanguageCode = languageCode
	user.Country = country
	user.WalletAddress = walletAddress

	// Сохраняем обновленные данные
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// updateUserBalance обновляет баланс пользователя
func (a *AdminPanel) updateUserBalance(c *gin.Context) {
	// Получаем ID пользователя
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Получаем пользователя из базы данных
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Получаем данные формы (сумму и операцию)
	amountStr := c.PostForm("amount")
	operation := c.PostForm("operation")

	// Преобразуем строку суммы в число
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат суммы"})
		return
	}

	// В зависимости от операции изменяем баланс
	if operation == "add" {
		user.Balance += amount
	} else if operation == "subtract" {
		// Проверяем, не будет ли баланс отрицательным
		if user.Balance < amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недостаточно средств на балансе"})
			return
		}
		user.Balance -= amount
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неизвестная операция"})
		return
	}

	// Сохраняем обновленные данные
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// TODO: Создаем запись в журнале операций (если нужно)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"balance": user.Balance,
	})
}

// Блокування користувача
func (a *AdminPanel) userBan(c *gin.Context) {
	// Отримуємо ID користувача
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID користувача"})
		return
	}

	// Отримуємо інформацію про користувача
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Блокуємо користувача
	user.Status = "BANNED"
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// create new record
	banLog := &models.UserBanLog{
		UserID:     user.ID,
		TypeStatus: "BANNED",
		Reason:     "manual",
		Active:     true,
		UntilTo:    time.Now().AddDate(1,0,0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	// Save to database
	if err := a.repo.CreateBanLog(banLog); err != nil {
		logger.Error.Printf("Failed to create ban log: %v", err)
	}

	// Удаляем рейтинг пользователя и пересчитываем позиции всех в рейтинге
	a.repo.DeleteRating(user.ID)

	// Обновляем все рейтинги
	year, week := time.Now().ISOWeek()
	a.repo.RefreshWeeklyRatingsPosition(year, week)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Розблокування користувача
func (a *AdminPanel) userUnban(c *gin.Context) {
	// Отримуємо ID користувача
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID користувача"})
		return
	}

	// Отримуємо інформацію про користувача
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Розблокуємо користувача
	user.Status = "ACTIVE"
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	banLog, err := a.repo.GetActiveBanLog(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	banLog.Active = false
	// Save to database
	if err := a.repo.UpdateBanLog(&banLog); err != nil {
		logger.Error.Printf("Failed to create ban log: %v", err)
	}

	// Создаем рейтинг пользователя и пересчитываем позиции всех в рейтинге
	a.repo.UpdateWeeklyRatingForUser(user.ID)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Установка реферальной ссылки
func (a *AdminPanel) userRef(c *gin.Context) {
	// Отримуємо ID користувача
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID користувача"})
		return
	}

	// Отримуємо інформацію про користувача
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if user.RefKey != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Используется ключ: " + user.RefKey})
		return
	}

	user.RefKey = fmt.Sprint(user.ID)
	// Сохраняем источник
	if err := a.repo.SetSourceKey(user.RefKey, "Пользователь "+fmt.Sprint(user.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
