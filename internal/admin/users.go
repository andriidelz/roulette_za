package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Список користувачів
func (a *AdminPanel) usersList(c *gin.Context) {
	// Отримуємо параметри пагінації
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20

	// Отримуємо користувачів
	users, totalUsers, err := a.repo.GetUsers(page, perPage)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Розраховуємо кількість сторінок
	totalPages := (int(totalUsers) + perPage - 1) / perPage

	// Розраховуємо номери попередньої та наступної сторінок
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}

	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}

	c.HTML(http.StatusOK, "users", gin.H{
		"title":      "Admin-panel - Користувачі",
		"users":      users,
		"page":       page,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
		"activeTab":  "users",
	})
}

// Деталі користувача
func (a *AdminPanel) userDetails(c *gin.Context) {
	// Отримуємо ID користувача
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Wrong user ID",
		})
		return
	}

	// Отримуємо інформацію про користувача
	// Цей метод треба додати в репозиторій
	user, err := a.repo.GetUserByID(uint(userID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Отримуємо статистику користувача
	stats, err := a.repo.GetUserStats(uint(userID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Отримуємо останні ставки користувача
	bets, err := a.repo.GetUserBets(uint(userID), 20)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "user_details", gin.H{
		"title":     fmt.Sprintf("Admin-panel - Користувач %s", user.Username),
		"user":      user,
		"stats":     stats,
		"bets":      bets,
		"activeTab": "users",
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
	user.Banned = true
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
	user.Banned = false
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
