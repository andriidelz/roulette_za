package admin

import (
	"net/http"
	"roulette/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Список запитів на виведення коштів
func (a *AdminPanel) withdrawalsList(c *gin.Context) {
	// Отримуємо список запитів на виведення коштів
	withdrawals, err := a.repo.GetPendingWithdrawals()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "withdrawals", gin.H{
		"title":       "Admin-panel - Withdrawals",
		"withdrawals": withdrawals,
		"activeTab":   "withdrawals",
	})
}

// Підтвердження виведення коштів
func (a *AdminPanel) withdrawalApprove(c *gin.Context) {
	// Отримуємо ID запиту
	withdrawalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID запиту"})
		return
	}

	// Получаем платежный сервис
	paymentService := c.MustGet("paymentService").(*service.PaymentService)

	// Обрабатываем запрос на вывод через платежный сервис
	if err := paymentService.ApproveWithdrawal(uint(withdrawalID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Відхилення виведення коштів
func (a *AdminPanel) withdrawalReject(c *gin.Context) {
	// Отримуємо ID запиту
	withdrawalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID запиту"})
		return
	}

	// Отримуємо запит
	withdrawal, err := a.repo.GetWithdrawalByID(uint(withdrawalID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Повертаємо кошти на баланс користувача
	user, err := a.repo.GetUserByID(withdrawal.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.Balance += withdrawal.Amount
	if err := a.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Відхиляємо запит
	if err := a.repo.UpdateWithdrawalStatus(uint(withdrawalID), "rejected"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
