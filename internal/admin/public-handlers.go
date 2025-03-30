package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"roulette/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// setupPublicRoutes настраивает публичные маршруты
func (a *AdminPanel) setupPublicRoutes() {
	// Публичные страницы (не требуют авторизации)
	public := a.router.Group("/")
	{
		public.GET("/", a.publicHomePage)
		public.GET("/verify", a.publicVerifyPage)
		public.GET("/faq", a.publicFaqPage)
		public.GET("/example", a.hashVerificationExample)

		// API для проверки хешей (публичный доступ)
		public.POST("/api/verify-hash", a.publicVerifyHashAPI)
	}
}

// publicVerifyPage отображает публичную страницу проверки результатов
func (a *AdminPanel) publicVerifyPage(c *gin.Context) {
	// Получаем параметры пагинации
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage := 10

	// Опционально получаем ID для подсветки
	highlightID, _ := strconv.ParseUint(c.Query("highlight"), 10, 64)

	// Получаем хеши с пагинацией
	entries, totalPages, err := a.service.GetHashEntries(page, perPage)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Ошибка",
			"error": err.Error(),
		})
		return
	}

	// Получаем текущий активный раунд (который не должен отображаться)
	currentRound, _ := a.service.GetCurrentRound()

	// Подготавливаем данные для шаблона
	hashEntries := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		// Пропускаем незавершенные раунды и текущий активный раунд для публичной страницы
		if !entry.IsCompleted || (currentRound != nil && entry.ID == currentRound.ID) {
			continue
		}

		// Используем utils.GetColorForNumber для определения цвета
		colorStr := utils.GetColorForNumber(entry.Number)
		// Преобразуем "zero (green)" в просто "zero" для совместимости
		if colorStr == "zero (green)" {
			colorStr = "zero"
		}

		// Конвертируем ID в Base62 формат
		base62ID := utils.ToBase62(uint(entry.ID))

		hashEntries = append(hashEntries, gin.H{
			"ID":        entry.ID,
			"Base62ID":  base62ID,
			"Number":    entry.Number,
			"Color":     colorStr,
			"Hash":      entry.Hash,
			"SaltHEX":   entry.SaltHEX,
			"CreatedAt": entry.CreatedAt.Format("02.01.2006 15:04:05"),
		})
	}

	// Подготавливаем массив номеров страниц для пагинации
	var pagination []int
	if totalPages <= 7 {
		// Если общее количество страниц не больше 7, показываем все
		for i := 1; i <= totalPages; i++ {
			pagination = append(pagination, i)
		}
	} else {
		// Иначе показываем текущую страницу, 2 предыдущих, 2 следующих, первую и последнюю
		pagination = []int{1}

		start := page - 2
		if start < 2 {
			start = 2
		}

		end := page + 2
		if end >= totalPages {
			end = totalPages - 1
		}

		if start > 2 {
			pagination = append(pagination, -1) // Вставляем "..."
		}

		for i := start; i <= end; i++ {
			pagination = append(pagination, i)
		}

		if end < totalPages-1 {
			pagination = append(pagination, -1) // Вставляем "..."
		}

		pagination = append(pagination, totalPages)
	}

	c.HTML(http.StatusOK, "public_hashes", gin.H{
		"title":       "Проверка результатов | Roulette Bot",
		"entries":     hashEntries,
		"currentPage": page,
		"totalPages":  totalPages,
		"pagination":  pagination,
		"highlightID": highlightID,
	})
}

// publicHomePage отображает публичную главную страницу
func (a *AdminPanel) publicHomePage(c *gin.Context) {
	// Получаем статистику для отображения на главной странице
	stats := gin.H{
		"players": "10,000+",
		"bets":    "1,500,000+",
		"prizes":  "25,000+",
	}

	// Пытаемся получить реальную статистику из БД
	totalStats, err := a.repo.GetTotalStats()
	if err == nil {
		// Если удалось получить статистику из БД, используем ее
		userCount, _ := a.repo.GetUserCount()

		// Форматируем значения для более приятного отображения
		players := formatNumberWithSuffix(int(userCount))
		bets := formatNumberWithSuffix(int(totalStats["totalBets"]))

		// Рассчитываем количество призов (примерно)
		prizesCount := int(totalStats["totalPoints"] / 100)
		prizes := formatNumberWithSuffix(prizesCount)

		stats = gin.H{
			"players": players,
			"bets":    bets,
			"prizes":  prizes,
		}
	}

	c.HTML(http.StatusOK, "public_home", gin.H{
		"title": "Roulette Bot | Социальное казино в Telegram",
		"stats": stats,
	})
}

// formatNumberWithSuffix форматирует число с суффиксом k, M, B для больших чисел
func formatNumberWithSuffix(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	} else if n < 1000000 {
		k := float64(n) / 1000.0
		return fmt.Sprintf("%.1f%s", k, "k+")
	} else if n < 1000000000 {
		m := float64(n) / 1000000.0
		return fmt.Sprintf("%.1f%s", m, "M+")
	}
	b := float64(n) / 1000000000.0
	return fmt.Sprintf("%.1f%s", b, "B+")
}

// publicFaqPage отображает публичную страницу с часто задаваемыми вопросами
func (a *AdminPanel) publicFaqPage(c *gin.Context) {
	c.HTML(http.StatusOK, "public_faq", gin.H{
		"title": "FAQ | Roulette Bot",
	})
}

// hashVerificationExample отображает страницу с примером проверки хеша
func (a *AdminPanel) hashVerificationExample(c *gin.Context) {
	c.HTML(http.StatusOK, "hash_verification_example", gin.H{
		"title": "Пример проверки хеша | Roulette Bot",
	})
}

// publicVerifyHashAPI - публичный API эндпоинт для проверки хеша
func (a *AdminPanel) publicVerifyHashAPI(c *gin.Context) {
	// Структура для получения данных из запроса
	var verifyRequest struct {
		Number       int64  `json:"number"`
		Salt         string `json:"salt"`
		OriginalHash string `json:"originalHash"`
	}

	// Парсим тело запроса
	if err := c.ShouldBindJSON(&verifyRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	// Проверяем хеш
	data := fmt.Sprintf("%d:%s", verifyRequest.Number, verifyRequest.Salt)
	hash := sha256.New()
	hash.Write([]byte(data))
	computedHash := hex.EncodeToString(hash.Sum(nil))

	// Проверяем валидность
	valid := computedHash == verifyRequest.OriginalHash

	// Возвращаем результат
	c.JSON(http.StatusOK, gin.H{
		"valid":        valid,
		"originalHash": verifyRequest.OriginalHash,
		"computedHash": computedHash,
	})
}
