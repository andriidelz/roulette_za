package repository

import (
	"roulette/internal/models"
	"time"

	"gorm.io/gorm"
)

// Реалізація методів для користувачів

func (r *PostgresRepository) CreateUser(user *models.User) error {
	tx := r.db.Begin()

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Створюємо статистику для користувача
	stats := &models.UserStats{
		UserID:    user.ID,
		LastReset: time.Now(),
	}

	if err := tx.Create(stats).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *PostgresRepository) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User
	err := r.db.Where("telegram_id = ?", telegramID).First(&user).Error

	// Перевіряємо, чи помилка є "record not found"
	if err == gorm.ErrRecordNotFound {
		// Повертаємо помилку, але не викликаємо panic
		return nil, err
	} else if err != nil {
		// Інша помилка
		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *PostgresRepository) GetUserCount() (int64, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresRepository) GetUserStats(userID uint) (*models.UserStats, error) {
	var stats models.UserStats
	err := r.db.Where("user_id = ?", userID).First(&stats).Error

	// Якщо статистики немає, створюємо нову
	if err == gorm.ErrRecordNotFound {
		stats = models.UserStats{
			UserID:    userID,
			LastReset: time.Now(),
		}
		if err := r.db.Create(&stats).Error; err != nil {
			return nil, err
		}
		return &stats, nil
	} else if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *PostgresRepository) UpdateUserStats(stats *models.UserStats) error {
	return r.db.Save(stats).Error
}

func (r *PostgresRepository) ResetDailyBets() error {
	return r.db.Model(&models.User{}).Update("today_bets", 0).Error
}
