package service

import (
	"fmt"
	"log"
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
	log.Printf("[PaymentService] Approving withdrawal ID: %d", withdrawalID)
	log.Printf("[PaymentService] Default provider name: %s", s.defaultProvider)

	// Получаем запрос на вывод средств
	withdrawal, err := s.repo.GetWithdrawalByID(withdrawalID)
	if err != nil {
		log.Printf("[PaymentService] Error getting withdrawal %d: %v", withdrawalID, err)
		return err
	}

	// Если запрос уже обработан, возвращаем ошибку
	if withdrawal.Status != "pending" {
		log.Printf("[PaymentService] Withdrawal %d is not in pending status: %s", withdrawalID, withdrawal.Status)
		return fmt.Errorf("withdrawal is not in pending status: %s", withdrawal.Status)
	}

	log.Printf("[PaymentService] Before getting provider, defaultProvider = %s", s.defaultProvider)

	// ПОПРОБУЕМ ОБА СПОСОБА ПОЛУЧЕНИЯ ПРОВАЙДЕРА ДЛЯ ОТЛАДКИ

	// 1. Получаем провайдер напрямую по имени
	providerByName, errByName := s.paymentProviders.GetProvider("oxapay")
	if errByName != nil {
		log.Printf("[PaymentService] Error getting oxapay provider directly: %v", errByName)
	} else {
		log.Printf("[PaymentService] Successfully got oxapay provider: %T", providerByName)
	}

	// 2. Получаем дефолтный провайдер
	providerDefault, errDefault := s.paymentProviders.GetDefaultProvider()
	if errDefault != nil {
		log.Printf("[PaymentService] Error getting default provider: %v", errDefault)
	} else {
		log.Printf("[PaymentService] Successfully got default provider: %T", providerDefault)
	}

	// 3. Получаем mock провайдер для сравнения
	providerMock, errMock := s.paymentProviders.GetProvider("mock")
	if errMock != nil {
		log.Printf("[PaymentService] Error getting mock provider: %v", errMock)
	} else {
		log.Printf("[PaymentService] Successfully got mock provider: %T", providerMock)
	}

	// Выбираем провайдер для использования (первый успешный)
	var provider payment.Provider
	var providerName string

	if errByName == nil {
		provider = providerByName
		providerName = "oxapay"
		log.Printf("[PaymentService] Using oxapay provider")
	} else if errDefault == nil {
		provider = providerDefault
		providerName = s.defaultProvider
		log.Printf("[PaymentService] Using default provider: %s", providerName)
	} else if errMock == nil {
		provider = providerMock
		providerName = "mock"
		log.Printf("[PaymentService] Using mock provider as fallback")
	} else {
		log.Printf("[PaymentService] All provider getters failed, cannot proceed")
		return fmt.Errorf("failed to get any payment provider")
	}

	// Создаем запрос на вывод через выбранный провайдер
	log.Printf("[PaymentService] Creating withdrawal using provider: %s", providerName)

	result, err := provider.CreateWithdrawal(
		withdrawal.UserID,
		withdrawal.Amount,
		"USDT", // Валюта должна быть настраиваемой
		withdrawal.Wallet,
	)

	if err != nil {
		log.Printf("[PaymentService] Error creating withdrawal: %v", err)
		return err
	}

	// Проверяем, что возвращено из провайдера
	if result == nil {
		log.Printf("[PaymentService] Provider returned nil result")
		return fmt.Errorf("provider returned nil result")
	}

	log.Printf("[PaymentService] Provider result details: ID=%s, Status=%s, ProviderName=%s",
		result.ID, result.Status, result.ProviderName)

	// Обновляем запрос на вывод
	oldStatus := withdrawal.Status
	oldProviderName := withdrawal.ProviderName

	withdrawal.Status = string(result.Status)

	// ВАЖНО: Возможно проблема именно здесь - проверяем ProviderName и устанавливаем явно
	if result.ProviderName == "" {
		log.Printf("[PaymentService] Provider returned empty ProviderName, using %s", providerName)
		withdrawal.ProviderName = providerName
	} else {
		withdrawal.ProviderName = result.ProviderName
		log.Printf("[PaymentService] Using ProviderName from result: %s", result.ProviderName)
	}

	withdrawal.ProviderID = result.ID
	withdrawal.TransactionHash = result.TransactionHash

	log.Printf("[PaymentService] Updating withdrawal from Status=%s to Status=%s, from ProviderName=%s to ProviderName=%s",
		oldStatus, withdrawal.Status, oldProviderName, withdrawal.ProviderName)

	// Сохраняем изменения
	if err := s.repo.UpdateWithdrawal(withdrawal); err != nil {
		log.Printf("[PaymentService] Error updating withdrawal: %v", err)
		return err
	}

	log.Printf("[PaymentService] Withdrawal %d approved successfully", withdrawalID)
	return nil
}
