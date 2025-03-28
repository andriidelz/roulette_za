// Файл: internal/utils/telegram.go

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
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

	photosData, err := ioutil.ReadAll(photosResp.Body)
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

	fileData, err := ioutil.ReadAll(fileResp.Body)
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
