package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"roulette/internal/models"
	"roulette/internal/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Структура для отображения в локализации в шаблоне
type LocalizationView struct {
	Key   string
	Value string
}

// LocalizationExportData структура для экспорта локализаций
type LocalizationExportData struct {
	Language      string            `json:"language"`
	Localizations map[string]string `json:"localizations"`
	ExportedAt    string            `json:"exported_at"`
	TotalCount    int               `json:"total_count"`
}

// LocalizationImportData структура для импорта локализаций
type LocalizationImportData struct {
	Language      string            `json:"language"`
	Localizations map[string]string `json:"localizations"`
}

// Обработчик списка локализаций
func (a *AdminPanel) localizationsList(c *gin.Context) {
	// Получаем локализации для всех языков
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

	ukLocalizations, err := a.getLocalizationsForLanguage("uk")
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
			"en": enLocalizations,
			"ru": ruLocalizations,
			"uk": ukLocalizations,
		},
	})
}

// Обработчик редактирования локализации
func (a *AdminPanel) localizationEdit(c *gin.Context) {
	// Получаем ключ локализации
	key := c.Param("key")

	// Получаем локализации для всех языков
	enValue, _ := a.repo.GetLocalization(key, "en")
	ruValue, _ := a.repo.GetLocalization(key, "ru")
	ukValue, _ := a.repo.GetLocalization(key, "uk")

	c.HTML(http.StatusOK, "localization_edit", gin.H{
		"title":     "Admin-panel - Edit Localization",
		"activeTab": "localizations",
		"key":       key,
		"values": gin.H{
			"en": enValue,
			"ru": ruValue,
			"uk": ukValue,
		},
	})
}

