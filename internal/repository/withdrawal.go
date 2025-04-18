package repository

import (
	"fmt"
	"roulette/internal/models"
)

// Реалізація методів для виведення коштів

func (r *PostgresRepository) CreateWithdrawal(withdrawal *models.Withdrawal) error {
	return r.db.Create(withdrawal).Error
}

func (r *PostgresRepository) GetPendingWithdrawals() ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	if err := r.db.Where("status = ?", "pending").
		Preload("User").
		Find(&withdrawals).Error; err != nil {
		return nil, err
	}
	return withdrawals, nil
}

func (r *PostgresRepository) UpdateWithdrawalStatus(id uint, status string) error {
	return r.db.Model(&models.Withdrawal{}).Where("id = ?", id).
		Update("status", status).Error
}

// GetWithdrawalByID отримує запит на виведення коштів за його ID
func (r *PostgresRepository) GetWithdrawalByID(id uint) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	if err := r.db.First(&withdrawal, id).Error; err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

// GetUserWithdrawals получает историю выводов для конкретного пользователя
func (r *PostgresRepository) GetUserWithdrawals(userID uint, limit int) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	query := r.db.Where("user_id = ?", userID).Order("created_at desc")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&withdrawals).Error; err != nil {
		return nil, err
	}

	return withdrawals, nil
}

// GetWithdrawalByProviderID получает запись о выводе средств по идентификатору провайдера
func (r *PostgresRepository) GetWithdrawalByProviderID(providerName, providerID string) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	if err := r.db.Where("provider_name = ? AND provider_id = ?", providerName, providerID).First(&withdrawal).Error; err != nil {
		return nil, fmt.Errorf("failed to find withdrawal: %w", err)
	}
	return &withdrawal, nil
}

// UpdateWithdrawal обновляет всю запись о выводе средств
func (r *PostgresRepository) UpdateWithdrawal(withdrawal *models.Withdrawal) error {
	return r.db.Save(withdrawal).Error
}
