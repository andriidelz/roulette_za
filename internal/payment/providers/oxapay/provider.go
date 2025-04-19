package oxapay

import (
	"fmt"
	"net/url"
	"os"
	"roulette/internal/payment"
	oxapayclient "roulette/pkg/oxapay"
	"strconv"

	"gorm.io/gorm"
)

// Provider implements the payment.Provider interface for OxaPay
type Provider struct {
	client      *oxapayclient.Client
	callbackURL string
}

// NewProvider creates a new OxaPay provider
func NewProvider(apiKey, webhookKey, callbackURL string, db interface{}) *Provider {
	// Проверяем, что db имеет нужный тип
	var gormDB *gorm.DB
	var ok bool
	if db != nil {
		gormDB, ok = db.(*gorm.DB)
		if !ok {
			panic("db must be of type *gorm.DB")
		}
	}

	// Создаем конфигурацию клиента OxaPay
	config := oxapayclient.Config{
		APIKey:      apiKey,
		WebhookKey:  webhookKey,
		CallbackURL: callbackURL,
		DB:          gormDB,
	}

	// Создаем клиент OxaPay
	client := oxapayclient.NewClient(config)

	return &Provider{
		client:      client,
		callbackURL: callbackURL,
	}
}

// CreateWithdrawal implements payment.Provider
func (p *Provider) CreateWithdrawal(userID uint, amount float64, currency string, address string) (*payment.Withdrawal, error) {
	// Определяем сеть на основе валюты
	network := p.getNetworkForCurrency(currency)

	// Создаем запрос на вывод средств через OxaPay
	payoutRequest := oxapayclient.PayoutRequest{
		Currency:    currency,
		Amount:      amount,
		Address:     address,
		Network:     network,
		CallbackURL: p.callbackURL,
		Description: fmt.Sprintf("Withdrawal for user %d", userID),
		UserID:      strconv.FormatUint(uint64(userID), 10),
	}

	// Отправляем запрос через клиент OxaPay
	payout, err := p.client.CreatePayout(payoutRequest)
	if err != nil {
		return nil, fmt.Errorf("oxapay withdrawal creation failed: %w", err)
	}

	// Преобразуем статус OxaPay в наш внутренний статус
	status := mapStatus(payout.Status)

	// Создаем и возвращаем объект Withdrawal
	return &payment.Withdrawal{
		ID:              payout.ID,
		UserID:          userID,
		Amount:          amount,
		Currency:        currency,
		Address:         address,
		Status:          status,
		TransactionHash: payout.TransactionHash,
		Description:     payoutRequest.Description,
		ProviderName:    "oxapay",
		ProviderData:    payout, // Сохраняем оригинальный ответ от провайдера
		CreatedAt:       payout.CreatedAt,
		UpdatedAt:       payout.UpdatedAt,
	}, nil
}

// Вспомогательный метод для определения сети на основе валюты
func (p *Provider) getNetworkForCurrency(currency string) string {
	// Здесь можно определить правила для различных валют
	// Можно загрузить эти правила из конфигурации или базы данных
	networkMap := map[string]string{
		"USDT": "TRC20", // По умолчанию используем TRC20 для USDT
		"USDC": "ERC20", // По умолчанию используем ERC20 для USDC
		"BTC":  "Bitcoin Network",
		"ETH":  "Ethereum Network",
		"TRX":  "TRON Network",
		// Добавьте другие валюты и их сети
	}

	if network, exists := networkMap[currency]; exists {
		return network
	}

	// Возвращаем пустую строку для валют, которым не требуется указание сети
	return ""
}

// GetWithdrawalStatus implements payment.Provider
func (p *Provider) GetWithdrawalStatus(withdrawalID string) (payment.WithdrawalStatus, error) {
	// Получаем информацию о выводе средств из OxaPay
	payout, err := p.client.GetPayout(withdrawalID)
	if err != nil {
		return payment.StatusFailed, fmt.Errorf("failed to get withdrawal status: %w", err)
	}

	// Преобразуем статус OxaPay в наш внутренний статус
	return mapStatus(payout.Status), nil
}

// SetupWebhooks implements payment.Provider
func (p *Provider) SetupWebhooks() error {
	// OxaPay webhooks настраиваются при создании клиента и через конфигурацию HTTP маршрутов
	if p.client != nil && p.callbackURL != "" {
		// Извлекаем путь из URL-а колбэка
		callbackPath := extractPathFromURL(p.callbackURL)

		if callbackPath != "" {
			p.client.SetupWebhookHandler(nil, callbackPath)
			return nil
		}

		// Если не удалось извлечь путь, используем путь по умолчанию
		p.client.SetupWebhookHandler(nil, "/webhooks/payment/oxapay")
	}
	return nil
}

// extractPathFromURL извлекает путь из URL
func extractPathFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return parsedURL.Path
}

// mapStatus converts OxaPay status to internal WithdrawalStatus
func mapStatus(oxaStatus string) payment.WithdrawalStatus {
	switch oxaStatus {
	case "pending":
		return payment.StatusPending
	case "processing":
		return payment.StatusProcessing
	case "completed", "confirmed", "success":
		return payment.StatusCompleted
	case "failed", "error", "rejected":
		return payment.StatusFailed
	default:
		return payment.StatusPending
	}
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}
