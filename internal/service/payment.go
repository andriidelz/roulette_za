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
	log.Printf("[PaymentService] Initialized with default provider: %s", defaultProvider)
	return &PaymentService{
		repo:             repo,
		paymentProviders: providers,
		defaultProvider:  defaultProvider,
	}
}

// RequestWithdrawal processes a withdrawal request
func (s *PaymentService) RequestWithdrawal(userID uint, amount float64, walletAddress string) (*models.Withdrawal, error) {
	log.Printf("[PaymentService] Processing withdrawal request: userID=%d, amount=%.2f, wallet=%s",
		userID, amount, walletAddress)

	// Get user to check balance
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		log.Printf("[PaymentService] Error getting user %d: %v", userID, err)
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	// Check if user has enough balance
	if user.Balance < amount {
		log.Printf("[PaymentService] Insufficient balance for user %d: %.2f < %.2f",
			userID, user.Balance, amount)
		return nil, fmt.Errorf("insufficient balance: %.2f < %.2f", user.Balance, amount)
	}

	// Get the payment provider
	provider, err := s.paymentProviders.GetProvider(s.defaultProvider)
	if err != nil {
		log.Printf("[PaymentService] Error getting payment provider '%s': %v", s.defaultProvider, err)
		return nil, fmt.Errorf("error getting payment provider: %w", err)
	}

	// Create the withdrawal request
	log.Printf("[PaymentService] Creating withdrawal via provider '%s'", s.defaultProvider)
	withdrawal, err := provider.CreateWithdrawal(userID, amount, "USDT", walletAddress)
	if err != nil {
		log.Printf("[PaymentService] Error creating withdrawal: %v", err)
		return nil, fmt.Errorf("error creating withdrawal: %w", err)
	}

	log.Printf("[PaymentService] Withdrawal created successfully: ID=%s, Status=%s",
		withdrawal.ID, withdrawal.Status)

	// Subtract amount from user's balance
	user.Balance -= amount
	if err := s.repo.UpdateUser(user); err != nil {
		// If we fail to update user balance, log the error but continue
		log.Printf("[PaymentService] Error updating user balance: %v", err)
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
		log.Printf("[PaymentService] Error creating withdrawal record in database: %v", err)
		return nil, fmt.Errorf("error creating withdrawal record: %w", err)
	}

	log.Printf("[PaymentService] Withdrawal record saved in database: ID=%d", withdrawalModel.ID)
	return withdrawalModel, nil
}

// GetWithdrawalStatus gets the current status of a withdrawal
func (s *PaymentService) GetWithdrawalStatus(withdrawalID uint) (string, error) {
	log.Printf("[PaymentService] Getting status for withdrawal ID=%d", withdrawalID)

	// Get withdrawal from database
	withdrawal, err := s.repo.GetWithdrawalByID(withdrawalID)
	if err != nil {
		log.Printf("[PaymentService] Error getting withdrawal %d: %v", withdrawalID, err)
		return "", fmt.Errorf("error getting withdrawal: %w", err)
	}

	// If we don't have provider information, just return current status
	if withdrawal.ProviderName == "" || withdrawal.ProviderID == "" {
		log.Printf("[PaymentService] Withdrawal %d has no provider info, returning current status: %s",
			withdrawalID, withdrawal.Status)
		return withdrawal.Status, nil
	}

	// Get provider
	log.Printf("[PaymentService] Getting provider '%s' for status check", withdrawal.ProviderName)
	provider, err := s.paymentProviders.GetProvider(withdrawal.ProviderName)
	if err != nil {
		log.Printf("[PaymentService] Error getting provider '%s': %v", withdrawal.ProviderName, err)
		return withdrawal.Status, fmt.Errorf("error getting payment provider: %w", err)
	}

	// Check status with provider
	log.Printf("[PaymentService] Checking status with provider for external ID=%s", withdrawal.ProviderID)
	status, txHash, err := provider.GetWithdrawalStatus(withdrawal.ProviderID)
	if err != nil {
		log.Printf("[PaymentService] Error getting withdrawal status from provider: %v", err)
		return withdrawal.Status, fmt.Errorf("error getting withdrawal status: %w", err)
	}

	// Update status and tx_hash if changed
	statusChanged := string(status) != withdrawal.Status
	txHashChanged := txHash != "" && txHash != withdrawal.TransactionHash

	if statusChanged || txHashChanged {
		log.Printf("[PaymentService] Updating withdrawal %d: status %s->%s, tx_hash %s->%s",
			withdrawalID, withdrawal.Status, status, withdrawal.TransactionHash, txHash)

		// Обновляем поля
		withdrawal.Status = string(status)
		withdrawal.UpdatedAt = time.Now()

		// Обновляем transaction_hash только если он не пустой
		if txHash != "" {
			withdrawal.TransactionHash = txHash
		}

		if err := s.repo.UpdateWithdrawal(withdrawal); err != nil {
			log.Printf("[PaymentService] Error updating withdrawal: %v", err)
			return string(status), fmt.Errorf("error updating withdrawal: %w", err)
		}
	}

	return string(status), nil
}

