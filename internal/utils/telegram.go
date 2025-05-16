// Файл: internal/utils/telegram.go

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mymmrac/telego"
)

// TelegramUserProfilePhotos структура для хранения ответа от Telegram API на запрос getUserProfilePhotos
type TelegramUserProfilePhotos struct {
	Ok     bool `json:"ok"`
	Result struct {
		TotalCount int `json:"total_count"`
		Photos     [][]struct {
			FileID       string `json:"file_id"`
			FileUniqueID string `json:"file_unique_id"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			FileSize     int    `json:"file_size,omitempty"`
		} `json:"photos"`
	} `json:"result"`
}

// TelegramFile структура для хранения ответа от Telegram API на запрос getFile
type TelegramFile struct {
	Ok     bool `json:"ok"`
	Result struct {
		FileID       string `json:"file_id"`
		FileUniqueID string `json:"file_unique_id"`
		FileSize     int    `json:"file_size,omitempty"`
		FilePath     string `json:"file_path"`
	} `json:"result"`
}

// GetUserProfilePhoto получает URL аватарки пользователя из Telegram API
func GetUserProfilePhoto(telegramToken string, userID int64) (string, error) {
	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Запрос на получение фотографий профиля пользователя
	photosURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUserProfilePhotos?user_id=%d&limit=1", telegramToken, userID)
	photosResp, err := client.Get(photosURL)
	if err != nil {
		return "", fmt.Errorf("error getting user profile photos: %w", err)
	}
	defer photosResp.Body.Close()

	photosData, err := io.ReadAll(photosResp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading photos response: %w", err)
	}

	var photos TelegramUserProfilePhotos
	if err := json.Unmarshal(photosData, &photos); err != nil {
		return "", fmt.Errorf("error unmarshaling photos response: %w", err)
	}

	// Проверяем, есть ли у пользователя фотографии профиля
	if !photos.Ok || photos.Result.TotalCount == 0 || len(photos.Result.Photos) == 0 || len(photos.Result.Photos[0]) == 0 {
		return "", nil // Нет фотографий профиля
	}

	// Берем fileID последней (самой новой) фотографии
	fileID := photos.Result.Photos[0][0].FileID

	// Запрос на получение пути к файлу
	fileURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", telegramToken, fileID)
	fileResp, err := client.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("error getting file info: %w", err)
	}
	defer fileResp.Body.Close()

	fileData, err := io.ReadAll(fileResp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading file response: %w", err)
	}

	var telegramFile TelegramFile
	if err := json.Unmarshal(fileData, &telegramFile); err != nil {
		return "", fmt.Errorf("error unmarshaling file response: %w", err)
	}

	if !telegramFile.Ok || telegramFile.Result.FilePath == "" {
		return "", fmt.Errorf("error getting file path")
	}

	// Формируем прямую ссылку на файл
	avatarURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", telegramToken, telegramFile.Result.FilePath)

	return avatarURL, nil
}

// Шаблон регулярного выражения для кастомных эмодзи в формате {{emoji:UNICODE:ID}}
var emojiPattern = regexp.MustCompile(`\{\{emoji:(.+?):([0-9]+)\}\}`)

// toUTF16Count преобразует индекс из UTF-8 строки в индекс UTF-16
func toUTF16Count(s string) int {
	utf16Count := 0
	for _, r := range s {
		if r <= 0xFFFF {
			utf16Count++
		} else {
			// Символы вне BMP (Basic Multilingual Plane) занимают 2 кодовые единицы в UTF-16
			utf16Count += 2
		}
	}
	return utf16Count
}

// BuildMessageWithCustomEmojis заменяет плейсхолдеры вида {{emoji:UNICODE:ID}}
// на Unicode-эмодзи и создаёт массив MessageEntity для кастомных эмодзи Telegram.
func BuildMessageWithCustomEmojis(textTemplate string) (string, []telego.MessageEntity) {
	// Если в тексте нет шаблонов, возвращаем исходный текст без изменений
	if !strings.Contains(textTemplate, "{{emoji:") {
		return textTemplate, []telego.MessageEntity{}
	}

	// Ищем все совпадения шаблона
	matches := emojiPattern.FindAllStringSubmatch(textTemplate, -1)
	if len(matches) == 0 {
		return textTemplate, []telego.MessageEntity{}
	}

	// Результирующий текст и сущности
	resultText := textTemplate
	entities := []telego.MessageEntity{}

	// Обрабатываем каждое совпадение, начиная с конца, чтобы не нарушить индексы
	for i := len(matches) - 1; i >= 0; i-- {
		fullMatch := matches[i][0]    // {{emoji:UNICODE:ID}}
		unicodeEmoji := matches[i][1] // UNICODE
		emojiID := matches[i][2]      // ID

		// Находим позицию шаблона
		startPos := strings.Index(resultText, fullMatch)
		if startPos == -1 {
			continue // Странная ситуация, но лучше проверить
		}

		// Вычисляем UTF-16 смещение
		prefixText := resultText[:startPos]
		utf16Offset := toUTF16Count(prefixText)

		// Заменяем шаблон на Unicode-эмодзи
		resultText = resultText[:startPos] + unicodeEmoji + resultText[startPos+len(fullMatch):]

		// Добавляем сущность для кастомного эмодзи
		// Длина в UTF-16 кодовых единицах
		utf16Length := toUTF16Count(unicodeEmoji)

		entities = append(entities, telego.MessageEntity{
			Type:          "custom_emoji",
			Offset:        utf16Offset,
			Length:        utf16Length,
			CustomEmojiID: emojiID,
		})
	}

	return resultText, entities
}
