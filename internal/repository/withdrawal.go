package repository

import (
	"fmt"
	"roulette/internal/config"
	"roulette/internal/models"
	"time"

	"gorm.io/gorm"
)

// Реалізація методів для виведення коштів

// GetWithdrawalsStat отримує статистику по виплатам за період
func (r *PostgresRepository) GetWithdrawalsStat(dateFrom, dateTo string) ([]models.WithdrawalStat, error) {

	var stats []models.WithdrawalStat
	err := r.db.Where("created_at >= ? AND created_at <= ?", dateFrom, dateTo).Find(&stats).Error
	if err != nil {
		return stats, err
	}

	return stats, nil
}

// FindWithdrawalsStat отримує або створює запис в статистиці по виплатам
func (r *PostgresRepository) FindWithdrawalsStat(day string) (models.WithdrawalStat, error) {

	var data models.WithdrawalStat

	err := r.db.Where("day = ?", day).First(&data).Error

	if err == gorm.ErrRecordNotFound {
		// Створюємо новий запис
		data = models.WithdrawalStat{
			Day:       day,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := r.db.Create(&data).Error; err != nil {
			return data, err
		}
	} else if err != nil {
		return data, err
	}
	return data, nil
}

// RecalculateWithdrawalsStat - періодичне оновлення статистики
func (r *PostgresRepository) RecalculateWithdrawalsStat(id string) (models.WithdrawalStat, error) {

	data, err := r.FindWithdrawalsStat(id)
	if err != nil {
		return data, err
	}

	// Сума балансів
	var balance float64
	err = r.db.Model(&models.User{}).Where("status = ? ", config.UserStatusActive).Select("COALESCE(SUM(balance), 0)").Scan(&balance).Error
	if err != nil {
		return data, err
	}
	data.Balance = balance

	// Запрошено до виплати
	var withdrawal float64
	now := time.Now().UTC() // Час в UTC
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 0, 1)

	err = r.db.Model(&models.Withdrawal{}).Where("created_at >= ? AND created_at <= ?", from, to).Select(
		"COALESCE(SUM(amount), 0)").Scan(&withdrawal).Error
	if err != nil {
		return data, err
	}
	data.Withdrawal = withdrawal

	// Виплачено
	var payout float64
	err = r.db.Model(&models.Withdrawal{}).Where("status = ? AND created_at >= ? AND created_at <= ?", "completed", from, to).Select(
		"COALESCE(SUM(amount), 0)").Scan(&payout).Error
	if err != nil {
		return data, err
	}
	data.Payout = payout

	return data, r.UpdateWithdrawalsStat(&data)
}

// UpdateWithdrawalsStat обновляет всю запись о статистике
func (r *PostgresRepository) UpdateWithdrawalsStat(data *models.WithdrawalStat) error {
	data.UpdatedAt = time.Now()
	return r.db.Save(data).Error
}

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

// GetProcessingWithdrawals получает выводы со статусом "processing"
func (r *PostgresRepository) GetProcessingWithdrawals() ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	if err := r.db.Where("status = ?", "processing").
		Preload("User").
		Find(&withdrawals).Error; err != nil {
		return nil, err
	}
	return withdrawals, nil
}

// GetWithdrawalsHistory получает историю выводов (все, кроме статуса "pending")
func (r *PostgresRepository) GetWithdrawalsHistory(limit int) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	query := r.db.Where("status != ?", "pending").
		Preload("User").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&withdrawals).Error; err != nil {
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
