package admin

import (
	"fmt"
	"net/http"
	"roulette/internal/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Список рейтингів
func (a *AdminPanel) ratingsList(c *gin.Context) {
	// Получаем список последних недельных рейтингов
	currentYear, currentWeek := time.Now().ISOWeek()

	// Получаем данные о 10 последних недельных рейтингах
	weeklyRatings, err := a.getWeeklyRatingsHistory(currentYear, currentWeek, 10)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Ошибка",
			"error": err.Error(),
		})
		return
	}

	// Получаем данные о текущем призовом фонде
	// Принудительно обновляем данные из базы, игнорируя возможный кэш
	prizeFund, err := a.repo.GetPrizeFund(currentYear, currentWeek)
	if err != nil {
		prizeFund = &models.PrizeFund{
			Year:     currentYear,
			Week:     currentWeek,
			Amount:   1000,
			TopCount: 100,
		}
	}

	// Получаем информацию о супер-рейтингах
	superRatings, err := a.getSuperRatingsHistory()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Ошибка",
			"error": err.Error(),
		})
		return
	}

	// Отправляем данные в шаблон
	c.HTML(http.StatusOK, "ratings", gin.H{
		"title":         "Рейтинги",
		"activeTab":     "ratings",
		"currentYear":   currentYear,
		"currentWeek":   currentWeek,
		"weeklyRatings": weeklyRatings,
		"superRatings":  superRatings,
		"prizeFund":     prizeFund,
	})
}

// getWeeklyRatingsHistory получает историю недельных рейтингов
func (a *AdminPanel) getWeeklyRatingsHistory(currentYear, currentWeek int, limit int) ([]gin.H, error) {
	var result []gin.H

	// Получаем фактические данные о недельных рейтингах из базы данных
	prizeFunds, err := a.repo.GetRecentPrizeFunds(limit)
	if err != nil {
		return nil, err
	}

	// Если в базе нет данных о фондах, создаем только одну запись для текущей недели
	if len(prizeFunds) == 0 {
		// Определяем статус текущей недели
		status := "active"
		statusText := "Активен"
		badgeClass := "bg-primary"

		// Создаем запись для текущей недели
		weekRating := gin.H{
			"year":       currentYear,
			"week":       currentWeek,
			"amount":     1000.0, // Значение по умолчанию
			"topCount":   100,    // Значение по умолчанию
			"processed":  false,
			"status":     status,
			"statusText": statusText,
			"badgeClass": badgeClass,
		}

		result = append(result, weekRating)
		return result, nil
	}

	// Преобразуем полученные данные в нужный формат для шаблона
	for _, fund := range prizeFunds {
		// Определяем статус рейтинга
		status := "pending"
		statusText := "В ожидании"
		badgeClass := "bg-warning"

		if fund.Processed {
			status = "completed"
			statusText = "Завершен"
			badgeClass = "bg-success"
		} else {
			// Проверяем, закончилась ли уже эта неделя
			now := time.Now()
			currentYear, currentWeek := now.ISOWeek()

			if fund.Year < currentYear || (fund.Year == currentYear && fund.Week < currentWeek) {
				status = "ended"
				statusText = "Незавершен"
				badgeClass = "bg-danger"
			} else if fund.Year == currentYear && fund.Week == currentWeek {
				status = "active"
				statusText = "Активен"
				badgeClass = "bg-primary"
			}
		}

		// Формируем данные для шаблона
		weekRating := gin.H{
			"year":       fund.Year,
			"week":       fund.Week,
			"amount":     fund.Amount,
			"topCount":   fund.TopCount,
			"processed":  fund.Processed,
			"status":     status,
			"statusText": statusText,
			"badgeClass": badgeClass,
		}

		result = append(result, weekRating)
	}

	return result, nil
}

