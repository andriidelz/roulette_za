package oxapay

import (
	"fmt"
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
	// Создаем запрос на вывод средств через OxaPay
	payoutRequest := oxapayclient.PayoutRequest{
		Currency:    currency,
		Amount:      amount,
		Address:     address,
		CallbackURL: p.callbackURL,
		Description: fmt.Sprintf("Withdrawal for user %d", userID),
		UserID:      strconv.FormatUint(uint64(userID), 10), // Преобразуем ID пользователя в строку
		IsSandbox:   getEnvBool("OXAPAY_USE_SANDBOX", false),
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
		IsSandbox:       getEnvBool("OXAPAY_USE_SANDBOX", false),
		CreatedAt:       payout.CreatedAt,
		UpdatedAt:       payout.UpdatedAt,
	}, nil
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
	// Этот метод может быть пустым или содержать дополнительную логику настройки вебхуков
	return nil
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
