package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"roulette/internal/models"
	"roulette/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// setupPublicRoutes настраивает публичные маршруты
func (a *AdminPanel) setupPublicRoutes() {
	// Публичные страницы (не требуют авторизации)
	public := a.router.Group("/")
	{
		public.GET("/", a.publicHome)
		public.GET("/hashes", a.publicHashes)
		public.GET("/faq", a.publicFAQ)
		public.GET("/example", a.publicExample)

		// API для проверки хешей (публичный доступ)
		public.POST("/api/verify-hash", a.publicVerifyHashAPI)
	}
}

// publicHashes отображает публичную страницу проверки результатов
func (a *AdminPanel) publicHashes(c *gin.Context) {
	// Получаем параметры пагинации
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage := 10
	var totalPages int
	var entries []models.HashEntry

	// Опционально получаем ID для подсветки
	highlightID, _ := strconv.ParseUint(c.Query("highlight"), 10, 64)

	id := c.Query("id")
	if id != "" {
		// Якщо вказано id
		hashID := utils.FromBase62(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID хеша"})
			return
		}

		targetEntry, err := a.service.GetHashEntryByID(uint(hashID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if targetEntry == nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"title": "Ошибка",
				"error": "Хеш не найден",
			})
			return
		}
		totalPages = 1
		entries = append(entries, *targetEntry)

	} else {

		// Получаем хеши с пагинацией
		entries, totalPages, err = a.service.GetHashEntries(page, perPage)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"title": "Ошибка",
				"error": err.Error(),
			})
			return
		}
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
		"activeTab":   "hashes",
		"entries":     hashEntries,
		"currentPage": page,
		"totalPages":  totalPages,
		"pagination":  pagination,
		"highlightID": highlightID,
		"id":          id,
		"cssFiles": []string{
			"home.css",
		},
		"jsFiles": []string{
			"verification.js",
		},
	})
}

// publicHome отображает публичную главную страницу
func (a *AdminPanel) publicHome(c *gin.Context) {
	c.HTML(http.StatusOK, "public_home", gin.H{
		"title":     "Roulette Bot | Социальное казино в Telegram",
		"activeTab": "home",
		"cssFiles": []string{
			"home.css",
		},
		"jsFiles": []string{
			"roulette.js",
		},
	})
}

// publicFAQ отображает публичную страницу с часто задаваемыми вопросами
func (a *AdminPanel) publicFAQ(c *gin.Context) {
	c.HTML(http.StatusOK, "public_faq", gin.H{
		"title":     "FAQ | Roulette Bot",
		"activeTab": "faq",
		"cssFiles": []string{
			"faq.css",
		},
		"jsFiles": []string{},
	})
}

// publicExample отображает страницу с примером проверки хеша
func (a *AdminPanel) publicExample(c *gin.Context) {
	c.HTML(http.StatusOK, "public_example", gin.H{
		"title":     "Пример проверки хеша | Roulette Bot",
		"activeTab": "example",
		"cssFiles":  []string{},
		"jsFiles": []string{
			"example.js",
		},
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