// ProcessWebhookUpdate processes a webhook update for a withdrawal
func (s *PaymentService) ProcessWebhookUpdate(providerName string, withdrawalID string, status payment.WithdrawalStatus, transactionHash string) error {
	log.Printf("[PaymentService] Processing webhook update: provider=%s, id=%s, status=%s",
		providerName, withdrawalID, status)

	// Find withdrawal in our database by provider ID
	withdrawal, err := s.repo.GetWithdrawalByProviderID(providerName, withdrawalID)
	if err != nil {
		log.Printf("[PaymentService] Error finding withdrawal by provider ID: %v", err)
		return fmt.Errorf("error finding withdrawal: %w", err)
	}

	// Update status
	log.Printf("[PaymentService] Updating withdrawal %d status: %s -> %s, hash: %s",
		withdrawal.ID, withdrawal.Status, status, transactionHash)
	withdrawal.Status = string(status)
	withdrawal.TransactionHash = transactionHash
	withdrawal.UpdatedAt = time.Now()

	// Save updated withdrawal
	if err := s.repo.UpdateWithdrawal(withdrawal); err != nil {
		log.Printf("[PaymentService] Error updating withdrawal: %v", err)
		return fmt.Errorf("error updating withdrawal: %w", err)
	}

	log.Printf("[PaymentService] Withdrawal %d successfully updated", withdrawal.ID)
	return nil
}

// ApproveWithdrawal approves a withdrawal request and initiates the actual withdrawal
func (s *PaymentService) ApproveWithdrawal(withdrawalID uint) error {
	log.Printf("[PaymentService] Approving withdrawal ID=%d", withdrawalID)

	// Get the withdrawal request
	withdrawal, err := s.repo.GetWithdrawalByID(withdrawalID)
	if err != nil {
		log.Printf("[PaymentService] Error getting withdrawal %d: %v", withdrawalID, err)
		return fmt.Errorf("error getting withdrawal: %w", err)
	}

	// Check if already processed
	if withdrawal.Status != "pending" {
		log.Printf("[PaymentService] Withdrawal %d is not in pending status: %s",
			withdrawalID, withdrawal.Status)
		return fmt.Errorf("withdrawal is not in pending status: %s", withdrawal.Status)
	}

	log.Printf("[PaymentService] Getting provider '%s' for withdrawal", s.defaultProvider)

	// Get payment provider
	provider, err := s.paymentProviders.GetProvider(s.defaultProvider)
	if err != nil {
		log.Printf("[PaymentService] Error getting primary provider '%s': %v", s.defaultProvider, err)

		// Try getting the default provider
		log.Printf("[PaymentService] Attempting to get any available provider")
		provider, err = s.paymentProviders.GetDefaultProvider(s.defaultProvider)
		if err != nil {
			log.Printf("[PaymentService] Failed to get any payment provider: %v", err)
			return fmt.Errorf("failed to get any payment provider: %w", err)
		}
		log.Printf("[PaymentService] Successfully got alternative provider")
	}

	// Create withdrawal request through the provider
	log.Printf("[PaymentService] Creating withdrawal via provider for user %d, amount %.2f",
		withdrawal.UserID, withdrawal.Amount)
	result, err := provider.CreateWithdrawal(
		withdrawal.UserID,
		withdrawal.Amount,
		"USDT", // Currency should be configurable
		withdrawal.Wallet,
	)
	if err != nil {
		log.Printf("[PaymentService] Error creating withdrawal via provider: %v", err)
		return fmt.Errorf("error creating withdrawal: %w", err)
	}

	log.Printf("[PaymentService] Provider returned: ID=%s, Status=%s", result.ID, result.Status)

	// Update withdrawal details
	withdrawal.Status = string(result.Status)
	withdrawal.ProviderName = s.defaultProvider
	withdrawal.ProviderID = result.ID
	withdrawal.TransactionHash = result.TransactionHash
	withdrawal.UpdatedAt = time.Now()

	// Save changes
	if err := s.repo.UpdateWithdrawal(withdrawal); err != nil {
		log.Printf("[PaymentService] Error updating withdrawal in database: %v", err)
		return fmt.Errorf("error updating withdrawal: %w", err)
	}

	log.Printf("[PaymentService] Withdrawal %d approved and updated successfully", withdrawalID)
	return nil
}

