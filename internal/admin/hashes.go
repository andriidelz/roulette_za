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

// Страница хешей
func (a *AdminPanel) hashesPage(c *gin.Context) {
	// Получаем параметры пагинации
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage := 10

	// Получаем хеши с пагинацией
	entries, totalPages, err := a.service.GetHashEntries(page, perPage)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Ошибка",
			"error": err.Error(),
		})
		return
	}

	// Подготавливаем данные для шаблона
	hashEntries := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
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

	c.HTML(http.StatusOK, "hashes", gin.H{
		"title":       "Admin-panel - Hashes",
		"activeTab":   "hashes",
		"entries":     hashEntries,
		"currentPage": page,
		"totalPages":  totalPages,
		"pagination":  pagination,
	})
}

// API эндпоинт для получения списка хешей
func (a *AdminPanel) getHashesAPI(c *gin.Context) {
	// Получаем параметры пагинации
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage := 10

	// Получаем хеши с пагинацией
	entries, totalPages, err := a.service.GetHashEntries(page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Подготавливаем данные для ответа
	hashEntries := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
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

	c.JSON(http.StatusOK, gin.H{
		"entries":     hashEntries,
		"currentPage": page,
		"totalPages":  totalPages,
	})
}

// API эндпоинт для получения деталей конкретного хеша
func (a *AdminPanel) getHashDetailsAPI(c *gin.Context) {
	// Получаем ID хеша из URL
	hashID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID хеша"})
		return
	}

	// Получаем хеш из базы данных
	entries, _, err := a.service.GetHashEntries(1, 100) // Временное решение, пока не будет метода GetHashByID
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ищем нужный хеш
	var targetEntry *models.HashEntry
	for _, entry := range entries {
		if entry.ID == uint(hashID) {
			targetEntry = &entry
			break
		}
	}

	if targetEntry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Хеш не найден"})
		return
	}

	// Используем utils.GetColorForNumber для определения цвета
	colorStr := utils.GetColorForNumber(targetEntry.Number)
	// Преобразуем "zero (green)" в просто "zero" для совместимости
	if colorStr == "zero (green)" {
		colorStr = "zero"
	}

	// Конвертируем ID в Base62 формат
	base62ID := utils.ToBase62(uint(targetEntry.ID))

	// Возвращаем данные
	c.JSON(http.StatusOK, gin.H{
		"ID":        targetEntry.ID,
		"Base62ID":  base62ID,
		"Number":    targetEntry.Number,
		"Color":     colorStr,
		"Hash":      targetEntry.Hash,
		"SaltHEX":   targetEntry.SaltHEX,
		"CreatedAt": targetEntry.CreatedAt.Format("02.01.2006 15:04:05"),
	})
}

// API эндпоинт для проверки хеша
func (a *AdminPanel) verifyHashAPI(c *gin.Context) {
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
