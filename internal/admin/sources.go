package admin

import (
	"net/http"
	"roulette/internal/models"

	"github.com/gin-gonic/gin"
)

// Обработчик списка источников
func (a *AdminPanel) sourcesKeysPage(c *gin.Context) {
	sources, err := a.getAllSourceKeys()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "sources_keys", gin.H{
		"title":        "Admin-panel - Sources",
		"activeTab":    "sources",
		"activeSubTab": "keys",
		"sources":      sources,
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

// Получить все источники
func (a *AdminPanel) getAllSourceKeys() ([]models.SourceKey, error) {
	result, err := a.repo.GetAllSourceKeys()
	if err != nil {
		return nil, err
	}

	return result, nil
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
