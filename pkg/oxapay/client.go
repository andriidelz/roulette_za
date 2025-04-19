package oxapay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gorm.io/gorm"
)

const (
	// Базовый URL API OxaPay
	apiBaseURL = "https://api.oxapay.com"
)

// Client представляет клиент для взаимодействия с API OxaPay
type Client struct {
	apiKey     string
	httpClient *http.Client
	db         *gorm.DB
	baseURL    string // URL API
}

// NewClient создает новый клиент OxaPay
func NewClient(config Config) *Client {
	return &Client{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		db:      config.DB,
		baseURL: apiBaseURL,
	}
}

// CreatePayout создает новый запрос на вывод средств
func (c *Client) CreatePayout(request PayoutRequest) (*Payout, error) {
	// Преобразуем запрос в формат API
	apiRequest := map[string]interface{}{
		"address":  request.Address,
		"currency": request.Currency,
		"amount":   request.Amount,
	}

	// Добавляем опциональные поля
	if request.Description != "" {
		apiRequest["description"] = request.Description
	}

	if request.Memo != "" {
		apiRequest["memo"] = request.Memo
	}

	if request.Network != "" {
		apiRequest["network"] = request.Network
	}

	// Преобразуем запрос в JSON
	jsonData, err := json.Marshal(apiRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payout request: %w", err)
	}

	// Создаем HTTP-запрос
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/payout", c.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating payout request: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("payout_api_key", c.apiKey)

	// Отправляем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending payout request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating payout, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Анализируем ответ
	var response struct {
		Data struct {
			TrackID string `json:"track_id"`
			Status  string `json:"status"`
		} `json:"data"`
		Message string      `json:"message"`
		Error   interface{} `json:"error"`
		Status  int         `json:"status"`
		Version string      `json:"version"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	if response.Status != http.StatusOK {
		return nil, fmt.Errorf("error from OxaPay: %s", response.Message)
	}

	// Создаем и заполняем объект Payout
	payout := &Payout{
		ID:          response.Data.TrackID,
		Currency:    request.Currency,
		Amount:      request.Amount,
		Address:     request.Address,
		Description: request.Description,
		UserID:      request.UserID,
		Status:      response.Data.Status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Сохраняем в БД, если она доступна
	if c.db != nil {
		if err := c.db.Create(payout).Error; err != nil {
			return nil, fmt.Errorf("error saving payout to database: %w", err)
		}
	}

	return payout, nil
}

// GetPayout получает информацию о выводе средств по ID
func (c *Client) GetPayout(id string) (*Payout, error) {
	// Создаем HTTP-запрос
	url := fmt.Sprintf("%s/v1/payout/%s", c.baseURL, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("payout_api_key", c.apiKey)

	// Отправляем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting payout, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Анализируем ответ
	var response struct {
		Data struct {
			TrackID     string  `json:"track_id"`
			Address     string  `json:"address"`
			Currency    string  `json:"currency"`
			Network     string  `json:"network"`
			Amount      float64 `json:"amount"`
			Fee         float64 `json:"fee"`
			Status      string  `json:"status"`
			TxHash      string  `json:"tx_hash"`
			Description string  `json:"description"`
			Internal    bool    `json:"internal"`
			Memo        string  `json:"memo"`
			Date        int64   `json:"date"`
		} `json:"data"`
		Message string      `json:"message"`
		Error   interface{} `json:"error"`
		Status  int         `json:"status"`
		Version string      `json:"version"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	if response.Status != http.StatusOK {
		return nil, fmt.Errorf("error from OxaPay: %s", response.Message)
	}

	// Преобразуем ответ API в объект Payout
	payout := &Payout{
		ID:              response.Data.TrackID,
		Currency:        response.Data.Currency,
		Amount:          response.Data.Amount,
		Address:         response.Data.Address,
		Description:     response.Data.Description,
		Status:          response.Data.Status,
		TransactionHash: response.Data.TxHash,
		Network:         response.Data.Network,
		Fee:             response.Data.Fee,
		Memo:            response.Data.Memo,
		Internal:        response.Data.Internal,
		CreatedAt:       time.Unix(response.Data.Date, 0),
		UpdatedAt:       time.Now(),
	}

	// Обновляем в БД, если она доступна
	if c.db != nil {
		if err := c.db.Save(payout).Error; err != nil {
			return nil, fmt.Errorf("error updating payout in database: %w", err)
		}
	}

	return payout, nil
}

// InitializeTables создает необходимые таблицы в базе данных
func (c *Client) InitializeTables() error {
	if c.db == nil {
		return fmt.Errorf("database connection is not configured")
	}

	// Миграция таблиц
	if err := c.db.AutoMigrate(&Payout{}); err != nil {
		return fmt.Errorf("failed to migrate database tables: %w", err)
	}

	return nil
}
