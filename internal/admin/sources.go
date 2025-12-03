package admin

import (
	"net/http"
	"roulette/internal/logger"
	"roulette/internal/models"

	"github.com/gin-gonic/gin"
)

type (
	sourcesKeysStruct struct {
		SourceKey models.SourceKey
		CountOpen interface{}
		CountReg  interface{}
	}
)

// Обработчик списка источников
func (a *AdminPanel) sourcesKeysPage(c *gin.Context) {
	sources, err := a.repo.GetAllSourceKeys()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	// Получаем статистику с разбивкой по источникам
	stat, err := a.service.GetSource()
	if err != nil {
		logger.Error.Println(err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	statData := []sourcesKeysStruct{}
	for i := range sources {
		data := sourcesKeysStruct{SourceKey: sources[i]}
		if count, ok := stat[sources[i].Key]; ok {
			if count_open, ok1 := count["count_open"]; ok1 {
				data.CountOpen = count_open
			}
			if count_reg, ok1 := count["count_reg"]; ok1 {
				data.CountReg = count_reg
			}
		}
		statData = append(statData, data)
	}

	c.HTML(http.StatusOK, "sources_keys", gin.H{
		"title":        "Admin-panel - Referral Link",
		"activeTab":    "sources",
		"activeSubTab": "keys",
		"sources":      statData,
		"botName":      a.settings.BotName,
	})
}

// Обработчик сохранения источника
func (a *AdminPanel) sourceKeysSave(c *gin.Context) {
	// Получаем ключ
	key := c.Param("key")

	// Получаем данные из формы
	name := c.PostForm("name")

	if err := a.repo.SetSourceKey(key, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Обработчик добавления новый источник
func (a *AdminPanel) sourceKeysAdd(c *gin.Context) {
	// Получаем данные из формы
	key := c.PostForm("key")
	name := c.PostForm("name")

	// Проверяем, существует ли уже такой источник
	exists, _ := a.repo.CheckSourceKeyExists(key)
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ключ уже существует"})
		return
	}

	// Проверяем, что все необходимые данные предоставлены
	if key == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Все поля должны быть заполнены"})
		return
	}

	// Проверяем, что ключ имеет допустимый формат
	if !a.isValidSourceKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ключ может содержать только латинские буквы и цифры"})
		return
	}

	// Сохраняем источник
	if err := a.repo.SetSourceKey(key, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Проверка валидности ключа источника
func (a *AdminPanel) isValidSourceKey(key string) bool {
	for _, char := range key {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}
