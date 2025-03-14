package repository

import "roulette/internal/models"

// Реалізація методів для налаштувань

func (r *PostgresRepository) GetSetting(key string) (*models.Setting, error) {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *PostgresRepository) UpdateSetting(key string, value string) error {
	return r.db.Model(&models.Setting{}).Where("key = ?", key).
		Update("value", value).Error
}
