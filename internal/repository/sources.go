package repository

import (
	"roulette/internal/models"
	"time"

	"gorm.io/gorm"
)

// SetSourceKey сохранить источник
func (r *PostgresRepository) SetSourceKey(key string, name string) error {
	var res models.SourceKey
	err := r.db.Where("key = ?", key).First(&res).Error

	if err == gorm.ErrRecordNotFound {
		// Создаем новый источник
		res = models.SourceKey{
			Key:       key,
			Name:      name,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return r.db.Create(&res).Error
	} else if err != nil {
		return err
	}

	// Обновляем источник
	res.Name = name
	res.UpdatedAt = time.Now()
	return r.db.Save(&res).Error
}

// GetAllSourceKeys получает все источники
func (r *PostgresRepository) GetAllSourceKeys() ([]models.SourceKey, error) {
	var sources []models.SourceKey
	if err := r.db.Order("key").Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

// CheckSourceKeyExists проверяет существование источника по ключу
func (r *PostgresRepository) CheckSourceKeyExists(key string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.SourceKey{}).Where("key = ?", key).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
