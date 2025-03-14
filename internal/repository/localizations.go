package repository

import (
	"roulette/internal/models"

	"gorm.io/gorm"
)

// GetAllLocalizationsForLanguage получает все локализации для указанного языка
func (r *PostgresRepository) GetAllLocalizationsForLanguage(language string) ([]models.Localization, error) {
	var localizations []models.Localization
	if err := r.db.Where("language = ?", language).Order("key").Find(&localizations).Error; err != nil {
		return nil, err
	}
	return localizations, nil
}

// DeleteLocalization удаляет локализацию для всех языков по ключу
func (r *PostgresRepository) DeleteLocalization(key string) error {
	return r.db.Where("key = ?", key).Delete(&models.Localization{}).Error
}

// GetLocalizationCount возвращает количество локализаций для указанного языка
func (r *PostgresRepository) GetLocalizationCount(language string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Localization{}).Where("language = ?", language).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CheckLocalizationExists проверяет существование локализации по ключу
func (r *PostgresRepository) CheckLocalizationExists(key string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Localization{}).Where("key = ?", key).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ImportLocalizations импортирует или обновляет набор локализаций
func (r *PostgresRepository) ImportLocalizations(localizations []models.Localization) error {
	tx := r.db.Begin()

	for _, loc := range localizations {
		// Проверяем, существует ли такая локализация
		var existing models.Localization
		err := tx.Where("key = ? AND language = ?", loc.Key, loc.Language).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// Если не существует, создаем новую
			if err := tx.Create(&loc).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else if err != nil {
			// Если произошла другая ошибка
			tx.Rollback()
			return err
		} else {
			// Если существует, обновляем значение
			existing.Value = loc.Value
			if err := tx.Save(&existing).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}