// Обработчик сохранения локализации
func (a *AdminPanel) localizationSave(c *gin.Context) {
	// Получаем ключ локализации
	key := c.Param("key")

	// Получаем данные из формы для каждого языка
	enValue := c.PostForm("en")
	ruValue := c.PostForm("ru")
	ukValue := c.PostForm("uk")

	// Сохраняем локализации для каждого языка, если они предоставлены
	if enValue != "" {
		if err := a.clearAndSaveLocalization(key, "en", enValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if ruValue != "" {
		if err := a.clearAndSaveLocalization(key, "ru", ruValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if ukValue != "" {
		if err := a.clearAndSaveLocalization(key, "uk", ukValue); err != nil {
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
	var request struct {
		Key     string            `json:"key"`
		Message map[string]string `json:"message"` // Локализации сообщения
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, существует ли уже такая локализация
	exists, _ := a.repo.CheckLocalizationExists(request.Key)
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ключ уже существует"})
		return
	}

	// Проверяем, что все необходимые данные предоставлены
	if request.Key == "" || len(request.Message) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Все поля должны быть заполнены"})
		return
	}

	// Проверяем, что ключ имеет допустимый формат
	if !a.isValidKey(request.Key) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ключ может содержать только латинские буквы, цифры и нижнее подчеркивание"})
		return
	}

	// Сохраняем локализации для каждого языка
	for lang, val := range request.Message {
		if err := a.clearAndSaveLocalization(request.Key, lang, val); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Обработчик для получения локализаций по ключу
func (a *AdminPanel) getLocalizationsByKey(c *gin.Context) {
	// Получаем ключ из параметров запроса
	key := c.Param("key")

	// Получаем локализации для всех языков
	en, _ := a.repo.GetLocalization(key, "en")
	ru, _ := a.repo.GetLocalization(key, "ru")
	uk, _ := a.repo.GetLocalization(key, "uk")

	// Формируем ответ
	c.JSON(http.StatusOK, gin.H{
		"en": en,
		"ru": ru,
		"uk": uk,
	})
}

// Обработчик экспорта локализаций
func (a *AdminPanel) localizationExport(c *gin.Context) {
	// Получаем параметр языка из query string
	language := c.Query("lang")

	// Если язык не указан, экспортируем все языки
	if language == "" {
		a.exportAllLocalizations(c)
		return
	}

	// Валидация языка
	if !isValidLanguage(language) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language code"})
		return
	}

	// Получаем локализации для указанного языка
	localizations, err := a.repo.GetAllLocalizationsForLanguage(language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Преобразуем в map для удобства
	localizationMap := make(map[string]string)
	for _, loc := range localizations {
		localizationMap[loc.Key] = loc.Value
	}

	// Создаем структуру для экспорта
	exportData := LocalizationExportData{
		Language:      language,
		Localizations: localizationMap,
		ExportedAt:    time.Now().Format("2006-01-02 15:04:05"),
		TotalCount:    len(localizationMap),
	}

	// Устанавливаем заголовки для скачивания файла
	filename := fmt.Sprintf("localizations_%s_%s.json", language, time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json; charset=utf-8")

	// Отправляем JSON
	c.JSON(http.StatusOK, exportData)
}

// Экспорт всех локализаций
func (a *AdminPanel) exportAllLocalizations(c *gin.Context) {
	languages := []string{"en", "ru", "uk"}
	allLocalizations := make(map[string]LocalizationExportData)

	for _, lang := range languages {
		localizations, err := a.repo.GetAllLocalizationsForLanguage(lang)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		localizationMap := make(map[string]string)
		for _, loc := range localizations {
			localizationMap[loc.Key] = loc.Value
		}

		allLocalizations[lang] = LocalizationExportData{
			Language:      lang,
			Localizations: localizationMap,
			ExportedAt:    time.Now().Format("2006-01-02 15:04:05"),
			TotalCount:    len(localizationMap),
		}
	}

	// Устанавливаем заголовки для скачивания файла
	filename := fmt.Sprintf("localizations_all_%s.json", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json; charset=utf-8")

	// Отправляем JSON
	c.JSON(http.StatusOK, allLocalizations)
}

// Обработчик импорта локализаций
func (a *AdminPanel) localizationImport(c *gin.Context) {
	// Получаем загруженный файл
	file, header, err := c.Request.FormFile("localization_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Проверяем расширение файла
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".json") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JSON files are allowed"})
		return
	}

	// Читаем содержимое файла
	buf := make([]byte, header.Size)
	_, err = file.Read(buf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Пытаемся распарсить как импорт одного языка
	var singleImport LocalizationImportData
	if err := json.Unmarshal(buf, &singleImport); err == nil && singleImport.Language != "" {
		a.importSingleLanguage(c, singleImport)
		return
	}

	// Пытаемся распарсить как импорт всех языков
	var multiImport map[string]LocalizationExportData
	if err := json.Unmarshal(buf, &multiImport); err == nil {
		a.importMultipleLanguages(c, multiImport)
		return
	}

	// Пытаемся распарсить как экспорт одного языка
	var exportImport LocalizationExportData
	if err := json.Unmarshal(buf, &exportImport); err == nil && exportImport.Language != "" {
		singleImport = LocalizationImportData{
			Language:      exportImport.Language,
			Localizations: exportImport.Localizations,
		}
		a.importSingleLanguage(c, singleImport)
		return
	}

	// Если ничего не подошло, возвращаем ошибку
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "Invalid JSON format. Expected localization export or import format",
	})
}

// Импорт локализаций одного языка
func (a *AdminPanel) importSingleLanguage(c *gin.Context, importData LocalizationImportData) {
	// Валидация языка
	if !isValidLanguage(importData.Language) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language code: " + importData.Language})
		return
	}

	// Валидация локализаций
	if len(importData.Localizations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No localizations found in file"})
		return
	}

	// Валидация ключей
	invalidKeys := make([]string, 0)
	for key := range importData.Localizations {
		if !a.isValidKey(key) {
			invalidKeys = append(invalidKeys, key)
		}
	}

	if len(invalidKeys) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "Invalid localization keys found",
			"invalid_keys": invalidKeys,
		})
		return
	}

	// Импортируем локализации
	importedCount := 0
	updatedCount := 0

	for key, value := range importData.Localizations {
		// Проверяем, существует ли уже такая локализация
		exists, err := a.repo.CheckLocalizationExists(key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}

		if err := a.clearAndSaveLocalization(key, importData.Language, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save localization: " + err.Error()})
			return
		}

		if exists {
			updatedCount++
		} else {
			importedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"language":       importData.Language,
		"imported_count": importedCount,
		"updated_count":  updatedCount,
		"total_count":    len(importData.Localizations),
	})
}

// Импорт локализаций нескольких языков
func (a *AdminPanel) importMultipleLanguages(c *gin.Context, multiImport map[string]LocalizationExportData) {
	results := make(map[string]interface{})
	totalImported := 0
	totalUpdated := 0

	for language, data := range multiImport {
		// Валидация языка
		if !isValidLanguage(language) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language code: " + language})
			return
		}

		// Валидация ключей для текущего языка
		invalidKeys := make([]string, 0)
		for key := range data.Localizations {
			if !a.isValidKey(key) {
				invalidKeys = append(invalidKeys, key)
			}
		}

		if len(invalidKeys) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":        fmt.Sprintf("Invalid localization keys found for %s", language),
				"language":     language,
				"invalid_keys": invalidKeys,
			})
			return
		}

		// Импортируем локализации для текущего языка
		importedCount := 0
		updatedCount := 0

		for key, value := range data.Localizations {
			exists, err := a.repo.CheckLocalizationExists(key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
				return
			}

			if err := a.clearAndSaveLocalization(key, language, value); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Failed to save localization for %s: %s", language, err.Error()),
				})
				return
			}

			if exists {
				updatedCount++
			} else {
				importedCount++
			}
		}

		results[language] = gin.H{
			"imported_count": importedCount,
			"updated_count":  updatedCount,
			"total_count":    len(data.Localizations),
		}

		totalImported += importedCount
		totalUpdated += updatedCount
	}

	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"languages":           results,
		"total_imported":      totalImported,
		"total_updated":       totalUpdated,
		"processed_languages": len(results),
	})
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

// clearAndSaveLocalization очищаем и сохраняем локализацию
func (a *AdminPanel) clearAndSaveLocalization(key, lang, val string) error {
	newVal, err := utils.ParseNode(val, true)
	if err != nil {
		return err
	}

	return a.repo.SetLocalization(models.Localization{
		Key:      key,
		Language: lang,
		Value:    newVal,
	})
}

// Валидация языка
func isValidLanguage(language string) bool {
	validLanguages := []string{"en", "ru", "uk"}
	for _, lang := range validLanguages {
		if lang == language {
			return true
		}
	}
	return false
}
