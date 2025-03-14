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

// GetUsers отримує список користувачів з пагінацією
func (r *PostgresRepository) GetUsers(page, perPage int) ([]models.User, int64, error) {
	var users []models.User
	var totalCount int64

	// Отримуємо загальну кількість користувачів
	if err := r.db.Model(&models.User{}).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Отримуємо користувачів з пагінацією
	offset := (page - 1) * perPage
	if err := r.db.Offset(offset).Limit(perPage).Order("created_at desc").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, totalCount, nil
}

// GetUserByID отримує користувача за його ID
func (r *PostgresRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetWithdrawalByID отримує запит на виведення коштів за його ID
func (r *PostgresRepository) GetWithdrawalByID(id uint) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	if err := r.db.First(&withdrawal, id).Error; err != nil {
		return nil, err
	}
	return &withdrawal, nil
}
