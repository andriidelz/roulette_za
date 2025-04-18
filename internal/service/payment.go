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

	// Get provider
	provider, err := s.paymentProviders.GetProvider(withdrawal.ProviderName)
	if err != nil {
		return "", fmt.Errorf("error getting payment provider: %w", err)
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

// ApproveWithdrawal метод сервиса для одобрения вывода средств
func (s *PaymentService) ApproveWithdrawal(withdrawalID uint) error {
	// Получаем запрос на вывод средств
	withdrawal, err := s.repo.GetWithdrawalByID(withdrawalID)
	if err != nil {
		return err
	}

	// Если запрос уже обработан, возвращаем ошибку
	if withdrawal.Status != "pending" {
		return fmt.Errorf("withdrawal is not in pending status: %s", withdrawal.Status)
	}

	// Получаем платежный провайдер
	provider, err := s.paymentProviders.GetDefaultProvider()
	if err != nil {
		return err
	}

	// Создаем запрос на вывод через провайдер
	result, err := provider.CreateWithdrawal(
		withdrawal.UserID,
		withdrawal.Amount,
		"USDT", // Валюта должна быть настраиваемой
		withdrawal.Wallet,
	)
	if err != nil {
		return err
	}

	// Обновляем запрос на вывод
	withdrawal.Status = string(result.Status)
	withdrawal.ProviderName = result.ProviderName
	withdrawal.ProviderID = result.ID
	withdrawal.TransactionHash = result.TransactionHash

	// Сохраняем изменения
	return s.repo.UpdateWithdrawal(withdrawal)
}