// CheckPendingWithdrawals periodically checks status of pending withdrawals via API
func (s *PaymentService) CheckPendingWithdrawals() error {
	log.Printf("[PaymentService] Starting periodic check of processing withdrawals")

	withdrawals, err := s.repo.GetProcessingWithdrawals()
	if err != nil {
		log.Printf("[PaymentService] Error getting processing withdrawals: %v", err)
		return fmt.Errorf("error getting processing withdrawals: %w", err)
	}

	log.Printf("[PaymentService] Found %d processing withdrawals to check", len(withdrawals))

	updatedCount := 0
	for _, withdrawal := range withdrawals {
		// Skip if no provider info
		if withdrawal.ProviderName == "" || withdrawal.ProviderID == "" {
			log.Printf("[PaymentService] Withdrawal %d has no provider info, skipping", withdrawal.ID)
			continue
		}

		log.Printf("[PaymentService] Checking withdrawal %d (provider=%s, id=%s)",
			withdrawal.ID, withdrawal.ProviderName, withdrawal.ProviderID)

		provider, err := s.paymentProviders.GetProvider(withdrawal.ProviderName)
		if err != nil {
			log.Printf("[PaymentService] Error getting provider '%s': %v", withdrawal.ProviderName, err)
			continue // Skip this one and move to next
		}

		status, txHash, err := provider.GetWithdrawalStatus(withdrawal.ProviderID)
		if err != nil {
			log.Printf("[PaymentService] Error getting status from provider: %v", err)
			continue // Skip this one and move to next
		}

		log.Printf("[PaymentService] Got status=%s, tx_hash=%s for withdrawal %d",
			status, txHash, withdrawal.ID)

		// Проверяем, изменился ли статус или появился transaction_hash
		statusChanged := string(status) != withdrawal.Status
		txHashChanged := txHash != "" && txHash != withdrawal.TransactionHash

		if statusChanged || txHashChanged {
			log.Printf("[PaymentService] Updating withdrawal %d: status %s->%s, tx_hash %s->%s",
				withdrawal.ID, withdrawal.Status, status, withdrawal.TransactionHash, txHash)

			withdrawal.Status = string(status)
			withdrawal.UpdatedAt = time.Now()

			if txHash != "" {
				withdrawal.TransactionHash = txHash
			}

			if err := s.repo.UpdateWithdrawal(&withdrawal); err != nil {
				log.Printf("[PaymentService] Error updating withdrawal: %v", err)
				continue
			}

			updatedCount++
		} else {
			log.Printf("[PaymentService] No changes for withdrawal %d: status=%s, tx_hash=%s",
				withdrawal.ID, withdrawal.Status, withdrawal.TransactionHash)
		}
	}

	log.Printf("[PaymentService] Completed checking withdrawals. Updated %d of %d withdrawals",
		updatedCount, len(withdrawals))

	return nil
}