// getSuperRatingsHistory получает историю супер-рейтингов
func (a *AdminPanel) getSuperRatingsHistory() ([]gin.H, error) {
	var result []gin.H

	// Получаем текущий год и квартал
	now := time.Now()
	currentYear := now.Year()
	currentQuarter := (int(now.Month())-1)/3 + 1

	// Создаем список из 4 последних кварталов
	var periods []string
	year, quarter := currentYear, currentQuarter

	for i := 0; i < 4; i++ {
		period := fmt.Sprintf("%d-Q%d", year, quarter)
		periods = append(periods, period)

		// Переходим к предыдущему кварталу
		quarter--
		if quarter < 1 {
			year--
			quarter = 4
		}
	}

	// Для каждого периода получаем информацию о супер-рейтинге
	for _, period := range periods {
		// Получаем супер-рейтинг
		ratings, err := a.repo.GetSuperRating(period, 1)
		if err != nil || len(ratings) == 0 {
			// Если рейтинг не найден, создаем пустой
			result = append(result, gin.H{
				"period":     period,
				"amount":     5000.0, // Значение по умолчанию
				"status":     "pending",
				"statusText": "В ожидании",
				"badgeClass": "bg-warning",
			})
			continue
		}

		// Определяем статус рейтинга
		status := "active"
		statusText := "Активен"
		badgeClass := "bg-primary"

		// Если текущий период уже завершен, помечаем как завершенный
		// Это упрощенная логика, в реальности нужно проверять по датам
		parts := strings.Split(period, "-Q")
		periodYear, _ := strconv.Atoi(parts[0])
		periodQuarter, _ := strconv.Atoi(parts[1])

		if periodYear < currentYear || (periodYear == currentYear && periodQuarter < currentQuarter) {
			status = "completed"
			statusText = "Завершен"
			badgeClass = "bg-success"
		}

		// Формируем данные для шаблона
		result = append(result, gin.H{
			"period":     period,
			"amount":     5000.0, // Здесь должна быть реальная сумма призового фонда
			"status":     status,
			"statusText": statusText,
			"badgeClass": badgeClass,
		})
	}

	return result, nil
}

// Деталі рейтингу - по незавершеним відображаємо теперішню ситуацію з приблизним розподілом нагород
func (a *AdminPanel) ratingDetails(c *gin.Context) {
	// Получаем параметры из URL
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Ошибка",
			"error": "Неверный год",
		})
		return
	}

	week, err := strconv.Atoi(c.Param("week"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Ошибка",
			"error": "Неверная неделя",
		})
		return
	}

	// Получаем призовой фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   1000,
			TopCount: 100,
		}
	}

	// Статус распределения призов
	prizeStatus, err := a.service.GetPrizeDistributionStatus(year, week)
	if err != nil {
		prizeStatus = "PENDING"
	}

	// Отримуємо рейтинг
	ratings, err := a.repo.GetWeeklyRating(year, week, prizeFund.TopCount)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Ошибка",
			"error": err.Error(),
		})
		return
	}

	// Якщо фонд не розподілений то виводимо приблизні призи які отримають учасники
	// по тим балам що вони мають на данний момент
	// Логіка аналогічна розподілу призів в (s *ServiceImpl) DistributePrizes
	if prizeStatus == "PENDING" {
		// Рахуємо загальну кількість балів у топі
		totalPoints := 0
		for _, rating := range ratings {
			totalPoints += rating.Points
		}

		if totalPoints > 0 {
			// Розподіляємо потенційний! призовий фонд пропорційно балам
			for i := range ratings {
				prize := (float64(ratings[i].Points) / float64(totalPoints)) * prizeFund.Amount
				ratings[i].Prize = prize
			}
		}
	}

	// Парсим дату начала и конца недели
	startDate, endDate := getWeekDates(year, week)

	currentYear, currentWeek := time.Now().ISOWeek()

	c.HTML(http.StatusOK, "rating_details", gin.H{
		"title":       fmt.Sprintf("Рейтинг %d/%d", year, week),
		"ratings":     ratings,
		"year":        year,
		"week":        week,
		"prizeFund":   prizeFund,
		"prizeStatus": prizeStatus,
		"startDate":   startDate.Format("02.01.2006"),
		"endDate":     endDate.Format("02.01.2006"),
		"activeTab":   "ratings",
		"currentYear": currentYear,
		"currentWeek": currentWeek,
	})
}

