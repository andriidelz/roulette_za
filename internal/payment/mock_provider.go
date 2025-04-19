package payment

import (
	"time"
)

// MockProvider реализует интерфейс Provider для тестирования
type MockProvider struct{}

// NewMockProvider создает новый мок-провайдер для тестирования
func NewMockProvider() Provider {
	return &MockProvider{}
}

// CreateWithdrawal имитирует создание запроса на вывод средств
func (p *MockProvider) CreateWithdrawal(userID uint, amount float64, currency string, address string) (*Withdrawal, error) {
	return &Withdrawal{
		ID:              "mock-id-" + time.Now().Format("20060102150405"),
		UserID:          userID,
		Amount:          amount,
		Currency:        currency,
		Address:         address,
		Status:          StatusPending,
		TransactionHash: "",
		Description:     "Mock withdrawal",
		ProviderName:    "mock",
		ProviderData:    nil,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

// GetWithdrawalStatus имитирует получение статуса вывода
func (p *MockProvider) GetWithdrawalStatus(withdrawalID string) (WithdrawalStatus, error) {
	return StatusProcessing, nil
}
