package repository

import (
	"roulette/internal/models"

	"gorm.io/gorm"
)

// GetAllSettings получает все настройки из базы данных
func (r *PostgresRepository) GetAllSettings() (map[string]*models.Setting, error) {
	var settings []models.Setting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}

	// Преобразуем слайс в карту для удобного доступа
	settingsMap := make(map[string]*models.Setting)
	for i := range settings {
		settingsMap[settings[i].Key] = &settings[i]
	}

	return settingsMap, nil
}

// CreateOrUpdateSetting создает или обновляет настройку
func (r *PostgresRepository) CreateOrUpdateSetting(key, value, defaultValue, description string) error {
	var setting models.Setting
	result := r.db.Where("key = ?", key).First(&setting)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Создаем новую настройку
			setting = models.Setting{
				Key:          key,
				Value:        value,
				DefaultValue: defaultValue,
				Description:  description,
			}
			return r.db.Create(&setting).Error
		}
		return result.Error
	}

	// Обновляем существующую настройку
	setting.Value = value
	setting.DefaultValue = defaultValue
	setting.Description = description
	return r.db.Save(&setting).Error
}

// GetSettingWithDefault получает настройку с значением по умолчанию
func (r *PostgresRepository) GetSettingWithDefault(key string) (*models.Setting, error) {
	var setting models.Setting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Возвращаем пустую настройку с ключом
			return &models.Setting{Key: key}, nil
		}
		return nil, err
	}
	return &setting, nil
}
