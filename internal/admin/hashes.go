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

// Сторінка хешів
func (a *AdminPanel) hashesPage(c *gin.Context) {
	// Отримуємо параметри пагінації
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage := 10

	// Отримуємо хеші з пагінацією
	entries, totalPages, err := a.service.GetHashEntries(page, perPage)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Помилка",
			"error": err.Error(),
		})
		return
	}

	// Підготовлюємо дані для шаблону
	hashEntries := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		colorStr := "black"
		if entry.Number == 0 {
			colorStr = "zero"
		} else {
			// Визначаємо колір для числа (червоне або чорне)
			redNumbers := []int64{1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36}
			isRed := false
			for _, n := range redNumbers {
				if entry.Number == n {
					isRed = true
					break
				}
			}
			if isRed {
				colorStr = "red"
			}
		}

		// Конвертуємо ID в Base62 формат
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

	// Підготовлюємо масив номерів сторінок для пагінації
	var pagination []int
	if totalPages <= 7 {
		// Якщо загальна кількість сторінок не більше 7, показуємо всі
		for i := 1; i <= totalPages; i++ {
			pagination = append(pagination, i)
		}
	} else {
		// Інакше показуємо поточну сторінку, 2 попередніх, 2 наступних, першу та останню
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
			pagination = append(pagination, -1) // Вставляємо "..."
		}

		for i := start; i <= end; i++ {
			pagination = append(pagination, i)
		}

		if end < totalPages-1 {
			pagination = append(pagination, -1) // Вставляємо "..."
		}

		pagination = append(pagination, totalPages)
	}

	c.HTML(http.StatusOK, "hashes.html", gin.H{
		"title":       "Адмін-панель - Хеші рулетки",
		"activeTab":   "hashes",
		"entries":     hashEntries,
		"currentPage": page,
		"totalPages":  totalPages,
		"pagination":  pagination,
	})
}

// API ендпоінт для отримання списку хешів
func (a *AdminPanel) getHashesAPI(c *gin.Context) {
	// Отримуємо параметри пагінації
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage := 10

	// Отримуємо хеші з пагінацією
	entries, totalPages, err := a.service.GetHashEntries(page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Підготовлюємо дані для відповіді
	hashEntries := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		colorStr := "black"
		if entry.Number == 0 {
			colorStr = "zero"
		} else {
			// Визначаємо колір для числа (червоне або чорне)
			redNumbers := []int64{1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36}
			isRed := false
			for _, n := range redNumbers {
				if entry.Number == n {
					isRed = true
					break
				}
			}
			if isRed {
				colorStr = "red"
			}
		}

		// Конвертуємо ID в Base62 формат
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

// API ендпоінт для отримання деталей конкретного хешу
func (a *AdminPanel) getHashDetailsAPI(c *gin.Context) {
	// Отримуємо ID хешу з URL
	hashID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний ID хешу"})
		return
	}

	// Отримуємо хеш з бази даних
	entries, _, err := a.service.GetHashEntries(1, 100) // Тимчасове рішення, поки не буде методу GetHashByID
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Шукаємо потрібний хеш
	var targetEntry *models.HashEntry
	for _, entry := range entries {
		if entry.ID == uint(hashID) {
			targetEntry = &entry
			break
		}
	}

	if targetEntry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Хеш не знайдено"})
		return
	}

	// Визначаємо колір
	colorStr := "black"
	if targetEntry.Number == 0 {
		colorStr = "zero"
	} else {
		// Визначаємо колір для числа (червоне або чорне)
		redNumbers := []int64{1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36}
		isRed := false
		for _, n := range redNumbers {
			if targetEntry.Number == n {
				isRed = true
				break
			}
		}
		if isRed {
			colorStr = "red"
		}
	}

	// Конвертуємо ID в Base62 формат
	base62ID := utils.ToBase62(uint(targetEntry.ID))

	// Повертаємо дані
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

// API ендпоінт для перевірки хешу
func (a *AdminPanel) verifyHashAPI(c *gin.Context) {
	// Структура для отримання даних з запиту
	var verifyRequest struct {
		Number       int64  `json:"number"`
		Salt         string `json:"salt"`
		OriginalHash string `json:"originalHash"`
	}

	// Парсимо тіло запиту
	if err := c.ShouldBindJSON(&verifyRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невірний формат запиту"})
		return
	}

	// Перевіряємо хеш
	data := fmt.Sprintf("%d:%s", verifyRequest.Number, verifyRequest.Salt)
	hash := sha256.New()
	hash.Write([]byte(data))
	computedHash := hex.EncodeToString(hash.Sum(nil))

	// Перевіряємо валідність
	valid := computedHash == verifyRequest.OriginalHash

	// Повертаємо результат
	c.JSON(http.StatusOK, gin.H{
		"valid":        valid,
		"originalHash": verifyRequest.OriginalHash,
		"computedHash": computedHash,
	})
}
