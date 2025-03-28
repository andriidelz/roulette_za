package repository

import (
	"roulette/internal/models"

	"gorm.io/gorm"
)

// Реалізація методів для користувачів

func (r *PostgresRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
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

// Реализация методов для работы со страной пользователя
func (r *PostgresRepository) SetUserCountry(userID uint, country string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).
		Update("country", country).Error
}

func (r *PostgresRepository) GetUserCountry(userID uint) (string, error) {
	var user models.User
	err := r.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return "", err
	}
	return user.Country, nil
}

// SearchUsers поиск пользователей по запросу
func (r *PostgresRepository) SearchUsers(query string, page, perPage int) ([]models.User, int64, error) {
	var users []models.User
	var totalCount int64

	// Строим запрос с фильтрацией
	searchQuery := "%" + query + "%"
	baseQuery := r.db.Model(&models.User{}).Where(
		"username LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR CAST(telegram_id AS TEXT) LIKE ?",
		searchQuery, searchQuery, searchQuery, searchQuery,
	)

	// Получаем общее количество пользователей, соответствующих критериям поиска
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Получаем пользователей с пагинацией
	offset := (page - 1) * perPage
	if err := baseQuery.Offset(offset).Limit(perPage).Order("id desc").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, totalCount, nil
}
