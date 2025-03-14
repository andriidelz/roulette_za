package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Структура для отображения в локализации в шаблоне
type LocalizationView struct {
	Key   string
	Value string
}

// Обработчик списка локализаций
func (a *AdminPanel) localizationsList(c *gin.Context) {
	// Получаем локализации для всех языков
	ukLocalizations, err := a.getLocalizationsForLanguage("uk")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	enLocalizations, err := a.getLocalizationsForLanguage("en")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	ruLocalizations, err := a.getLocalizationsForLanguage("ru")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "localizations", gin.H{
		"title":     "Admin-panel - Localizations",
		"activeTab": "localizations",
		"localizations": gin.H{
			"uk": ukLocalizations,
			"en": enLocalizations,
			"ru": ruLocalizations,
		},
	})
}

// Обработчик редактирования локализации
func (a *AdminPanel) localizationEdit(c *gin.Context) {
	// Получаем ключ локализации
	key := c.Param("key")

	// Получаем локализации для всех языков
	ukValue, _ := a.repo.GetLocalization(key, "uk")
	enValue, _ := a.repo.GetLocalization(key, "en")
	ruValue, _ := a.repo.GetLocalization(key, "ru")

	c.HTML(http.StatusOK, "localization_edit", gin.H{
		"title":     "Admin-panel - Edit Localization",
		"activeTab": "localizations",
		"key":       key,
		"values": gin.H{
			"uk": ukValue,
			"en": enValue,
			"ru": ruValue,
		},
	})
}

// Обработчик сохранения локализации
func (a *AdminPanel) localizationSave(c *gin.Context) {
	// Получаем ключ локализации
	key := c.Param("key")

	// Получаем данные из формы для каждого языка
	ukValue := c.PostForm("uk")
	enValue := c.PostForm("en")
	ruValue := c.PostForm("ru")

	// Сохраняем локализации для каждого языка, если они предоставлены
	if ukValue != "" {
		if err := a.repo.SetLocalization(key, "uk", ukValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if enValue != "" {
		if err := a.repo.SetLocalization(key, "en", enValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if ruValue != "" {
		if err := a.repo.SetLocalization(key, "ru", ruValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Обработчик удаления локализации
func (a *AdminPanel) localizationDelete(c *gin.Context) {
	// Получаем ключ локализации
	key := c.Param("key")

	// Удаляем локализации для всех языков
	if err := a.repo.DeleteLocalization(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Обработчик добавления новой локализации
func (a *AdminPanel) localizationAdd(c *gin.Context) {
	// Получаем данные из формы
	key := c.PostForm("key")
	ukValue := c.PostForm("uk")
	enValue := c.PostForm("en")
	ruValue := c.PostForm("ru")

	// Проверяем, что все необходимые данные предоставлены
	if key == "" || ukValue == "" || enValue == "" || ruValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Все поля должны быть заполнены"})
		return
	}

	// Проверяем, что ключ имеет допустимый формат
	if !a.isValidKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ключ может содержать только латинские буквы, цифры и нижнее подчеркивание"})
		return
	}

	// Сохраняем локализации для каждого языка
	if err := a.repo.SetLocalization(key, "uk", ukValue); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := a.repo.SetLocalization(key, "en", enValue); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := a.repo.SetLocalization(key, "ru", ruValue); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Получить все локализации для указанного языка
func (a *AdminPanel) getLocalizationsForLanguage(language string) ([]LocalizationView, error) {
	localizations, err := a.repo.GetAllLocalizationsForLanguage(language)
	if err != nil {
		return nil, err
	}

	// Преобразуем в удобный для шаблона формат
	result := make([]LocalizationView, 0, len(localizations))
	for _, loc := range localizations {
		result = append(result, LocalizationView{
			Key:   loc.Key,
			Value: loc.Value,
		})
	}

	return result, nil
}

// Проверка валидности ключа локализации
func (a *AdminPanel) isValidKey(key string) bool {
	for _, char := range key {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}
	return true
}
