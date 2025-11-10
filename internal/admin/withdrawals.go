package admin

import (
	"net/http"
	"roulette/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (a *AdminPanel) withdrawalsStat(c *gin.Context) {

	params := struct {
		Period   string `form:"period" json:"period"`
		DateFrom string `form:"dateFrom" json:"dateFrom"`
		DateTo   string `form:"dateTo" json:"dateTo"`
	}{}

	if err := c.Bind(&params); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := utils.PeriodControl(&params.DateFrom, &params.DateTo, &params.Period)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	data, err := a.repo.GetWithdrawalsStat(params.DateFrom, params.DateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// Список запитів на виведення коштів
func (a *AdminPanel) withdrawalsList(c *gin.Context) {
	// Получаем список запросов на вывод со статусом "pending"
	pendingWithdrawals, err := a.repo.GetPendingWithdrawals()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем историю выводов (все кроме pending)
	historyWithdrawals, err := a.repo.GetWithdrawalsHistory(50) // Ограничиваем 50 последними записями
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "withdrawals", gin.H{
		"title":              "Admin-panel - Withdrawals",
		"withdrawals":        pendingWithdrawals,
		"historyWithdrawals": historyWithdrawals, // Добавляем в контекст историю выводов
		"activeTab":          "withdrawals",
	})
}

// Подтверждение вывода средств
func (a *AdminPanel) withdrawalApprove(c *gin.Context) {
	// Получаем ID запроса
	withdrawalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID запроса"})
		return
	}

	if err := a.paymentService.ApproveWithdrawal(uint(withdrawalID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Отклонение вывода средств
func (a *AdminPanel) withdrawalReject(c *gin.Context) {
	// Получаем ID запроса
	withdrawalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID запроса"})
		return
	}

	// Получаем запрос
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
