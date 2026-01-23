package service

import (
	"roulette/internal/models"
	"strconv"
	"time"
)

// GetAllSettings получает все настройки с дополнительной информацией
func (s *ServiceImpl) GetSettings() (map[string]string, error) {
	settings, err := s.repo.GetAllSettings()
	if err != nil {
		return nil, err
	}

	// Преобразуем в карту string -> string
	result := make(map[string]string)
	for key, setting := range settings {
		result[key] = setting.Value
	}

	// Добавляем настройки по умолчанию, если они отсутствуют
	defaultSettings := s.getDefaultSettings()
	for key, info := range defaultSettings {
		if _, exists := result[key]; !exists {
			result[key] = info.DefaultValue
		}
	}

	return result, nil
}

// GetSettingsWithInfo получает все настройки с типами и описаниями
func (s *ServiceImpl) GetSettingsWithInfo() (map[string]models.SettingInfo, error) {
	settings, err := s.repo.GetAllSettings()
	if err != nil {
		return nil, err
	}

	// Получаем информацию о настройках по умолчанию
	defaultSettings := s.getDefaultSettings()

	// Объединяем информацию из БД с информацией по умолчанию
	result := make(map[string]models.SettingInfo)
	for key, info := range defaultSettings {
		if setting, exists := settings[key]; exists {
			result[key] = models.SettingInfo{
				Key:          key,
				Value:        setting.Value,
				DefaultValue: info.DefaultValue,
				Description:  info.Description,
				Type:         info.Type,
			}
		} else {
			result[key] = info
		}
	}

	return result, nil
}

// getDefaultSettings возвращает информацию о всех доступных настройках
func (s *ServiceImpl) getDefaultSettings() map[string]models.SettingInfo {
	return map[string]models.SettingInfo{
		"daily_bets_limit": {
			Key:          "daily_bets_limit",
			DefaultValue: "2880",
			Description:  "Лимит ставок за день",
			Type:         "int",
		},
		"daily_bets_zero_limit": {
			Key:          "daily_bets_zero_limit",
			DefaultValue: "100",
			Description:  "Лимит ставок за день для возможности ставить на zero",
			Type:         "int",
		},
		"weekly_prize_amount": {
			Key:          "weekly_prize_amount",
			DefaultValue: "1000",
			Description:  "Сумма недельного призового фонда",
			Type:         "float",
		},
		"weekly_prize_top": {
			Key:          "weekly_prize_top",
			DefaultValue: "100",
			Description:  "Количество призовых мест в недельном рейтинге",
			Type:         "int",
		},
		"minimum_withdrawal": {
			Key:          "minimum_withdrawal",
			DefaultValue: "10",
			Description:  "Минимальная сумма для вывода средств",
			Type:         "float",
		},
		"prize_distribution_day": {
			Key:          "prize_distribution_day",
			DefaultValue: "1", // Понедельник
			Description:  "День недели для раздачи призов (1-7, где 1 - Понедельник)",
			Type:         "day",
		},
		"prize_distribution_time": {
			Key:          "prize_distribution_time",
			DefaultValue: "00:00",
			Description:  "Время раздачи призов (UTC+0)",
			Type:         "time",
		},
		// Налаштування капчі
		"captcha_bet_activity": {
			Key:          "captcha_bet_activity",
			DefaultValue: "9",
			Description:  "Лимит ставок за период",
			Type:         "int",
		},
		"captcha_bet_activity_ttl": {
			Key:          "captcha_bet_activity_ttl",
			DefaultValue: "180",
			Description:  "Период ставок для лимита (сек)",
			Type:         "int",
		},
		"captcha_user_activity": {
			Key:          "captcha_user_activity",
			DefaultValue: "10",
			Description:  "Лимит действий за период",
			Type:         "int",
		},
		"captcha_user_activity_ttl": {
			Key:          "captcha_user_activity_ttl",
			DefaultValue: "10",
			Description:  "Период действий для лимита (сек)",
			Type:         "int",
		},
		"captcha_bet_points": {
			Key:          "captcha_bet_points",
			DefaultValue: "50",
			Description:  "Лимит баллов",
			Type:         "int",
		},
		"captcha_bet_duplicate_ttl": {
			Key:          "captcha_bet_duplicate_ttl",
			DefaultValue: "1800",
			Description:  "Период дубликатов ставок (сек)",
			Type:         "int",
		},
		"captcha_ttl": {
			Key:          "captcha_ttl",
			DefaultValue: "180",
			Description:  "Время ожидания капчи (мин)",
			Type:         "int",
		},
		"captcha_refresh_count": {
			Key:          "captcha_refresh_count",
			DefaultValue: "3",
			Description:  "Кол-во обновлений",
			Type:         "int",
		},
		"captcha_need_count": {
			Key:          "captcha_need_count",
			DefaultValue: "3",
			Description:  "Кол-во этапов",
			Type:         "int",
		},
		"captcha_wrong_count": {
			Key:          "captcha_wrong_count",
			DefaultValue: "3",
			Description:  "Кол-во неправильнх ответов",
			Type:         "int",
		},
		"captcha_ban_count": {
			Key:          "captcha_ban_count",
			DefaultValue: "3",
			Description:  "Кол-во банов",
			Type:         "int",
		},
		"captcha_ban_short_ttl": {
			Key:          "captcha_ban_short_ttl",
			DefaultValue: "60",
			Description:  "Время бана short (мин)",
			Type:         "int",
		},
		"captcha_ban_long_ttl": {
			Key:          "captcha_ban_long_ttl",
			DefaultValue: "1440",
			Description:  "Время бана long (мин)",
			Type:         "int",
		},
	}
}

// SaveSettings сохраняет все настройки из формы
func (s *ServiceImpl) SaveSettings(settings map[string]string) error {
	// Получаем информацию о настройках по умолчанию для проверки типов
	defaultSettings := s.getDefaultSettings()

	for key, value := range settings {
		info, exists := defaultSettings[key]
		if !exists {
			continue // Пропускаем неизвестные настройки
		}

		// Валидация в зависимости от типа
		switch info.Type {
		case "int":
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return err
			}
		case "float":
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return err
			}
		case "day":
			day, err := strconv.ParseInt(value, 10, 64)
			if err != nil || day < 1 || day > 7 {
				value = "1" // Устанавливаем значение по умолчанию
			}
		case "time":
			_, err := time.Parse("15:04", value)
			if err != nil {
				value = "00:00" // Устанавливаем значение по умолчанию
			}
		}

		// Сохраняем настройку
		if err := s.repo.CreateOrUpdateSetting(key, value, info.DefaultValue, info.Description); err != nil {
			return err
		}
	}

	return nil
}
