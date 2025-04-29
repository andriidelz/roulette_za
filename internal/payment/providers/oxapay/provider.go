package oxapay

import (
	"fmt"
	"log"
	"roulette/internal/payment"
	oxapayclient "roulette/pkg/oxapay"

	"gorm.io/gorm"
)

// Provider implements the payment.Provider interface for OxaPay
type Provider struct {
	client *oxapayclient.Client
}

// NewProvider creates a new OxaPay provider
func NewProvider(apiKey string, db interface{}) *Provider {
	log.Printf("[OxaPayProvider] Initializing provider")

	// Проверяем, что db имеет нужный тип
	var gormDB *gorm.DB
	var ok bool
	if db != nil {
		gormDB, ok = db.(*gorm.DB)
		if !ok {
			log.Printf("[OxaPayProvider] Warning: db must be of type *gorm.DB")
			panic("db must be of type *gorm.DB")
		}
	}

	// Создаем конфигурацию клиента OxaPay
	config := oxapayclient.Config{
		APIKey: apiKey,
		DB:     gormDB,
	}

	// Создаем клиент OxaPay
	client := oxapayclient.NewClient(config)
	log.Printf("[OxaPayProvider] Provider initialized successfully")

	return &Provider{
		client: client,
	}
}

// CreateWithdrawal implements payment.Provider
func (p *Provider) CreateWithdrawal(userID uint, amount float64, currency string, address string) (*payment.Withdrawal, error) {
	log.Printf("[OxaPayProvider] Creating withdrawal: userID=%d, amount=%.2f, currency=%s, address=%s",
		userID, amount, currency, address)

	// Определяем сеть на основе валюты
	network := p.getNetworkForCurrency(currency)
	log.Printf("[OxaPayProvider] Using network: %s for currency: %s", network, currency)

	// Создаем запрос на вывод средств через OxaPay
	payoutRequest := oxapayclient.PayoutRequest{
		Currency:    currency,
		Amount:      amount,
		Address:     address,
		Network:     network,
		Description: fmt.Sprintf("Withdrawal for user %d", userID),
		UserID:      fmt.Sprintf("%d", userID),
	}

	// Отправляем запрос через клиент OxaPay
	payout, err := p.client.CreatePayout(payoutRequest)
	if err != nil {
		log.Printf("[OxaPayProvider] Withdrawal creation failed: %v", err)
		return nil, fmt.Errorf("oxapay withdrawal creation failed: %w", err)
	}

	log.Printf("[OxaPayProvider] Withdrawal created successfully: ID=%s, Status=%s",
		payout.ID, payout.Status)

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
		ProviderName:    "oxapay", // Сохраняем название провайдера
		ProviderData:    payout,   // Сохраняем оригинальный ответ от провайдера
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
	log.Printf("[OxaPayProvider] Getting withdrawal status for ID: %s", withdrawalID)

	// Получаем информацию о выводе средств из OxaPay
	payout, err := p.client.GetPayout(withdrawalID)
	if err != nil {
		log.Printf("[OxaPayProvider] Failed to get withdrawal status: %v", err)
		return payment.StatusFailed, fmt.Errorf("failed to get withdrawal status: %w", err)
	}

	log.Printf("[OxaPayProvider] Got status: %s for withdrawal: %s", payout.Status, withdrawalID)

	// Преобразуем статус OxaPay в наш внутренний статус
	return mapStatus(payout.Status), nil
}

// SetupWebhooks implements payment.Provider
func (p *Provider) SetupWebhooks() error {
	// OxaPay webhooks больше не используются, так как мы используем периодический опрос API
	log.Printf("[OxaPayProvider] Webhooks are not used anymore, using API polling instead")
	return nil
}

// mapStatus converts OxaPay status to internal WithdrawalStatus
// https://docs.oxapay.com/api-reference/payout/payout-status-table
func mapStatus(oxaStatus string) payment.WithdrawalStatus {
	log.Printf("[OxaPayProvider] Mapping status: %s", oxaStatus)

	switch oxaStatus {
	case "processing":
		// Запрос отправлен и обрабатывается
		return payment.StatusProcessing
	case "pending":
		// Запрос обработан и находится в очереди на оплату
		// Важно: это НЕ то же самое, что "pending" в нашей системе (ожидание подтверждения)
		return payment.StatusProcessing
	case "confirming":
		// Транзакция создана и ожидает подтверждения в блокчейне
		return payment.StatusProcessing
	case "confirmed":
		// Транзакция успешно оплачена
		return payment.StatusCompleted
	case "canceled":
		// Запрос на выплату был отменен
		return payment.StatusFailed
	case "rejected":
		// Запрос был отклонен по каким-либо причинам
		return payment.StatusFailed
	default:
		log.Printf("[OxaPayProvider] Unknown status: %s, mapping to processing", oxaStatus)
		return payment.StatusProcessing
	}
}