// getWeekDates возвращает дату начала и конца недели
func getWeekDates(year, week int) (time.Time, time.Time) {
	// Первый день года
	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)

	// Корректируем на первый день недели (понедельник)
	dayOffset := int(jan1.Weekday())
	if dayOffset == 0 { // Воскресенье (в Go - 0)
		dayOffset = 7
	}
	dayOffset--

	// Первый понедельник года
	firstMonday := jan1.AddDate(0, 0, -dayOffset)

	// Если 1 января - это уже неделя 1, то первый понедельник года
	// относится к неделе 1, иначе - к неделе 52/53 предыдущего года
	_, firstWeek := jan1.ISOWeek()
	if firstWeek > 1 {
		// Первая неделя начинается с первого понедельника после 1 января
		firstMonday = firstMonday.AddDate(0, 0, 7)
	}

	// Начало запрошенной недели
	startDate := firstMonday.AddDate(0, 0, (week-1)*7)

	// Конец запрошенной недели
	endDate := startDate.AddDate(0, 0, 6)

	return startDate, endDate
}

// Розподіл призів рейтингу
func (a *AdminPanel) distributeRatingPrizes(c *gin.Context) {
	// Получаем год и неделю из URL
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный год"})
		return
	}

	week, err := strconv.Atoi(c.Param("week"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверная неделя"})
		return
	}

	// Проверяем, не является ли это текущей неделей
	currentYear, currentWeek := time.Now().ISOWeek()
	if year == currentYear && week == currentWeek {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нельзя распределить призы за текущую неделю"})
		return
	}

	// Получаем параметры из формы
	amount, err := strconv.ParseFloat(c.PostForm("amount"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверная сумма призового фонда"})
		return
	}

	topCount, err := strconv.Atoi(c.PostForm("top_count"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверное количество призовых мест"})
		return
	}

	// Получаем или создаем призовой фонд
	prizeFund, err := a.repo.GetPrizeFund(year, week)
	if err != nil {
		// Если фонд не найден, создаем новый
		prizeFund = &models.PrizeFund{
			Year:     year,
			Week:     week,
			Amount:   amount,
			TopCount: topCount,
		}
	} else {
		// Обновляем существующий фонд
		prizeFund.Amount = amount
		prizeFund.TopCount = topCount
	}

	// Сохраняем призовой фонд
	if err := a.repo.UpdatePrizeFund(prizeFund); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Проверяем статус и распределяем призы, если необходимо
	action := c.PostForm("action")
	if action == "distribute" {
		// Проверяем статус распределения призов
		status, err := a.service.GetPrizeDistributionStatus(year, week)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Распределяем призы в зависимости от статуса
		switch status {
		case "DISTRIBUTED":
			c.JSON(http.StatusBadRequest, gin.H{"error": "Призы уже распределены"})
			return
		case "PARTIALLY_DISTRIBUTED":
			// Сбрасываем частично распределенные призы
			if err := a.repo.FixPartiallyDistributedPrizes(year, week, "reset"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// И продолжаем распределение
			fallthrough
		case "PENDING", "NOT_CONFIGURED":
			// Распределяем призы с указанием года и недели из URL
			if err := a.service.DistributePrizes(year, week); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	// Возвращаем успешный результат
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *AdminPanel) cancelRatingPrizes(c *gin.Context) {
	// Получаем год и неделю из URL
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	week, err := strconv.Atoi(c.Param("week"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid week"})
		return
	}

	// Отменяем распределение призов
	if err := a.service.CancelPrizeDistribution(year, week); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем успешный результат
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Prize distribution cancelled"})
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
