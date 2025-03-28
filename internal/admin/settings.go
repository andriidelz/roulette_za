package admin

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// settingsPage страница настроек
func (a *AdminPanel) settingsPage(c *gin.Context) {
	// Получаем настройки с дополнительной информацией
	settingsWithInfo, err := a.service.GetSettingsWithInfo()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем список дней недели для выбора
	daysOfWeek := []gin.H{
		{"Value": "1", "Name": "Понедельник"},
		{"Value": "2", "Name": "Вторник"},
		{"Value": "3", "Name": "Среда"},
		{"Value": "4", "Name": "Четверг"},
		{"Value": "5", "Name": "Пятница"},
		{"Value": "6", "Name": "Суббота"},
		{"Value": "7", "Name": "Воскресенье"},
	}

	c.HTML(http.StatusOK, "settings", gin.H{
		"title":       "Admin-panel - Настройки",
		"settings":    settingsWithInfo,
		"daysOfWeek":  daysOfWeek,
		"activeTab":   "settings",
		"currentTime": time.Now().Format("15:04"),
	})
}

// saveSettings обработчик для сохранения настроек
func (a *AdminPanel) saveSettings(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		log.Printf("Ошибка разбора формы: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка разбора формы: " + err.Error()})
		return
	}

	formData := make(map[string]string)
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			formData[key] = values[0]
		}
	}

	if len(formData) == 0 {
		// Проверим сырые данные запроса
		bodyBytes, _ := c.GetRawData()
		log.Printf("Сырые данные запроса: %s", string(bodyBytes))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Форма не содержит данных"})
		return
	}

	// Проверяем, изменились ли настройки призового фонда
	var prizeAmountChanged, topCountChanged bool
	var newPrizeAmount float64
	var newTopCount int

	if prizeAmountStr, ok := formData["weekly_prize_amount"]; ok {
		if amount, err := strconv.ParseFloat(prizeAmountStr, 64); err == nil {
			prizeAmountChanged = true
			newPrizeAmount = amount
		}
	}

	if topCountStr, ok := formData["weekly_prize_top"]; ok {
		if count, err := strconv.Atoi(topCountStr); err == nil {
			topCountChanged = true
			newTopCount = count
		}
	}

	// Сохраняем настройки
	if err := a.service.SaveSettings(formData); err != nil {
		log.Printf("Ошибка сохранения настроек: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Если изменились настройки призового фонда, обновляем текущий фонд напрямую
	if prizeAmountChanged && topCountChanged {
		// Напрямую обновляем текущий призовой фонд
		if err := a.service.UpdateCurrentPrizeFund(newPrizeAmount, newTopCount); err != nil {
			log.Printf("Ошибка прямого обновления призового фонда: %v", err)
			// Отправляем специальное сообщение об ошибке
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"warning": fmt.Sprintf("Настройки сохранены, но при обновлении призового фонда возникла ошибка: %v", err),
			})
			return
		}

		// Логируем успешное обновление
		log.Printf("Настройки и призовой фонд успешно обновлены: сумма = %.2f, топ = %d",
			newPrizeAmount, newTopCount)
	} else if prizeAmountChanged {
		// Получаем текущее количество призовых мест
		year, week := time.Now().ISOWeek()
		prizeFund, err := a.repo.GetPrizeFund(year, week)
		if err == nil && !prizeFund.Processed {
			if err := a.service.UpdateCurrentPrizeFund(newPrizeAmount, prizeFund.TopCount); err != nil {
				log.Printf("Ошибка обновления суммы призового фонда: %v", err)
			}
		}
	} else if topCountChanged {
		// Получаем текущую сумму призового фонда
		year, week := time.Now().ISOWeek()
		prizeFund, err := a.repo.GetPrizeFund(year, week)
		if err == nil && !prizeFund.Processed {
			if err := a.service.UpdateCurrentPrizeFund(prizeFund.Amount, newTopCount); err != nil {
				log.Printf("Ошибка обновления количества призовых мест: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
