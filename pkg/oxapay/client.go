package oxapay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	log.Printf("[OxaPayClient] Initializing client with API URL: %s", apiBaseURL)
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
	log.Printf("[OxaPayClient] Creating payout: currency=%s, amount=%.2f, address=%s",
		request.Currency, request.Amount, request.Address)

	// Преобразуем запрос в формат API
	apiRequest := map[string]interface{}{
		"address":  request.Address,
		"currency": request.Currency,
		"amount":   request.Amount,
	}

	// Добавляем опциональные поля
	if request.Description != "" {
		apiRequest["description"] = request.Description
		log.Printf("[OxaPayClient] Added description: %s", request.Description)
	}

	if request.Memo != "" {
		apiRequest["memo"] = request.Memo
		log.Printf("[OxaPayClient] Added memo field")
	}

	if request.Network != "" {
		apiRequest["network"] = request.Network
		log.Printf("[OxaPayClient] Using network: %s", request.Network)
	}

	// Преобразуем запрос в JSON
	jsonData, err := json.Marshal(apiRequest)
	if err != nil {
		log.Printf("[OxaPayClient] Error marshaling request to JSON: %v", err)
		return nil, fmt.Errorf("error marshaling payout request: %w", err)
	}

	// Создаем HTTP-запрос
	url := fmt.Sprintf("%s/v1/payout", c.baseURL)
	log.Printf("[OxaPayClient] Sending request to: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[OxaPayClient] Error creating HTTP request: %v", err)
		return nil, fmt.Errorf("error creating payout request: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("payout_api_key", c.apiKey)

	// Отправляем запрос
	log.Printf("[OxaPayClient] Sending HTTP request...")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[OxaPayClient] HTTP request failed: %v", err)
		return nil, fmt.Errorf("error sending payout request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[OxaPayClient] Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		log.Printf("[OxaPayClient] Request failed with status code %d. Response: %s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("error creating payout, status code: %d, body: %s",
			resp.StatusCode, string(body))
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
		log.Printf("[OxaPayClient] Error parsing response JSON: %v", err)
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	if response.Status != http.StatusOK {
		log.Printf("[OxaPayClient] API returned error: %s", response.Message)
		return nil, fmt.Errorf("error from OxaPay: %s", response.Message)
	}

	log.Printf("[OxaPayClient] Payout created successfully: ID=%s, Status=%s",
		response.Data.TrackID, response.Data.Status)

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
		log.Printf("[OxaPayClient] Saving payout to database")
		if err := c.db.Create(payout).Error; err != nil {
			log.Printf("[OxaPayClient] Error saving to database: %v", err)
			return nil, fmt.Errorf("error saving payout to database: %w", err)
		}
	}

	return payout, nil
}

// GetPayout получает информацию о выводе средств по ID
func (c *Client) GetPayout(id string) (*Payout, error) {
	log.Printf("[OxaPayClient] Getting payout info for ID: %s", id)

	// Создаем HTTP-запрос
	url := fmt.Sprintf("%s/v1/payout/%s", c.baseURL, id)
	log.Printf("[OxaPayClient] Sending request to: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[OxaPayClient] Error creating HTTP request: %v", err)
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("payout_api_key", c.apiKey)

	// Отправляем запрос
	log.Printf("[OxaPayClient] Sending HTTP request...")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[OxaPayClient] HTTP request failed: %v", err)
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[OxaPayClient] Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		log.Printf("[OxaPayClient] Request failed with status code %d. Response: %s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("error getting payout, status code: %d, body: %s",
			resp.StatusCode, string(body))
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
		log.Printf("[OxaPayClient] Error parsing response JSON: %v", err)
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	if response.Status != http.StatusOK {
		log.Printf("[OxaPayClient] API returned error: %s", response.Message)
		return nil, fmt.Errorf("error from OxaPay: %s", response.Message)
	}

	log.Printf("[OxaPayClient] Got payout status: ID=%s, Status=%s, TxHash=%s",
		response.Data.TrackID, response.Data.Status, response.Data.TxHash)

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
		log.Printf("[OxaPayClient] Updating payout in database")
		if err := c.db.Save(payout).Error; err != nil {
			log.Printf("[OxaPayClient] Error updating database: %v", err)
			return nil, fmt.Errorf("error updating payout in database: %w", err)
		}
	}

	return payout, nil
}

// InitializeTables создает необходимые таблицы в базе данных
func (c *Client) InitializeTables() error {
	if c.db == nil {
		log.Printf("[OxaPayClient] Cannot initialize tables: database connection is not configured")
		return fmt.Errorf("database connection is not configured")
	}

	log.Printf("[OxaPayClient] Initializing database tables")

	// Миграция таблиц
	if err := c.db.AutoMigrate(&Payout{}); err != nil {
		log.Printf("[OxaPayClient] Database migration failed: %v", err)
		return fmt.Errorf("failed to migrate database tables: %w", err)
	}

	log.Printf("[OxaPayClient] Database tables initialized successfully")
	return nil
}
