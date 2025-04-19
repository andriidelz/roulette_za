package service

import (
	"fmt"
	"roulette/internal/models"
	"roulette/internal/payment"
	"roulette/internal/repository"
	"time"
)

// PaymentService handles payment-related operations
type PaymentService struct {
	repo             repository.Repository
	paymentProviders *payment.Factory
	defaultProvider  string
}

// NewPaymentService creates a new payment service
func NewPaymentService(repo repository.Repository, providers *payment.Factory, defaultProvider string) *PaymentService {
	return &PaymentService{
		repo:             repo,
		paymentProviders: providers,
		defaultProvider:  defaultProvider,
	}
}

// RequestWithdrawal processes a withdrawal request
func (s *PaymentService) RequestWithdrawal(userID uint, amount float64, walletAddress string) (*models.Withdrawal, error) {
	// Get user to check balance
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	// Check if user has enough balance
	if user.Balance < amount {
		return nil, fmt.Errorf("insufficient balance: %.2f < %.2f", user.Balance, amount)
	}

	// Get the payment provider
	provider, err := s.paymentProviders.GetProvider(s.defaultProvider)
	if err != nil {
		return nil, fmt.Errorf("error getting payment provider: %w", err)
	}

	// Create the withdrawal request
	withdrawal, err := provider.CreateWithdrawal(userID, amount, "USDT", walletAddress)
	if err != nil {
		return nil, fmt.Errorf("error creating withdrawal: %w", err)
	}

	// Subtract amount from user's balance
	user.Balance -= amount
	if err := s.repo.UpdateUser(user); err != nil {
		// If we fail to update user balance, log the error but continue
		// We'll need to handle this in a transaction in a real-world scenario
		fmt.Printf("Error updating user balance: %v\n", err)
	}

	// Create withdrawal record in our database
	withdrawalModel := &models.Withdrawal{
		UserID:          userID,
		Amount:          amount,
		Wallet:          walletAddress,
		Status:          string(withdrawal.Status),
		ProviderName:    withdrawal.ProviderName,
		ProviderID:      withdrawal.ID,
		TransactionHash: withdrawal.TransactionHash,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateWithdrawal(withdrawalModel); err != nil {
		return nil, fmt.Errorf("error creating withdrawal record: %w", err)
	}

	return withdrawalModel, nil
}

// GetWithdrawalStatus gets the current status of a withdrawal
func (s *PaymentService) GetWithdrawalStatus(withdrawalID uint) (string, error) {
	// Get withdrawal from database
	withdrawal, err := s.repo.GetWithdrawalByID(withdrawalID)
	if err != nil {
		return "", fmt.Errorf("error getting withdrawal: %w", err)
	}

	// If we don't have provider information, just return current status
	if withdrawal.ProviderName == "" || withdrawal.ProviderID == "" {
		return withdrawal.Status, nil
	}

	// Get provider
	provider, err := s.paymentProviders.GetProvider(withdrawal.ProviderName)
	if err != nil {
		return withdrawal.Status, fmt.Errorf("error getting payment provider: %w", err)
	}

	// Check status with provider
	status, err := provider.GetWithdrawalStatus(withdrawal.ProviderID)
	if err != nil {
		return withdrawal.Status, fmt.Errorf("error getting withdrawal status: %w", err)
	}

	// Update status if changed
	if string(status) != withdrawal.Status {
		withdrawal.Status = string(status)
		withdrawal.UpdatedAt = time.Now()

		if err := s.repo.UpdateWithdrawalStatus(withdrawalID, withdrawal.Status); err != nil {
			return string(status), fmt.Errorf("error updating withdrawal status: %w", err)
		}
	}

	return string(status), nil
}

// ProcessWebhookUpdate processes a webhook update for a withdrawal
func (s *PaymentService) ProcessWebhookUpdate(providerName string, withdrawalID string, status payment.WithdrawalStatus, transactionHash string) error {
	// Find withdrawal in our database by provider ID
	withdrawal, err := s.repo.GetWithdrawalByProviderID(providerName, withdrawalID)
	if err != nil {
		return fmt.Errorf("error finding withdrawal: %w", err)
	}

	// Update status
	withdrawal.Status = string(status)
	withdrawal.TransactionHash = transactionHash
	withdrawal.UpdatedAt = time.Now()

	// Save updated withdrawal
	if err := s.repo.UpdateWithdrawal(withdrawal); err != nil {
		return fmt.Errorf("error updating withdrawal: %w", err)
	}

	return nil
}

// ApproveWithdrawal approves a withdrawal request and initiates the actual withdrawal
func (s *PaymentService) ApproveWithdrawal(withdrawalID uint) error {
	// Get the withdrawal request
	withdrawal, err := s.repo.GetWithdrawalByID(withdrawalID)
	if err != nil {
		return fmt.Errorf("error getting withdrawal: %w", err)
	}

	// Check if already processed
	if withdrawal.Status != "pending" {
		return fmt.Errorf("withdrawal is not in pending status: %s", withdrawal.Status)
	}

	// Get payment provider
	provider, err := s.paymentProviders.GetProvider(s.defaultProvider)
	if err != nil {
		// Try getting the default provider
		provider, err = s.paymentProviders.GetDefaultProvider(s.defaultProvider)
		if err != nil {
			return fmt.Errorf("failed to get any payment provider: %w", err)
		}
	}

	// Create withdrawal request through the provider
	result, err := provider.CreateWithdrawal(
		withdrawal.UserID,
		withdrawal.Amount,
		"USDT", // Currency should be configurable
		withdrawal.Wallet,
	)
	if err != nil {
		return fmt.Errorf("error creating withdrawal: %w", err)
	}

	// Update withdrawal details
	withdrawal.Status = string(result.Status)
	withdrawal.ProviderName = s.defaultProvider
	withdrawal.ProviderID = result.ID
	withdrawal.TransactionHash = result.TransactionHash
	withdrawal.UpdatedAt = time.Now()

	// Save changes
	if err := s.repo.UpdateWithdrawal(withdrawal); err != nil {
		return fmt.Errorf("error updating withdrawal: %w", err)
	}

	return nil
}

// CheckPendingWithdrawals periodically checks status of pending withdrawals via API
func (s *PaymentService) CheckPendingWithdrawals() error {
	withdrawals, err := s.repo.GetPendingWithdrawals()
	if err != nil {
		return fmt.Errorf("error getting pending withdrawals: %w", err)
	}

	for _, withdrawal := range withdrawals {
		// Skip if no provider info
		if withdrawal.ProviderName == "" || withdrawal.ProviderID == "" {
			continue
		}

		provider, err := s.paymentProviders.GetProvider(withdrawal.ProviderName)
		if err != nil {
			continue // Skip this one and move to next
		}

		status, err := provider.GetWithdrawalStatus(withdrawal.ProviderID)
		if err != nil {
			continue // Skip this one and move to next
		}

		// Update if status changed
		if string(status) != withdrawal.Status {
			withdrawal.Status = string(status)
			withdrawal.UpdatedAt = time.Now()
			s.repo.UpdateWithdrawal(&withdrawal)
		}
	}

	return nil
}
