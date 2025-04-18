package payment

import (
	"time"
)

// Provider interface defines methods that any payment provider must implement
type Provider interface {
	// CreateWithdrawal creates a withdrawal request
	CreateWithdrawal(userID uint, amount float64, currency string, address string) (*Withdrawal, error)

	// GetWithdrawalStatus returns the current status of a withdrawal
	GetWithdrawalStatus(withdrawalID string) (WithdrawalStatus, error)

	// SetupWebhooks configures the webhook endpoints for the provider
	SetupWebhooks() error
}

// WithdrawalStatus represents possible status values for a withdrawal
type WithdrawalStatus string

const (
	StatusPending    WithdrawalStatus = "pending"
	StatusProcessing WithdrawalStatus = "processing"
	StatusCompleted  WithdrawalStatus = "completed"
	StatusFailed     WithdrawalStatus = "failed"
)

// Withdrawal represents a withdrawal transaction
type Withdrawal struct {
	ID              string
	UserID          uint
	Amount          float64
	Currency        string
	Address         string
	Status          WithdrawalStatus
	TransactionHash string
	Description     string
	ProviderName    string
	ProviderData    interface{} // Provider-specific data
	IsSandbox       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
