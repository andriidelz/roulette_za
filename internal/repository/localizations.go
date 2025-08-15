package repository

import (
	"roulette/internal/models"

	"gorm.io/gorm"
)

// GetLocalization получает локализацию по ключу и языку
func (r *PostgresRepository) GetLocalization(key string, language string) (models.Localization, error) {
	var loc models.Localization
	err := r.db.Where("key = ? AND language = ?", key, language).First(&loc).Error
	if err != nil {
		// Якщо локалізація не знайдена для вказаної мови, спробуємо знайти англійську
		if language != "en" {
			return r.GetLocalization(key, "en")
		}
		return models.Localization{}, err
	}
	return loc, nil
}

// SetLocalization устанавливает локализацию по ключу и языку
func (r *PostgresRepository) SetLocalization(value models.Localization) error {
	var loc models.Localization
	err := r.db.Where("key = ? AND language = ?", value.Key, value.Language).First(&loc).Error

	if err == gorm.ErrRecordNotFound {
		// Створюємо нову локалізацію
		return r.db.Create(&value).Error
	} else if err != nil {
		return err
	}

	// Оновлюємо існуючу локалізацію
	loc.Value = value.Value
	loc.Image = value.Image
	loc.Video = value.Video
	return r.db.Save(&loc).Error
}

// GetAllLocalizationsForLanguage получает все локализации для указанного языка
func (r *PostgresRepository) GetAllLocalizationsForLanguage(language string) ([]models.Localization, error) {
	var localizations []models.Localization
	if err := r.db.Where("language = ?", language).Order("key").Find(&localizations).Error; err != nil {
		return nil, err
	}
	return localizations, nil
}

// GetAllLocalizationsByKey получает все локализации по ключу для всех языков
func (r *PostgresRepository) GetAllLocalizationsByKey(key string) ([]models.Localization, error) {
	var localizations []models.Localization
	err := r.db.Where("key = ?", key).Find(&localizations).Error
	return localizations, err
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
