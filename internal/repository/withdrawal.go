package repository

import "roulette/internal/models"

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
