package admin

import (
	"fmt"
	"net/http"
	"roulette/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

// Глобальная переменная для хранения времени запуска сервера
var serverStartTime time.Time

// Инициализируем время запуска при импорте пакета
func init() {
	serverStartTime = time.Now().UTC()
}

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

	// Получаем текущее время в различных форматах
	now := time.Now().UTC() // Текущее время в UTC
	currentDateTime := now.Format("2006-01-02 15:04:05")
	currentDate := now.Format("2006-01-02")
	currentTime := now.Format("15:04:05")

	// Форматируем время для отображения в JavaScript
	// JavaScript использует миллисекунды, поэтому умножаем на 1000
	jsTimestamp := now.Unix() * 1000

	// Получаем информацию о дне недели, месяце и т.д.
	weekday := now.Weekday().String()
	month := now.Month().String()
	day := now.Day()
	year := now.Year()

	// Расчет времени работы сервера
	uptime := now.Sub(serverStartTime)

	// Форматируем время работы в удобном для человека виде
	days := int(uptime.Hours() / 24)
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	var uptimeFormatted string
	if days > 0 {
		uptimeFormatted = fmt.Sprintf("%d дн. %d ч. %d мин. %d сек.", days, hours, minutes, seconds)
	} else if hours > 0 {
		uptimeFormatted = fmt.Sprintf("%d ч. %d мин. %d сек.", hours, minutes, seconds)
	} else if minutes > 0 {
		uptimeFormatted = fmt.Sprintf("%d мин. %d сек.", minutes, seconds)
	} else {
		uptimeFormatted = fmt.Sprintf("%d сек.", seconds)
	}

	// Также сохраняем время запуска в удобном формате
	serverStartFormatted := serverStartTime.Format("2006-01-02 15:04:05 MST")

	// Рік і тиждень для рейтингу
	_, week := now.ISOWeek()

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

	c.HTML(http.StatusOK, "dashboard", gin.H{
		"title":     "Admin-panel - Головна",
		"userCount": userCount,
		"year":      year,
		"week":      week,
		"prizeFund": prizeFund,
		"activeTab": "dashboard",

		// Информация о времени
		"currentDateTime": currentDateTime,
		"currentDate":     currentDate,
		"currentTime":     currentTime,
		"jsTimestamp":     jsTimestamp,
		"weekday":         weekday,
		"month":           month,
		"day":             day,
		"utcYear":         year,

		// Информация о времени работы сервера
		"uptime":          uptimeFormatted,
		"uptimeSeconds":   int(uptime.Seconds()),
		"serverStartTime": serverStartFormatted,
	})
}
