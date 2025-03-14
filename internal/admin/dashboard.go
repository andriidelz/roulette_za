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

	// Рік і тиждень для рейтингу
	year, week := time.Now().ISOWeek()

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

	// Поточна дата
	currentDateTime := time.Now().Format("2006-01-02")

	c.HTML(http.StatusOK, "dashboard", gin.H{
		"title":           "Admin-panel - Головна",
		"userCount":       userCount,
		"year":            year,
		"week":            week,
		"prizeFund":       prizeFund,
		"activeTab":       "dashboard",
		"currentDateTime": currentDateTime,
	})
}
