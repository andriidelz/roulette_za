package admin

import (
	"fmt"
	"net/http"
	"roulette/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Список рейтингів
func (a *AdminPanel) ratingsList(c *gin.Context) {
	// Отримуємо список рейтингів за останні тижні
	// Тут потрібно додати метод для отримання списку рейтингів

	c.HTML(http.StatusOK, "ratings", gin.H{
		"title":     "Admin-panel - Ratings",
		"activeTab": "ratings",
	})
}

// Деталі рейтингу
func (a *AdminPanel) ratingDetails(c *gin.Context) {
	// Отримуємо рік і тиждень
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Невірний рік",
		})
		return
	}

	week, err := strconv.Atoi(c.Param("week"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Невірний тиждень",
		})
		return
	}

	// Отримуємо рейтинг
	ratings, err := a.repo.GetWeeklyRating(year, week, 100)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Отримуємо призовий фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   1000,
			TopCount: 100,
		}
	}

	c.HTML(http.StatusOK, "rating_details", gin.H{
		"title":     fmt.Sprintf("Admin-panel - Rating %d/%d", year, week),
		"ratings":   ratings,
		"year":      year,
		"week":      week,
		"prizeFund": prizeFund,
		"activeTab": "ratings",
	})
}

// Розподіл призів рейтингу
func (a *AdminPanel) distributeRatingPrizes(c *gin.Context) {
	// Отримуємо рік і тиждень
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wrong year"})
		return
	}

	week, err := strconv.Atoi(c.Param("week"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wrong week"})
		return
	}

	// Отримуємо суму призового фонду
	amount, err := strconv.ParseFloat(c.PostForm("amount"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірна сума призового фонду"})
		return
	}

	topCount, err := strconv.Atoi(c.PostForm("top_count"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірна кількість призових місць"})
		return
	}

	// Оновлюємо або створюємо призовий фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   amount,
			TopCount: topCount,
		}
		if err := a.repo.UpdatePrizeFund(prizeFund); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		prizeFund.Amount = amount
		prizeFund.TopCount = topCount
		if err := a.repo.UpdatePrizeFund(prizeFund); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Розподіляємо призи
	if err := a.service.DistributePrizes(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Список супер-рейтингів
func (a *AdminPanel) superRatingsList(c *gin.Context) {
	// Отримуємо список супер-рейтингів
	// Тут потрібно додати метод для отримання списку супер-рейтингів

	c.HTML(http.StatusOK, "super_ratings", gin.H{
		"title":     "Admin-panel - Super-ratings",
		"activeTab": "super_ratings",
	})
}

// Деталі супер-рейтингу
func (a *AdminPanel) superRatingDetails(c *gin.Context) {
	// Отримуємо період
	period := c.Param("period")

	// Отримуємо супер-рейтинг
	ratings, err := a.repo.GetSuperRating(period, 100)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "super_rating_details", gin.H{
		"title":     fmt.Sprintf("Admin-panel - Super-rating %s", period),
		"ratings":   ratings,
		"period":    period,
		"activeTab": "super_ratings",
	})
}
